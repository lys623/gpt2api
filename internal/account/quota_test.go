package account

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestFlexibleStringListAcceptsStringsAndObjects(t *testing.T) {
	var got flexibleStringList
	data := []byte(`[
		"image_gen",
		{"feature_name":"voice_mode"},
		{"name":"canvas"},
		{"code":"search"},
		{"unexpected":true}
	]`)
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	want := flexibleStringList{
		"image_gen",
		"voice_mode",
		"canvas",
		"search",
		`{"unexpected":true}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestFlexibleStringListAcceptsSingleObject(t *testing.T) {
	var got flexibleStringList
	if err := json.Unmarshal([]byte(`{"feature":"image_gen"}`), &got); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	want := flexibleStringList{"image_gen"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestQuotaProbeStatusRestoresThrottledWhenQuotaReturns(t *testing.T) {
	acc := &Account{Status: StatusThrottled}

	status, cooldown, ok := quotaProbeStatus(acc, 12, time.Time{})
	if !ok {
		t.Fatal("expected status update")
	}
	if status != StatusHealthy {
		t.Fatalf("status = %q, want %q", status, StatusHealthy)
	}
	if cooldown != nil {
		t.Fatalf("cooldown = %v, want nil", cooldown)
	}
}

func TestQuotaProbeStatusMarksHealthyZeroAsThrottled(t *testing.T) {
	resetAt := time.Date(2026, 5, 11, 9, 30, 0, 0, time.UTC)
	acc := &Account{Status: StatusHealthy}

	status, cooldown, ok := quotaProbeStatus(acc, 0, resetAt)
	if !ok {
		t.Fatal("expected status update")
	}
	if status != StatusThrottled {
		t.Fatalf("status = %q, want %q", status, StatusThrottled)
	}
	if cooldown == nil || !cooldown.Equal(resetAt) {
		t.Fatalf("cooldown = %v, want %v", cooldown, resetAt)
	}
}

func TestQuotaProbeStatusUpdatesThrottledCooldownAfterZeroProbe(t *testing.T) {
	oldReset := time.Date(2026, 5, 10, 23, 39, 0, 0, time.UTC)
	newReset := time.Date(2026, 5, 11, 23, 39, 0, 0, time.UTC)
	acc := &Account{
		Status:        StatusThrottled,
		CooldownUntil: sql.NullTime{Time: oldReset, Valid: true},
	}

	status, cooldown, ok := quotaProbeStatus(acc, 0, newReset)
	if !ok {
		t.Fatal("expected status update")
	}
	if status != StatusThrottled {
		t.Fatalf("status = %q, want %q", status, StatusThrottled)
	}
	if cooldown == nil || !cooldown.Equal(newReset) {
		t.Fatalf("cooldown = %v, want %v", cooldown, newReset)
	}
}

func TestQuotaProbeStatusDoesNotReviveDeadAccount(t *testing.T) {
	acc := &Account{Status: StatusDead}

	if _, _, ok := quotaProbeStatus(acc, 0, time.Now()); ok {
		t.Fatal("did not expect status update for dead account")
	}
}
