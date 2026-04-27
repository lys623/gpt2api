package scheduler

import (
	"database/sql"
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
