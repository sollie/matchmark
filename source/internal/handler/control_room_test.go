package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/sollie/matchmark/internal/apitest"
)

type controlRoom struct {
	CurrentSegment int  `json:"current_segment"`
	CanStartNext   bool `json:"can_start_next"`
	CanAcceptAll   bool `json:"can_accept_all"`
	Lanes          []controlLane
}

type controlLane struct {
	Lane        int
	EntryID     int
	ShooterName string
	ScorerName  string
	Status      string
	Disputed    bool
	Disputes    int
	Paired      bool
	Connected   bool
}

func (f *scoringFixture) controlRoom(t *testing.T) controlRoom {
	t.Helper()
	status, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d/control-room", f.comp), nil)
	if status != http.StatusOK {
		t.Fatalf("control room: %d %v", status, out)
	}
	room := controlRoom{
		CurrentSegment: int(out["current_segment"].(float64)),
		CanStartNext:   out["can_start_next"].(bool),
		CanAcceptAll:   out["can_accept_all"].(bool),
	}
	for _, l := range out["lanes"].([]any) {
		m := l.(map[string]any)
		room.Lanes = append(room.Lanes, controlLane{
			Lane:        int(m["lane"].(float64)),
			EntryID:     int(m["entry_id"].(float64)),
			ShooterName: m["shooter_name"].(string),
			ScorerName:  m["scorer_name"].(string),
			Status:      m["status"].(string),
			Disputed:    m["disputed"].(bool),
			Disputes:    int(m["disputes"].(float64)),
			Paired:      m["paired"].(bool),
			Connected:   m["connected"].(bool),
		})
	}
	return room
}

func (r controlRoom) lane(t *testing.T, lane int) controlLane {
	t.Helper()
	for _, l := range r.Lanes {
		if l.Lane == lane {
			return l
		}
	}
	t.Fatalf("lane %d missing from control room", lane)
	return controlLane{}
}

func (f *scoringFixture) eventually(t *testing.T, what string, ok func(controlRoom) bool) controlRoom {
	t.Helper()
	deadline := time.Now().Add(eventWait)
	for {
		room := f.controlRoom(t)
		if ok(room) {
			return room
		}
		if time.Now().After(deadline) {
			t.Fatalf("control room never showed %s: %+v", what, room.Lanes)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestControlRoomListsEveryMannedLaneWithScorerShooterAndStatus(t *testing.T) {
	f := newScoringFixture(t, 1, 2, 3)
	f.deviceLock(t, 1, 1)

	room := f.controlRoom(t)
	if room.CurrentSegment != 1 || len(room.Lanes) != 3 {
		t.Fatalf("room %+v", room)
	}
	locked := room.lane(t, 2)
	if locked.Status != "awaiting_acceptance" || locked.ShooterName != "A SHOOTER2" || locked.ScorerName != "A SHOOTER1" || locked.Paired {
		t.Fatalf("lane 2 %+v", locked)
	}
	open := room.lane(t, 3)
	if open.Status != "open" || open.ScorerName != "A SHOOTER2" || open.Paired {
		t.Fatalf("lane 3 %+v", open)
	}
	if !room.lane(t, 1).Paired {
		t.Fatalf("lane 1 should be paired: %+v", room.lane(t, 1))
	}
}

func TestControlRoomCountsDisputesPerLane(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.deviceLock(t, 1, 1)
	if status, _ := f.deviceAction(f.pair(t, 2), 1, "dispute"); status != http.StatusOK {
		t.Fatalf("dispute: %d", status)
	}
	room := f.controlRoom(t)
	lane := room.lane(t, 2)
	if lane.Status != "open" || !lane.Disputed || lane.Disputes != 1 {
		t.Fatalf("lane 2 %+v", lane)
	}
	if room.lane(t, 1).Disputes != 0 {
		t.Fatalf("lane 1 %+v", room.lane(t, 1))
	}
}

func TestControlRoomDisputeCountIsCumulativeAcrossSegments(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.deviceLock(t, 1, 1)
	f.deviceAction(f.pair(t, 2), 1, "dispute")
	f.acceptAll(t, 1, 1, 2)
	if status, out := f.startNext(f.comp); status != http.StatusOK {
		t.Fatalf("start next: %d %v", status, out)
	}
	if got := f.controlRoom(t).lane(t, 2).Disputes; got != 1 {
		t.Fatalf("disputes after the next segment started: %d", got)
	}
}

func TestAdminLockIsNotCountedAsAScorer(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.acceptAll(t, 1, 1, 2)
	events, counts := f.history(t, "")
	if len(events) != 2 || len(counts) != 0 {
		t.Fatalf("events %+v counts %+v", events, counts)
	}
	for _, ev := range events {
		if ev.ScorerName != "" {
			t.Fatalf("event %+v", ev)
		}
	}
}

func TestControlRoomShowsBestEffortConnectedFromOpenScorerStreams(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	token := f.pair(t, 1)
	if f.controlRoom(t).lane(t, 1).Connected {
		t.Fatal("connected before any stream")
	}
	s := openScorerStream(f, token)
	f.eventually(t, "lane 1 connected", func(r controlRoom) bool { return r.lane(t, 1).Connected })
	if f.controlRoom(t).lane(t, 2).Connected {
		t.Fatal("lane 2 has no stream")
	}
	s.Close()
	f.eventually(t, "lane 1 disconnected", func(r controlRoom) bool { return !r.lane(t, 1).Connected })
}

func TestControlRoomGatesStartNextAndAcceptAll(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	room := f.controlRoom(t)
	if room.CanStartNext || room.CanAcceptAll {
		t.Fatalf("all lanes open: %+v", room)
	}
	f.deviceLock(t, 1, 1)
	if room = f.controlRoom(t); room.CanStartNext || room.CanAcceptAll {
		t.Fatalf("lane 1 still open: %+v", room)
	}
	f.lockOnDevice(t, f.pair(t, 2))
	if room = f.controlRoom(t); room.CanStartNext || !room.CanAcceptAll {
		t.Fatalf("every lane awaiting acceptance: %+v", room)
	}
	f.env.Do("POST", f.adminSegment(1, 1, "/accept"), nil)
	f.env.Do("POST", f.adminSegment(2, 1, "/accept"), nil)
	if room = f.controlRoom(t); !room.CanStartNext || room.CanAcceptAll {
		t.Fatalf("every lane accepted: %+v", room)
	}
}

func TestAcceptAllIsRejectedWhileALaneIsOpen(t *testing.T) {
	f := newScoringFixture(t, 1, 2, 3)
	f.deviceLock(t, 1, 1)
	status, _ := f.env.Do("POST", fmt.Sprintf("/api/matches/%d/accept-all", f.comp), nil)
	if status != http.StatusConflict {
		t.Fatalf("accept all with open lanes: %d", status)
	}
	if got := f.segmentStatus(t, 2, 1); got != "awaiting_acceptance" {
		t.Fatalf("nothing may be accepted: %q", got)
	}
}

func TestAcceptAllAcceptsEveryAwaitingSegmentInOneAction(t *testing.T) {
	f := newScoringFixture(t, 1, 2, 3)
	f.lockOnDevice(t, f.pair(t, 1))
	f.lockOnDevice(t, f.pair(t, 2))
	f.lockOnDevice(t, f.pair(t, 3))

	status, out := f.env.Do("POST", fmt.Sprintf("/api/matches/%d/accept-all", f.comp), nil)
	if status != http.StatusOK || out["accepted"] != float64(3) {
		t.Fatalf("accept all: %d %v", status, out)
	}
	for lane := 1; lane <= 3; lane++ {
		if got := f.segmentStatus(t, lane, 1); got != "accepted" {
			t.Fatalf("lane %d status %q", lane, got)
		}
	}
	if room := f.controlRoom(t); !room.CanStartNext {
		t.Fatalf("gate should be open: %+v", room)
	}
	var recorded int
	for _, ev := range f.env.SegmentEvents(f.comp) {
		if ev.Event == "accepted_all" && ev.ActorType == "user" {
			recorded++
		}
	}
	if recorded != 3 {
		t.Fatalf("accepted_all events: %d", recorded)
	}
}

func TestAcceptAllOnTheLastSegmentFinishesTheMatch(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	tokens := map[int]string{1: f.pair(t, 1), 2: f.pair(t, 2)}
	_, list := f.env.Do("GET", fmt.Sprintf("/api/matches/%d/entries", f.comp), nil)
	segments := len(list["entries"].([]any)[0].(map[string]any)["segments"].([]any))
	for segment := 1; segment <= segments; segment++ {
		if segment > 1 {
			if status, out := f.startNext(f.comp); status != http.StatusOK {
				t.Fatalf("start segment %d: %d %v", segment, status, out)
			}
		}
		f.scoreAndLock(t, tokens[1], 2, segment)
		f.scoreAndLock(t, tokens[2], 1, segment)
		if status, out := f.env.Do("POST", fmt.Sprintf("/api/matches/%d/accept-all", f.comp), nil); status != http.StatusOK {
			t.Fatalf("accept all segment %d: %d %v", segment, status, out)
		}
	}
	_, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d", f.comp), nil)
	if out["status"] != "finished" {
		t.Fatalf("status %v", out["status"])
	}
}

func TestAdminLocksAndUnlocksALaneWithoutADevice(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	f.env.Do("PUT", f.adminSegment(2, 1, ""), f.fullSegment(t, 2, 1))
	if status, out := f.env.Do("POST", f.adminSegment(2, 1, "/lock"), nil); status != http.StatusOK {
		t.Fatalf("admin lock: %d %v", status, out)
	}
	if lane := f.controlRoom(t).lane(t, 2); lane.Status != "accepted" || lane.Paired {
		t.Fatalf("lane 2 %+v", lane)
	}
	if status, _ := f.env.Do("POST", f.adminSegment(2, 1, "/unlock"), nil); status != http.StatusOK {
		t.Fatalf("admin unlock: %d", status)
	}
	if lane := f.controlRoom(t).lane(t, 2); lane.Status != "open" {
		t.Fatalf("lane 2 %+v", lane)
	}
}

func TestControlRoomAndAcceptAllRequireAdminAndAnActiveMatch(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	token := f.pair(t, 1)
	if status, _ := f.env.DoToken(token, "GET", fmt.Sprintf("/api/matches/%d/control-room", f.comp), nil); status == http.StatusOK {
		t.Fatal("a device must not read the control room")
	}
	draft := newFixture(t, "150")
	if status, _ := draft.env.Do("GET", fmt.Sprintf("/api/matches/%d/control-room", draft.comp), nil); status != http.StatusConflict {
		t.Fatalf("draft control room: %d", status)
	}
	if status, _ := draft.env.Do("POST", fmt.Sprintf("/api/matches/%d/accept-all", draft.comp), nil); status != http.StatusConflict {
		t.Fatalf("draft accept all: %d", status)
	}
}

type historyEvent struct {
	Segment     int
	Event       string
	ActorType   string
	ActorName   string
	ShooterName string
	ScorerName  string
	Time        string
}

type scorerCount struct {
	ScorerName string
	Locks      int
	Disputes   int
}

func (f *scoringFixture) history(t *testing.T, query string) ([]historyEvent, []scorerCount) {
	t.Helper()
	status, out := f.env.Do("GET", fmt.Sprintf("/api/matches/%d/segment-events%s", f.comp, query), nil)
	if status != http.StatusOK {
		t.Fatalf("segment events: %d %v", status, out)
	}
	var events []historyEvent
	for _, e := range out["events"].([]any) {
		m := e.(map[string]any)
		events = append(events, historyEvent{
			Segment: int(m["segment"].(float64)), Event: m["event"].(string), ActorType: m["actor_type"].(string),
			ActorName: m["actor_name"].(string), ShooterName: m["shooter_name"].(string),
			ScorerName: m["scorer_name"].(string), Time: m["time"].(string),
		})
	}
	var counts []scorerCount
	for _, c := range out["scorers"].([]any) {
		m := c.(map[string]any)
		counts = append(counts, scorerCount{ScorerName: m["scorer_name"].(string), Locks: int(m["locks"].(float64)), Disputes: int(m["disputes"].(float64))})
	}
	return events, counts
}

func TestSegmentEventHistoryResolvesDeviceActorsAndCountsDisputesPerScorer(t *testing.T) {
	f := newScoringFixture(t, 1, 2, 3)
	f.deviceLock(t, 1, 1)
	if status, _ := f.deviceAction(f.pair(t, 2), 1, "dispute"); status != http.StatusOK {
		t.Fatalf("dispute: %d", status)
	}
	f.lockOnDevice(t, f.pair(t, 1))
	f.deviceAction(f.pair(t, 2), 1, "accept")
	f.lockOnDevice(t, f.pair(t, 2))
	f.env.Do("POST", f.adminSegment(3, 1, "/accept"), nil)

	events, counts := f.history(t, "")
	if len(events) != 6 {
		t.Fatalf("events %+v", events)
	}
	first := events[0]
	if first.Event != "locked" || first.ActorType != "device" || first.ScorerName != "A SHOOTER1" || first.ShooterName != "A SHOOTER2" || first.Segment != 1 || first.Time == "" {
		t.Fatalf("first %+v", first)
	}
	if events[1].Event != "disputed" || events[1].ShooterName != "A SHOOTER2" || events[1].ScorerName != "A SHOOTER1" {
		t.Fatalf("dispute %+v", events[1])
	}
	last := events[5]
	if last.Event != "accepted" || last.ActorType != "user" || last.ActorName != apitest.AdminUsername {
		t.Fatalf("last %+v", last)
	}
	want := map[string]scorerCount{"A SHOOTER1": {Locks: 2, Disputes: 1}, "A SHOOTER2": {Locks: 1}}
	for _, c := range counts {
		w, ok := want[c.ScorerName]
		if ok && (c.Locks != w.Locks || c.Disputes != w.Disputes) {
			t.Fatalf("counts %+v", counts)
		}
		delete(want, c.ScorerName)
	}
	if len(want) != 0 {
		t.Fatalf("counts %+v", counts)
	}
}

func TestSegmentEventHistoryFiltersByScorerShooterAndSegment(t *testing.T) {
	f := newScoringFixture(t, 1, 2, 3)
	f.deviceLock(t, 1, 1)
	f.deviceAction(f.pair(t, 2), 1, "dispute")
	f.lockOnDevice(t, f.pair(t, 2))
	f.lockOnDevice(t, f.pair(t, 3))

	if events, _ := f.history(t, "?scorer=A+SHOOTER1"); len(events) != 2 {
		t.Fatalf("scorer filter %+v", events)
	}
	events, counts := f.history(t, "?shooter=A+SHOOTER3")
	if len(events) != 1 || events[0].ShooterName != "A SHOOTER3" || len(counts) != 1 || counts[0].Disputes != 0 {
		t.Fatalf("shooter filter %+v %+v", events, counts)
	}
	if events, _ := f.history(t, "?segment=2"); len(events) != 0 {
		t.Fatalf("segment filter %+v", events)
	}
	if status, _ := f.env.Do("GET", fmt.Sprintf("/api/matches/%d/segment-events?segment=x", f.comp), nil); status != http.StatusUnprocessableEntity {
		t.Fatalf("bad segment: %d", status)
	}
}

func TestSegmentEventHistoryRequiresAdmin(t *testing.T) {
	f := newScoringFixture(t, 1, 2)
	if status, _ := f.env.DoToken(f.pair(t, 1), "GET", fmt.Sprintf("/api/matches/%d/segment-events", f.comp), nil); status == http.StatusOK {
		t.Fatal("a device must not read the history")
	}
}
