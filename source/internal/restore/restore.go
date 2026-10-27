package restore

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"

	"github.com/sollie/matchmark/internal/rulebook"
)

const SchemaVersion = 3

const emittedAtLayout = "2006-01-02T15:04:05.000Z"

const (
	eventStageStatus   = "stage_status"
	eventStageLocked   = "stage_locked"
	eventStageUnlocked = "stage_unlocked"
)

const (
	StatusOpen               = "open"
	StatusAwaitingAcceptance = "awaiting_acceptance"
	StatusAccepted           = "accepted"
)

type CourseOfFire struct {
	Key        string `json:"key"`
	Discipline string `json:"discipline"`
	TotalShots int    `json:"total_shots"`
}

type Competition struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Match struct {
	ID           int64        `json:"id"`
	Name         string       `json:"name"`
	Date         string       `json:"date"`
	CourseOfFire CourseOfFire `json:"course_of_fire"`
}

type Shooter struct {
	WAID      string `json:"waid"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Club      string `json:"club"`
}

type Entry struct {
	ID       int64   `json:"id"`
	Lane     int     `json:"lane"`
	Category string  `json:"category"`
	Shooter  Shooter `json:"shooter"`
}

type Stage struct {
	Number        int `json:"number"`
	ExpectedShots int `json:"expected_shots"`
}

type Event struct {
	Event         string          `json:"event"`
	SchemaVersion int             `json:"schema_version"`
	EmittedAt     string          `json:"emitted_at"`
	Seq           int64           `json:"seq"`
	Competition   Competition     `json:"competition"`
	Match         Match           `json:"match"`
	Entry         Entry           `json:"entry"`
	Stage         Stage           `json:"stage"`
	RingCounts    []int           `json:"ring_counts"`
	Status        string          `json:"status"`
	LockedBy      json.RawMessage `json:"locked_by"`
	ScorerName    string          `json:"scorer_name"`
	LockedAt      string          `json:"locked_at"`
	AcceptedBy    string          `json:"accepted_by"`
	AcceptedAt    string          `json:"accepted_at"`
	DisputedAt    string          `json:"disputed_at"`
}

type Stats struct {
	Lines     int
	Events    int
	NonScore  int
	Malformed int
}

type ScoredStage struct {
	Number        int
	ExpectedShots int
	RingCounts    [7]int
	Status        string
	LockedBy      string
	Lane          int
	ScorerName    string
	LockedAt      string
	AcceptedBy    string
	AcceptedAt    string
	DisputedAt    string
}

type RestoredEntry struct {
	ID       int64
	Lane     int
	Category string
	Shooter  Shooter
	Stages   []ScoredStage
}

type RestoredMatch struct {
	Competition  Competition
	ID           int64
	Name         string
	Date         string
	CourseOfFire CourseOfFire
	Entries      []RestoredEntry
}

type Plan struct {
	Matches []RestoredMatch
}

func Parse(r io.Reader) ([]Event, Stats, error) {
	var events []Event
	var stats Stats
	br := bufio.NewReader(r)
	for {
		raw, err := br.ReadBytes('\n')
		if len(raw) > 0 {
			raw = bytes.TrimSpace(raw)
			if len(raw) > 0 {
				stats.Lines++
				ev, kind, perr := parseLine(raw)
				switch kind {
				case kindEvent:
					stats.Events++
					events = append(events, ev)
				case kindNonScore:
					stats.NonScore++
				case kindMalformed:
					stats.Malformed++
				case kindVersion:
					return nil, stats, perr
				}
			}
		}
		if errors.Is(err, io.EOF) {
			return events, stats, nil
		}
		if err != nil {
			return nil, stats, err
		}
	}
}

type lineKind int

const (
	kindEvent lineKind = iota
	kindNonScore
	kindMalformed
	kindVersion
)

func parseLine(raw []byte) (Event, lineKind, error) {
	var ev Event
	if err := json.Unmarshal(raw, &ev); err != nil {
		return Event{}, kindMalformed, nil
	}
	if ev.Event != eventStageStatus && ev.Event != eventStageLocked && ev.Event != eventStageUnlocked {
		return Event{}, kindNonScore, nil
	}
	if ev.Event != eventStageStatus || ev.SchemaVersion != SchemaVersion {
		return Event{}, kindVersion, fmt.Errorf("unsupported schema_version %d (supported: %d)", ev.SchemaVersion, SchemaVersion)
	}
	if !valid(ev) {
		return Event{}, kindMalformed, nil
	}
	return ev, kindEvent, nil
}

func valid(ev Event) bool {
	if _, err := rulebook.CourseByKey(ev.Match.CourseOfFire.Key); err != nil {
		return false
	}
	switch ev.Status {
	case StatusOpen:
	case StatusAwaitingAcceptance, StatusAccepted:
		if _, _, ok := parseLockedBy(ev.LockedBy); !ok {
			return false
		}
	default:
		return false
	}
	if t, err := time.Parse(emittedAtLayout, ev.EmittedAt); err != nil || t.Format(emittedAtLayout) != ev.EmittedAt {
		return false
	}
	return len(ev.RingCounts) == 7 &&
		ev.Competition.ID != 0 &&
		ev.Match.ID != 0 &&
		ev.Entry.ID != 0 &&
		ev.Stage.Number > 0
}

func parseLockedBy(raw json.RawMessage) (string, int, bool) {
	var name string
	if err := json.Unmarshal(raw, &name); err == nil {
		return name, 0, name != ""
	}
	var lane int
	if err := json.Unmarshal(raw, &lane); err == nil && lane > 0 {
		return strconv.Itoa(lane), lane, true
	}
	return "", 0, false
}

func ringCountsFromSlice(s []int) [7]int {
	var counts [7]int
	copy(counts[:], s)
	return counts
}

type stageKey struct {
	match int64
	entry int64
	stage int
}

func Fold(events []Event) Plan {
	ordered := make([]Event, len(events))
	copy(ordered, events)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].EmittedAt != ordered[j].EmittedAt {
			return ordered[i].EmittedAt < ordered[j].EmittedAt
		}
		return ordered[i].Seq < ordered[j].Seq
	})

	comps := map[int64]Match{}
	containers := map[int64]Competition{}
	entries := map[[2]int64]Entry{}
	stages := map[stageKey]Event{}
	for _, ev := range ordered {
		comps[ev.Match.ID] = ev.Match
		containers[ev.Match.ID] = ev.Competition
		entries[[2]int64{ev.Match.ID, ev.Entry.ID}] = ev.Entry
		k := stageKey{ev.Match.ID, ev.Entry.ID, ev.Stage.Number}
		stages[k] = ev
	}

	byEntry := map[[2]int64][]ScoredStage{}
	for k, ev := range stages {
		lockedBy, lane, _ := parseLockedBy(ev.LockedBy)
		ek := [2]int64{k.match, k.entry}
		byEntry[ek] = append(byEntry[ek], ScoredStage{
			Number:        k.stage,
			ExpectedShots: ev.Stage.ExpectedShots,
			RingCounts:    ringCountsFromSlice(ev.RingCounts),
			Status:        ev.Status,
			LockedBy:      lockedBy,
			Lane:          lane,
			ScorerName:    ev.ScorerName,
			LockedAt:      ev.LockedAt,
			AcceptedBy:    ev.AcceptedBy,
			AcceptedAt:    ev.AcceptedAt,
			DisputedAt:    ev.DisputedAt,
		})
	}

	byComp := map[int64][]RestoredEntry{}
	for ek, ss := range byEntry {
		sort.Slice(ss, func(i, j int) bool { return ss[i].Number < ss[j].Number })
		e := entries[ek]
		byComp[ek[0]] = append(byComp[ek[0]], RestoredEntry{
			ID: e.ID, Lane: e.Lane, Category: e.Category, Shooter: e.Shooter, Stages: ss,
		})
	}

	var plan Plan
	for id, es := range byComp {
		sort.Slice(es, func(i, j int) bool { return es[i].ID < es[j].ID })
		c := comps[id]
		plan.Matches = append(plan.Matches, RestoredMatch{
			Competition: containers[id], ID: c.ID, Name: c.Name, Date: c.Date, CourseOfFire: c.CourseOfFire, Entries: es,
		})
	}
	sort.Slice(plan.Matches, func(i, j int) bool { return plan.Matches[i].ID < plan.Matches[j].ID })
	return plan
}
