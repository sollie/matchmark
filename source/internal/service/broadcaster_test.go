package service_test

import (
	"testing"
	"time"

	"github.com/sollie/matchmark/internal/service"
)

const testWindow = 20 * time.Millisecond

func expectPing(t *testing.T, sub *service.Subscription) {
	t.Helper()
	select {
	case <-sub.C:
	case <-time.After(time.Second):
		t.Fatal("expected a ping")
	}
}

func expectQuiet(t *testing.T, sub *service.Subscription, wait time.Duration) {
	t.Helper()
	select {
	case <-sub.C:
		t.Fatal("unexpected ping")
	case <-time.After(wait):
	}
}

func TestBroadcasterCoalescesBurstIntoOnePing(t *testing.T) {
	b := service.NewBroadcaster(testWindow)
	sub := b.Subscribe(1)
	defer sub.Close()
	for i := 0; i < 10; i++ {
		b.Publish(1)
	}
	expectPing(t, sub)
	expectQuiet(t, sub, 3*testWindow)
}

func TestBroadcasterPingsOnlySubscribersOfThatCompetition(t *testing.T) {
	b := service.NewBroadcaster(testWindow)
	one := b.Subscribe(1)
	two := b.Subscribe(2)
	defer one.Close()
	defer two.Close()
	b.Publish(1)
	expectPing(t, one)
	expectQuiet(t, two, 3*testWindow)
}

func TestBroadcasterPublishAllReachesEveryCompetition(t *testing.T) {
	b := service.NewBroadcaster(testWindow)
	one := b.Subscribe(1)
	two := b.Subscribe(2)
	defer one.Close()
	defer two.Close()
	b.PublishAll()
	expectPing(t, one)
	expectPing(t, two)
}

func TestBroadcasterClosedSubscriptionGetsNoPing(t *testing.T) {
	b := service.NewBroadcaster(testWindow)
	sub := b.Subscribe(1)
	sub.Close()
	sub.Close()
	b.Publish(1)
	expectQuiet(t, sub, 3*testWindow)
}

func TestBroadcasterSlowSubscriberDoesNotBlockOthers(t *testing.T) {
	b := service.NewBroadcaster(testWindow)
	slow := b.Subscribe(1)
	fast := b.Subscribe(1)
	defer slow.Close()
	defer fast.Close()
	b.Publish(1)
	expectPing(t, fast)
	b.Publish(1)
	expectPing(t, fast)
}

func TestBroadcasterVersionAdvancesPerPublish(t *testing.T) {
	b := service.NewBroadcaster(testWindow)
	before := b.Version(1)
	b.Publish(1)
	if b.Version(1) == before {
		t.Fatal("version did not advance")
	}
}
