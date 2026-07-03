// Package scheduler 负责 chatgpt.com 账号的并发安全调度。
//
// 核心规则(参考 RISK_AND_SAAS.md):
//  1. 一号一锁:同账号同时只允许 1 个请求占用(Redis SETNX)。
//  2. 最小间隔:同账号相邻请求 >= min_interval_sec。
//  3. 每日配额:daily_image_quota 为人工硬熔断;daily_usage_ratio 仅用于提前降优先级。
//  4. 状态机:healthy -> warned -> throttled -> suspicious -> dead,冷却过期自动恢复。
//  5. 选择策略:status=healthy + cooldown 到期 + last_used_at 最早的优先。
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/432539/gpt2api/internal/account"
	"github.com/432539/gpt2api/internal/config"
	"github.com/432539/gpt2api/internal/proxy"
	"github.com/432539/gpt2api/pkg/lock"
	"github.com/432539/gpt2api/pkg/logger"

	"go.uber.org/zap"
)

// ErrNoAvailable 没有任何账号可用。
var ErrNoAvailable = errors.New("scheduler: no available account")

type noAvailableError struct {
	stats dispatchSkipStats
}

func (e *noAvailableError) Error() string {
	if e == nil {
		return ErrNoAvailable.Error()
	}
	if summary := e.stats.summary(); summary != "" {
		return ErrNoAvailable.Error() + ": " + summary
	}
	return ErrNoAvailable.Error()
}
func (e *noAvailableError) Unwrap() error { return ErrNoAvailable }

type dispatchSkipStats struct {
	CandidateCount     int
	SkippedInterval    int
	SkippedQuota       int
	SkippedDailyLimit  int
	DeprioritizedQuota int
	SkippedLockBusy    int
	SkippedLockErr     int
	Samples            []dispatchSkipSample
}

type dispatchSkipSample struct {
	AccountID       uint64 `json:"account_id"`
	Status          string `json:"status,omitempty"`
	Reason          string `json:"reason"`
	LastUsedAgoSec  int64  `json:"last_used_ago_sec,omitempty"`
	MinIntervalSec  int    `json:"min_interval_sec,omitempty"`
	UsedToday       int    `json:"used_today,omitempty"`
	QuotaRemaining  int    `json:"quota_remaining,omitempty"`
	QuotaResetAt    string `json:"quota_reset_at,omitempty"`
	DailyQuota      int    `json:"daily_quota,omitempty"`
	ConfiguredQuota int    `json:"configured_quota,omitempty"`
	ImageQuotaTotal int    `json:"image_quota_total,omitempty"`
	DailyLimit      int    `json:"daily_limit,omitempty"`
	DailyUsageRatio string `json:"daily_usage_ratio,omitempty"`
	Error           string `json:"error,omitempty"`
}

func (s *dispatchSkipStats) addSample(sample dispatchSkipSample) {
	const maxSamples = 12
	if len(s.Samples) >= maxSamples {
		return
	}
	s.Samples = append(s.Samples, sample)
}

func (s dispatchSkipStats) summary() string {
	parts := []string{fmt.Sprintf("candidates=%d", s.CandidateCount)}
	if s.SkippedInterval > 0 {
		parts = append(parts, fmt.Sprintf("min_interval=%d", s.SkippedInterval))
	}
	if s.SkippedQuota > 0 {
		parts = append(parts, fmt.Sprintf("quota_exhausted=%d", s.SkippedQuota))
	}
	if s.SkippedDailyLimit > 0 {
		parts = append(parts, fmt.Sprintf("daily_limit=%d", s.SkippedDailyLimit))
	}
	if s.DeprioritizedQuota > 0 {
		parts = append(parts, fmt.Sprintf("deprioritized=%d", s.DeprioritizedQuota))
	}
	if s.SkippedLockBusy > 0 {
		parts = append(parts, fmt.Sprintf("lock_busy=%d", s.SkippedLockBusy))
	}
	if s.SkippedLockErr > 0 {
		parts = append(parts, fmt.Sprintf("lock_error=%d", s.SkippedLockErr))
	}
	if len(s.Samples) > 0 {
		samples := make([]string, 0, len(s.Samples))
		for _, sample := range s.Samples {
			samples = append(samples, fmt.Sprintf("acct%d:%s", sample.AccountID, sample.Reason))
		}
		parts = append(parts, "samples="+strings.Join(samples, ","))
	}
	return strings.Join(parts, " ")
}

// Lease 代表一次账号占用的租约。
type Lease struct {
	Account     *account.Account
	AuthToken   string // 已解密
	ProxyURL    string // 已带密码
	ProxyID     uint64
	DeviceID    string
	SessionID   string // oai_session_id(按账号稳定)
	lockKey     string
	lockToken   string
	releaseFunc func(context.Context) error
}

// Release 释放锁并更新账号 last_used_at / today_used。
func (l *Lease) Release(ctx context.Context) error {
	if l.releaseFunc != nil {
		return l.releaseFunc(ctx)
	}
	return nil
}

// RuntimeParams 调度器运行期可热更的参数。
//   - 由外部 settings.Service 提供回调,每次读都取最新值;
//   - 回调未注入时回退到 cfg 的静态值。
type RuntimeParams struct {
	// 为 nil 时 Scheduler 使用 cfg 里的静态值。
	DailyUsageRatio func() float64
	Cooldown429Sec  func() int
	WarnedPauseHrs  func() int
	// QueueWaitSec 拿不到空闲账号时最长排队等待秒数,≤0 表示不排队(老语义)。
	QueueWaitSec func() int
}

// Scheduler 账号调度器。
type Scheduler struct {
	accSvc   *account.Service
	proxySvc *proxy.Service
	lock     *lock.RedisLock
	cfg      config.SchedulerConfig
	rt       RuntimeParams
}

func New(
	accSvc *account.Service,
	proxySvc *proxy.Service,
	rl *lock.RedisLock,
	cfg config.SchedulerConfig,
) *Scheduler {
	if cfg.LockTTLSec <= 0 {
		cfg.LockTTLSec = 180
	}
	if cfg.MinIntervalSec <= 0 {
		cfg.MinIntervalSec = 5
	}
	if cfg.DailyUsageRatio <= 0 {
		cfg.DailyUsageRatio = 0.8
	}
	if cfg.Cooldown429Sec <= 0 {
		cfg.Cooldown429Sec = 300
	}
	return &Scheduler{accSvc: accSvc, proxySvc: proxySvc, lock: rl, cfg: cfg}
}

// SetRuntime 注入运行期可热更的参数。建议在 main 里一次性设置:
//
//	sched.SetRuntime(scheduler.RuntimeParams{
//	    DailyUsageRatio: settingsSvc.DailyUsageRatio,
//	    Cooldown429Sec:  settingsSvc.Cooldown429Sec,
//	    WarnedPauseHrs:  settingsSvc.WarnedPauseHours,
//	})
func (s *Scheduler) SetRuntime(p RuntimeParams) { s.rt = p }

// 下面三个 getter 用于内部调用,保证"有回调用回调,否则用 cfg"。
func (s *Scheduler) dailyUsageRatio() float64 {
	if s.rt.DailyUsageRatio != nil {
		if v := s.rt.DailyUsageRatio(); v > 0 && v <= 1 {
			return v
		}
	}
	return s.cfg.DailyUsageRatio
}
func (s *Scheduler) cooldown429() time.Duration {
	if s.rt.Cooldown429Sec != nil {
		if v := s.rt.Cooldown429Sec(); v > 0 {
			return time.Duration(v) * time.Second
		}
	}
	return time.Duration(s.cfg.Cooldown429Sec) * time.Second
}
func (s *Scheduler) warnedPause() time.Duration {
	if s.rt.WarnedPauseHrs != nil {
		if v := s.rt.WarnedPauseHrs(); v > 0 {
			return time.Duration(v) * time.Hour
		}
	}
	return time.Duration(s.cfg.WarnedPauseHours) * time.Hour
}

// queueWait 拿不到账号时的最长排队等待时间。
// 返回 0 表示关闭排队(立即返回 ErrNoAvailable)。
func (s *Scheduler) queueWait() time.Duration {
	if s.rt.QueueWaitSec != nil {
		if v := s.rt.QueueWaitSec(); v >= 0 {
			return time.Duration(v) * time.Second
		}
	}
	return 120 * time.Second
}

// Dispatch 为本次请求挑选一个账号并加锁。调用方必须 defer lease.Release(ctx)。
//
// 语义(一号一任务 + 排队):
//   - 同账号同时只允许 1 个请求持有 Redis 锁(acct:lock:{id},SETNX+TTL)。
//   - 扫一遍所有 candidate 都被锁住 / 不满足 min_interval / 日配额时,
//     不立即返回失败,而是按指数退避轮询重试,直到拿到锁或超过 queueWait。
//   - queueWait=0 时退化为老语义(扫一次,失败即返回 ErrNoAvailable)。
func (s *Scheduler) Dispatch(ctx context.Context, modelType string) (*Lease, error) {
	deadline := time.Now().Add(s.queueWait())

	const (
		minBackoff = 200 * time.Millisecond
		maxBackoff = 2 * time.Second
	)
	backoff := minBackoff

	attempt := 0
	start := time.Now()
	var lastNoAvailable *noAvailableError

	for {
		attempt++
		lease, err := s.tryDispatchOnce(ctx, modelType)
		if err == nil {
			if attempt > 1 {
				logger.L().Info("scheduler queued dispatch ok",
					zap.Int("attempt", attempt),
					zap.Duration("waited", time.Since(start)),
					zap.Uint64("account_id", lease.Account.ID))
			}
			return lease, nil
		}
		if !errors.Is(err, ErrNoAvailable) {
			return nil, err
		}
		var noAvail *noAvailableError
		if errors.As(err, &noAvail) {
			lastNoAvailable = noAvail
		}

		// 所有候选都忙或不就绪:排队等待。
		if !time.Now().Before(deadline) {
			s.logNoAvailable(ctx, modelType, attempt, start, lastNoAvailable)
			return nil, noAvailableOrDefault(lastNoAvailable)
		}
		wait := backoff
		if remain := time.Until(deadline); remain < wait {
			wait = remain
		}
		if wait <= 0 {
			s.logNoAvailable(ctx, modelType, attempt, start, lastNoAvailable)
			return nil, noAvailableOrDefault(lastNoAvailable)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
		// 指数退避(×1.5)
		backoff += backoff / 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func noAvailableOrDefault(err *noAvailableError) error {
	if err != nil {
		return err
	}
	return ErrNoAvailable
}

// tryDispatchOnce 扫一遍 candidate,尝试为其中一个加锁;
// 全部 candidate 都被锁 / 不满足 min_interval / 明确无剩余额度时返回 ErrNoAvailable。
func (s *Scheduler) tryDispatchOnce(ctx context.Context, modelType string) (*Lease, error) {
	limit := 30
	dao := s.accSvc.DAO()
	candidates, err := dao.ListDispatchable(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("scheduler list: %w", err)
	}
	stats := dispatchSkipStats{CandidateCount: len(candidates)}
	if len(candidates) == 0 {
		return nil, &noAvailableError{stats: stats}
	}

	now := time.Now()
	minInterval := time.Duration(s.cfg.MinIntervalSec) * time.Second
	dailyRatio := s.dailyUsageRatio()
	preferred := make([]*account.Account, 0, len(candidates))
	deprioritized := make([]*account.Account, 0)

	for _, acc := range candidates {
		if acc.LastUsedAt.Valid && now.Sub(acc.LastUsedAt.Time) < minInterval {
			stats.SkippedInterval++
			stats.addSample(dispatchSkipSample{
				AccountID:      acc.ID,
				Status:         acc.Status,
				Reason:         "min_interval",
				LastUsedAgoSec: int64(now.Sub(acc.LastUsedAt.Time) / time.Second),
				MinIntervalSec: s.cfg.MinIntervalSec,
			})
			continue
		}
		if isImageQuotaExhausted(acc, now) {
			stats.SkippedQuota++
			stats.addSample(dispatchSkipSample{
				AccountID:      acc.ID,
				Status:         acc.Status,
				Reason:         "image_quota_exhausted",
				QuotaRemaining: acc.ImageQuotaRemaining,
				QuotaResetAt:   acc.ImageQuotaResetAt.Time.Format(time.RFC3339),
			})
			continue
		}
		usedToday := todayUsed(acc, now)
		if isDailyLimitExceeded(acc, usedToday) {
			stats.SkippedDailyLimit++
			stats.addSample(dispatchSkipSample{
				AccountID:       acc.ID,
				Status:          acc.Status,
				Reason:          "daily_quota_exhausted",
				UsedToday:       usedToday,
				DailyQuota:      acc.DailyImageQuota,
				ConfiguredQuota: acc.DailyImageQuota,
				ImageQuotaTotal: acc.ImageQuotaTotal,
			})
			continue
		}
		dailyQuota := effectiveDailyQuota(acc)
		if dailyQuota > 0 {
			softLimit := int(float64(dailyQuota) * dailyRatio)
			if softLimit > 0 && usedToday >= softLimit {
				stats.DeprioritizedQuota++
				stats.addSample(dispatchSkipSample{
					AccountID:       acc.ID,
					Status:          acc.Status,
					Reason:          "daily_quota_deprioritized",
					UsedToday:       usedToday,
					DailyQuota:      dailyQuota,
					ConfiguredQuota: acc.DailyImageQuota,
					ImageQuotaTotal: acc.ImageQuotaTotal,
					DailyLimit:      softLimit,
					DailyUsageRatio: fmt.Sprintf("%.4f", dailyRatio),
				})
				deprioritized = append(deprioritized, acc)
				continue
			}
		}
		preferred = append(preferred, acc)
	}

	if lease, ok := s.tryLockAny(ctx, preferred, &stats); ok {
		return lease, nil
	}
	if lease, ok := s.tryLockAny(ctx, deprioritized, &stats); ok {
		logger.L().Info("scheduler dispatch over daily usage ratio",
			zap.Uint64("account_id", lease.Account.ID),
			zap.Int("used_today", lease.Account.TodayUsedCount),
			zap.Int("daily_quota", effectiveDailyQuota(lease.Account)),
			zap.Float64("daily_usage_ratio", dailyRatio))
		return lease, nil
	}
	return nil, &noAvailableError{stats: stats}
}

func (s *Scheduler) tryLockAny(ctx context.Context, accounts []*account.Account, stats *dispatchSkipStats) (*Lease, bool) {
	for _, acc := range accounts {
		lease, err := s.tryLock(ctx, acc)
		if err == nil {
			return lease, true
		}
		if errors.Is(err, lock.ErrNotAcquired) {
			// 被别的请求占用,下一个候选
			stats.SkippedLockBusy++
			stats.addSample(dispatchSkipSample{
				AccountID: acc.ID,
				Status:    acc.Status,
				Reason:    "lock_busy",
			})
			continue
		}
		stats.SkippedLockErr++
		stats.addSample(dispatchSkipSample{
			AccountID: acc.ID,
			Status:    acc.Status,
			Reason:    "lock_error",
			Error:     err.Error(),
		})
		logger.L().Warn("scheduler tryLock error",
			zap.Uint64("account_id", acc.ID), zap.Error(err))
	}
	return nil, false
}

func (s *Scheduler) logNoAvailable(ctx context.Context, modelType string, attempt int, start time.Time, last *noAvailableError) {
	fields := []zap.Field{
		zap.String("model_type", modelType),
		zap.Int("attempt", attempt),
		zap.Duration("waited", time.Since(start)),
		zap.Int("queue_wait_sec", int(s.queueWait()/time.Second)),
		zap.Int("min_interval_sec", s.cfg.MinIntervalSec),
		zap.Float64("daily_usage_ratio", s.dailyUsageRatio()),
		zap.Int("lock_ttl_sec", s.cfg.LockTTLSec),
	}
	if last != nil {
		fields = append(fields,
			zap.Int("candidate_count", last.stats.CandidateCount),
			zap.Int("skipped_min_interval", last.stats.SkippedInterval),
			zap.Int("skipped_quota_exhausted", last.stats.SkippedQuota),
			zap.Int("skipped_daily_limit", last.stats.SkippedDailyLimit),
			zap.Int("deprioritized_daily_quota", last.stats.DeprioritizedQuota),
			zap.Int("skipped_lock_busy", last.stats.SkippedLockBusy),
			zap.Int("skipped_lock_error", last.stats.SkippedLockErr),
			zap.Any("skip_samples", last.stats.Samples),
		)
	}
	if dbStats, err := s.accSvc.DAO().DispatchableDiagnostics(ctx); err == nil && dbStats != nil {
		fields = append(fields,
			zap.Int("db_active_accounts", dbStats.ActiveAccounts),
			zap.Int("db_status_eligible", dbStats.StatusEligible),
			zap.Int("db_status_blocked", dbStats.StatusBlocked),
			zap.Int("db_cooldown_blocked", dbStats.CooldownBlocked),
			zap.Int("db_token_expired", dbStats.TokenExpired),
			zap.Int("db_dispatchable", dbStats.Dispatchable),
		)
	} else if err != nil {
		fields = append(fields, zap.Error(err))
	}
	logger.L().Warn("scheduler no available account", fields...)
}

func (s *Scheduler) tryLock(ctx context.Context, acc *account.Account) (*Lease, error) {
	key := fmt.Sprintf("acct:lock:%d", acc.ID)
	token := uuid.NewString()
	ttl := time.Duration(s.cfg.LockTTLSec) * time.Second
	if err := s.lock.Acquire(ctx, key, token, ttl); err != nil {
		return nil, err
	}

	authToken, err := s.accSvc.DecryptAuthToken(acc)
	if err != nil {
		_ = s.lock.Release(ctx, key, token)
		return nil, fmt.Errorf("decrypt auth_token: %w", err)
	}

	// 首次使用时为账号补发一个持久化的 oai_device_id(导入时常为空)。
	// chatgpt.com 要求请求头带 oai-device-id,等同于浏览器首访拿到的 oai-did cookie;
	// 一次生成后持久化,账号绑定的"设备身份"保持稳定,避免每次换 id 触发风控。
	deviceID := acc.OAIDeviceID
	if deviceID == "" {
		gen := uuid.NewString()
		if fixed, err := s.accSvc.DAO().EnsureDeviceID(ctx, acc.ID, gen); err == nil && fixed != "" {
			deviceID = fixed
			acc.OAIDeviceID = fixed
		} else {
			deviceID = gen
		}
	}

	// oai_session_id:真实浏览器是"每打开页面生成一次"。为了保持账号行为稳定
	// (风控倾向于把频繁变换 session_id 的账号识别为脚本),我们按账号持久化,
	// 与 device_id 同策略。
	sessionID := acc.OAISessionID
	if sessionID == "" {
		gen := uuid.NewString()
		if fixed, err := s.accSvc.DAO().EnsureSessionID(ctx, acc.ID, gen); err == nil && fixed != "" {
			sessionID = fixed
			acc.OAISessionID = fixed
		} else {
			sessionID = gen
		}
	}

	var proxyURL string
	var proxyID uint64
	if b, _ := s.accSvc.GetBinding(ctx, acc.ID); b != nil {
		p, err := s.proxySvc.Get(ctx, b.ProxyID)
		if err == nil && p != nil && p.Enabled {
			if u, err := s.proxySvc.BuildURL(p); err == nil {
				proxyURL = u
				proxyID = p.ID
			}
		}
	}

	if acc.Status == account.StatusThrottled {
		if err := s.accSvc.DAO().SetStatus(ctx, acc.ID, account.StatusHealthy, nil); err != nil {
			logger.L().Warn("scheduler restore throttled account failed",
				zap.Uint64("account_id", acc.ID), zap.Error(err))
		} else {
			acc.Status = account.StatusHealthy
		}
	}

	accCopy := acc
	lease := &Lease{
		Account:   accCopy,
		AuthToken: authToken,
		ProxyURL:  proxyURL,
		ProxyID:   proxyID,
		DeviceID:  deviceID,
		SessionID: sessionID,
		lockKey:   key,
		lockToken: token,
	}
	lease.releaseFunc = func(c context.Context) error {
		today := truncateDay(time.Now())
		_ = s.accSvc.DAO().MarkUsed(c, accCopy.ID, today)
		return s.lock.Release(c, key, token)
	}
	return lease, nil
}

// MarkRateLimited 上游 429:标记账号冷却并降级状态。
func (s *Scheduler) MarkRateLimited(ctx context.Context, accountID uint64) {
	cooldown := time.Now().Add(s.cooldown429())
	_ = s.accSvc.DAO().SetStatus(ctx, accountID, account.StatusThrottled, &cooldown)
}

// MarkWarned 上游返回 suspicious 横幅时降级。
func (s *Scheduler) MarkWarned(ctx context.Context, accountID uint64) {
	pause := time.Now().Add(s.warnedPause())
	_ = s.accSvc.DAO().SetStatus(ctx, accountID, account.StatusWarned, &pause)
}

// MarkDead 账号彻底不可用(403/token 失效)。
func (s *Scheduler) MarkDead(ctx context.Context, accountID uint64) {
	_ = s.accSvc.DAO().SetStatus(ctx, accountID, account.StatusDead, nil)
}

// MarkPaused 账号暂停(auth_required / token 过期),不再自动调度,需管理员人工恢复。
func (s *Scheduler) MarkPaused(ctx context.Context, accountID uint64) {
	_ = s.accSvc.DAO().SetStatus(ctx, accountID, account.StatusPaused, nil)
}

// RestoreHealthy 调度成功后回归健康(仅对 throttled 且冷却到期有效,
// 简单起见此处不强检查,由管理员按需恢复)。
func (s *Scheduler) RestoreHealthy(ctx context.Context, accountID uint64) {
	_ = s.accSvc.DAO().SetStatus(ctx, accountID, account.StatusHealthy, nil)
}

// ------ helpers ------

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func todayUsed(acc *account.Account, now time.Time) int {
	if acc == nil || !acc.TodayUsedDate.Valid || !sameDay(acc.TodayUsedDate.Time, now) {
		return 0
	}
	return acc.TodayUsedCount
}

func isDailyLimitExceeded(acc *account.Account, usedToday int) bool {
	if acc == nil || acc.DailyImageQuota <= 0 {
		return false
	}
	return usedToday >= acc.DailyImageQuota
}

func effectiveDailyQuota(acc *account.Account) int {
	if acc == nil {
		return 0
	}
	if acc.DailyImageQuota > 0 {
		return acc.DailyImageQuota
	}
	return acc.ImageQuotaTotal
}

func isImageQuotaExhausted(acc *account.Account, _ time.Time) bool {
	if acc == nil || !acc.ImageQuotaUpdatedAt.Valid || acc.ImageQuotaRemaining > 0 {
		return false
	}
	// Once the quota probe has observed zero, do not dispatch this account again
	// until a later probe records a positive remaining value. ListNeedProbeQuota
	// already prioritizes zero-quota accounts whose reset time is absent or due,
	// so dispatching stale zeroes only burns user requests on known-exhausted
	// accounts.
	return acc.ImageQuotaRemaining == 0
}
