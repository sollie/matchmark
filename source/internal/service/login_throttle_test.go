package service

import (
	"fmt"
	"testing"
	"time"
)

func TestThrottleBlocksClientAfterMaxFailuresUntilWindowPasses(t *testing.T) {
	th := NewLoginThrottle()
	now := time.Now()
	for i := 0; i < loginMaxFailures; i++ {
		if !th.Allowed("c", now) {
			t.Fatalf("blocked after %d failures", i)
		}
		th.FailClient("c", now)
	}
	if th.Allowed("c", now) {
		t.Fatal("not blocked at max failures")
	}
	if !th.Allowed("other", now) {
		t.Fatal("other client blocked")
	}
	if !th.Allowed("c", now.Add(loginWindow)) {
		t.Fatal("still blocked after window")
	}
}

func TestThrottleUsernameBackoffGrowsCapsAndResets(t *testing.T) {
	th := NewLoginThrottle()
	now := time.Now()
	var delays []time.Duration
	for i := 0; i < 12; i++ {
		d, _ := th.Fail("c", "bob", now)
		delays = append(delays, d)
	}
	for i := 0; i < usernameFreeFailures; i++ {
		if delays[i] != 0 {
			t.Fatalf("free failure %d delayed %v", i, delays[i])
		}
	}
	if delays[3] != usernameBaseDelay || delays[4] != 2*usernameBaseDelay {
		t.Fatalf("delays = %v", delays)
	}
	if delays[11] != usernameMaxDelay {
		t.Fatalf("not capped: %v", delays[11])
	}
	th.Success("bob")
	if d, _ := th.Fail("c", "bob", now); d != 0 {
		t.Fatalf("delay after success = %v", d)
	}
}

func TestThrottleUsernameBackoffExpiresAfterWindow(t *testing.T) {
	th := NewLoginThrottle()
	now := time.Now()
	for i := 0; i < 6; i++ {
		th.Fail("c", "bob", now)
	}
	if d, _ := th.Fail("c", "bob", now.Add(usernameWindow)); d != 0 {
		t.Fatalf("delay after window = %v", d)
	}
}

func TestThrottleUsernameEvictsOldestWhenFull(t *testing.T) {
	th := NewLoginThrottle()
	base := time.Now()
	for i := 0; i < usernameMaxTracked; i++ {
		th.Fail("c", fmt.Sprintf("user%d", i), base.Add(time.Duration(i)*time.Millisecond))
	}
	now := base.Add(usernameMaxTracked * time.Millisecond)
	if d, _ := th.Fail("c", "newcomer", now); d != 0 {
		t.Fatalf("untracked username delayed by %v", d)
	}
	if len(th.usernames) != usernameMaxTracked {
		t.Fatalf("tracked = %d", len(th.usernames))
	}
	if _, ok := th.usernames["user0"]; ok {
		t.Fatal("oldest entry was not evicted")
	}
	if _, ok := th.usernames["newcomer"]; !ok {
		t.Fatal("newcomer not tracked")
	}
}

func TestThrottleAuditDeduplicatesPerWindow(t *testing.T) {
	th := NewLoginThrottle()
	now := time.Now()
	if _, audit := th.Fail("c", "bob", now); !audit {
		t.Fatal("first failure not audited")
	}
	if _, audit := th.Fail("c", "BOB", now.Add(time.Minute)); audit {
		t.Fatal("duplicate audited")
	}
	if _, audit := th.Fail("c", "alice", now); !audit {
		t.Fatal("other username not audited")
	}
	if !th.ShouldAudit("c", "bob", "throttled", now) {
		t.Fatal("throttled event not audited")
	}
	if th.ShouldAudit("c", "bob", "throttled", now.Add(time.Minute)) {
		t.Fatal("duplicate throttled event audited")
	}
	if _, audit := th.Fail("c", "bob", now.Add(loginAuditWindow)); !audit {
		t.Fatal("not audited after window")
	}
}

func TestThrottleAuditEvictsOldestWhenFull(t *testing.T) {
	th := NewLoginThrottle()
	base := time.Now()
	for i := 0; i < loginAuditMaxTracked; i++ {
		th.ShouldAudit("c", fmt.Sprintf("u%d", i), "", base.Add(time.Duration(i)*time.Millisecond))
	}
	th.ShouldAudit("c", "newcomer", "", base.Add(time.Minute))
	if len(th.audited) != loginAuditMaxTracked {
		t.Fatalf("tracked = %d", len(th.audited))
	}
	if _, ok := th.audited["|c|u0"]; ok {
		t.Fatal("oldest not evicted")
	}
}
