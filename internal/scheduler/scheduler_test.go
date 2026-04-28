package scheduler

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/432539/gpt2api/internal/account"
)

func TestDailyLimitExceeded(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.Local)
	acc := &account.Account{
		DailyImageQuota: 100,
		TodayUsedCount:  100,
		TodayUsedDate:   sql.NullTime{Time: truncateDay(now), Valid: true},
	}

	if used := todayUsed(acc, now); used != 100 {
		t.Fatalf("todayUsed() = %d, want 100", used)
	}
	if !isDailyLimitExceeded(acc, todayUsed(acc, now)) {
		t.Fatal("expected daily limit to be exceeded at the configured quota")
	}
}

func TestDailyLimitIgnoresStaleUsage(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.Local)
	yesterday := now.AddDate(0, 0, -1)
	acc := &account.Account{
		DailyImageQuota: 100,
		TodayUsedCount:  100,
		TodayUsedDate:   sql.NullTime{Time: truncateDay(yesterday), Valid: true},
	}

	if used := todayUsed(acc, now); used != 0 {
		t.Fatalf("todayUsed() = %d, want 0 for stale usage date", used)
	}
	if isDailyLimitExceeded(acc, todayUsed(acc, now)) {
		t.Fatal("did not expect stale usage to trip today's daily limit")
	}
}

func TestDailyLimitDisabledByZero(t *testing.T) {
	acc := &account.Account{
		DailyImageQuota: 0,
		TodayUsedCount:  10000,
	}

	if isDailyLimitExceeded(acc, acc.TodayUsedCount) {
		t.Fatal("daily_image_quota=0 should disable the manual daily limit")
	}
}

func TestEffectiveDailyQuotaPrefersManualLimit(t *testing.T) {
	acc := &account.Account{
		DailyImageQuota: 100,
		ImageQuotaTotal: 200,
	}

	if got := effectiveDailyQuota(acc); got != 100 {
		t.Fatalf("effectiveDailyQuota() = %d, want manual limit 100", got)
	}
}

func TestEffectiveDailyQuotaFallsBackToDetectedTotal(t *testing.T) {
	acc := &account.Account{
		DailyImageQuota: 0,
		ImageQuotaTotal: 200,
	}

	if got := effectiveDailyQuota(acc); got != 200 {
		t.Fatalf("effectiveDailyQuota() = %d, want detected total 200", got)
	}
}

func TestImageQuotaZeroIsExhaustedEvenWithoutResetAt(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.Local)
	acc := &account.Account{
		ImageQuotaRemaining: 0,
		ImageQuotaUpdatedAt: sql.NullTime{Time: now, Valid: true},
	}

	if !isImageQuotaExhausted(acc, now) {
		t.Fatal("expected probed zero image quota to be exhausted even without reset_at")
	}
}

func TestImageQuotaUnknownIsNotExhausted(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.Local)
	acc := &account.Account{
		ImageQuotaRemaining: -1,
		ImageQuotaUpdatedAt: sql.NullTime{Time: now, Valid: true},
	}

	if isImageQuotaExhausted(acc, now) {
		t.Fatal("did not expect unknown image quota to be exhausted")
	}
}

func TestNoAvailableErrorIncludesSkipSummary(t *testing.T) {
	err := &noAvailableError{stats: dispatchSkipStats{
		CandidateCount:  2,
		SkippedQuota:    1,
		SkippedLockBusy: 1,
		Samples: []dispatchSkipSample{
			{AccountID: 1, Reason: "lock_busy"},
			{AccountID: 2, Reason: "image_quota_exhausted"},
		},
	}}

	if !errors.Is(err, ErrNoAvailable) {
		t.Fatalf("errors.Is(err, ErrNoAvailable) = false")
	}
	msg := err.Error()
	for _, want := range []string{"candidates=2", "quota_exhausted=1", "lock_busy=1", "acct1:lock_busy", "acct2:image_quota_exhausted"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q does not contain %q", msg, want)
		}
	}
}
