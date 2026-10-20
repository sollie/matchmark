package service

import (
	"fmt"
	"testing"
	"time"
)

func TestUsernameThrottleEvictsOldestWhenFull(t *testing.T) {
	a := NewAuth(nil)
	base := time.Now()
	for i := 0; i < usernameMaxTracked; i++ {
		a.recordUsernameFailure(fmt.Sprintf("user%d", i), base.Add(time.Duration(i)*time.Millisecond))
	}
	now := base.Add(usernameMaxTracked * time.Millisecond)
	if delay := a.recordUsernameFailure("newcomer", now); delay != 0 {
		t.Fatalf("untracked username delayed by %v", delay)
	}
	if len(a.userFail) != usernameMaxTracked {
		t.Fatalf("tracked = %d", len(a.userFail))
	}
	if _, ok := a.userFail["user0"]; ok {
		t.Fatal("oldest entry was not evicted")
	}
	if _, ok := a.userFail["newcomer"]; !ok {
		t.Fatal("newcomer not tracked")
	}
}
