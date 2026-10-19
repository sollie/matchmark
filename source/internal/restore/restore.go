package restore

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
)

const SchemaVersion = 1

const (
	eventStageLocked   = "stage_locked"
	eventStageUnlocked = "stage_unlocked"
)

type CourseOfFire struct {
	Key        string `json:"key"`
	Discipline string `json:"discipline"`
	TotalShots int    `json:"total_shots"`
}

type Competition struct {
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
	Entry         Entry           `json:"entry"`
	Stage         Stage           `json:"stage"`
	RingCounts    []int           `json:"ring_counts"`
	LockedBy      json.RawMessage `json:"locked_by"`
	ScorerName    string          `json:"scorer_name"`
	LockedAt      string          `json:"locked_at"`
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
	LockedBy      string
	ScorerName    string
	LockedAt      string
}

type RestoredEntry struct {
	ID       int64
	Lane     int
	Category string
	Shooter  Shooter
	Stages   []ScoredStage
}

type RestoredCompetition struct {
	ID           int64
	Name         string
	Date         string
	CourseOfFire CourseOfFire
	Entries      []RestoredEntry
}

type Plan struct {
	Competitions []RestoredCompetition
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
	if ev.Event != eventStageLocked && ev.Event != eventStageUnlocked {
		return Event{}, kindNonScore, nil
	}
	if ev.SchemaVersion != SchemaVersion {
		return Event{}, kindVersion, fmt.Errorf("unsupported schema_version %d (supported: %d)", ev.SchemaVersion, SchemaVersion)
	}
	if !valid(ev) {
		return Event{}, kindMalformed, nil
	}
	return ev, kindEvent, nil
}

func valid(ev Event) bool {
	return len(ev.RingCounts) == 7 &&
		ev.EmittedAt != "" &&
		ev.Competition.ID != 0 &&
		ev.Entry.ID != 0 &&
		ev.Stage.Number > 0
}

type stageKey struct {
	competition int64
	entry       int64
	stage       int
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

	comps := map[int64]Competition{}
	entries := map[[2]int64]Entry{}
	stages := map[stageKey]Event{}
	for _, ev := range ordered {
		comps[ev.Competition.ID] = ev.Competition
		entries[[2]int64{ev.Competition.ID, ev.Entry.ID}] = ev.Entry
		k := stageKey{ev.Competition.ID, ev.Entry.ID, ev.Stage.Number}
		if ev.Event == eventStageUnlocked {
			delete(stages, k)
			continue
		}
		stages[k] = ev
	}

	byEntry := map[[2]int64][]ScoredStage{}
	for k, ev := range stages {
		var counts [7]int
		copy(counts[:], ev.RingCounts)
		var lockedBy string
		if err := json.Unmarshal(ev.LockedBy, &lockedBy); err != nil {
			lockedBy = string(ev.LockedBy)
		}
		ek := [2]int64{k.competition, k.entry}
		byEntry[ek] = append(byEntry[ek], ScoredStage{
			Number:        k.stage,
			ExpectedShots: ev.Stage.ExpectedShots,
			RingCounts:    counts,
			LockedBy:      lockedBy,
			ScorerName:    ev.ScorerName,
			LockedAt:      ev.LockedAt,
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
		plan.Competitions = append(plan.Competitions, RestoredCompetition{
			ID: c.ID, Name: c.Name, Date: c.Date, CourseOfFire: c.CourseOfFire, Entries: es,
		})
	}
	sort.Slice(plan.Competitions, func(i, j int) bool { return plan.Competitions[i].ID < plan.Competitions[j].ID })
	return plan
}
