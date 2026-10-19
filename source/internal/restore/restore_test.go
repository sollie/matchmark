package restore

import (
	"fmt"
	"strings"
	"testing"
)

type line struct {
	event     string
	emittedAt string
	seq       int
	compName  string
	entryID   int
	category  string
	stage     int
	counts    string
}

func (l line) String() string {
	if l.compName == "" {
		l.compName = "Club Match"
	}
	if l.entryID == 0 {
		l.entryID = 1
	}
	if l.category == "" {
		l.category = "Open"
	}
	if l.stage == 0 {
		l.stage = 1
	}
	if l.counts == "" {
		l.counts = "[1,2,3,4,0,0,0]"
	}
	return fmt.Sprintf(`{"time":"x","level":"INFO","msg":"%s","schema_version":1,"event":"%s","emitted_at":"%s","seq":%d,`+
		`"competition":{"id":7,"name":"%s","date":"2026-10-19","course_of_fire":{"key":"wa1500","discipline":"WA1500","total_shots":60}},`+
		`"entry":{"id":%d,"lane":2,"category":"%s","shooter":{"waid":"W1","first_name":"Ann","last_name":"Lee","club":"SK"}},`+
		`"stage":{"number":%d,"expected_shots":10},"ring_counts":%s,"locked_by":"Admin","scorer_name":"Admin","locked_at":"2026-10-19T10:00:00Z"}`,
		l.event, l.event, l.emittedAt, l.seq, l.compName, l.entryID, l.category, l.stage, l.counts)
}

func lockedAt(t string, seq int) line {
	return line{event: "stage_locked", emittedAt: "2026-10-19T10:00:" + t + ".000Z", seq: seq}
}

func run(t *testing.T, input string) (Plan, Stats) {
	t.Helper()
	events, stats, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return Fold(events), stats
}

func TestParseSkipsNonScoreAndMalformedLines(t *testing.T) {
	input := strings.Join([]string{
		`{"msg":"request","method":"GET"}`,
		"not json at all",
		`{"msg":"stage_locked","schema_version":1,"event":"stage_locked","ring_counts":[1]}`,
		"",
		lockedAt("01", 1).String(),
	}, "\n")
	plan, stats := run(t, input)
	if stats.Lines != 4 || stats.Events != 1 || stats.NonScore != 1 || stats.Malformed != 2 {
		t.Fatalf("stats = %+v", stats)
	}
	if len(plan.Competitions) != 1 {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestParseRejectsUnknownSchemaVersion(t *testing.T) {
	bad := strings.Replace(lockedAt("01", 1).String(), `"schema_version":1`, `"schema_version":2`, 1)
	_, _, err := Parse(strings.NewReader(bad))
	if err == nil || !strings.Contains(err.Error(), "schema_version 2") {
		t.Fatalf("err = %v", err)
	}
}

func TestFoldLastLineWinsByEmittedAtThenSeq(t *testing.T) {
	a := lockedAt("05", 9)
	a.counts = "[0,0,0,0,0,0,10]"
	b := lockedAt("05", 10)
	b.counts = "[10,0,0,0,0,0,0]"
	c := lockedAt("04", 99)
	c.counts = "[0,10,0,0,0,0,0]"
	plan, _ := run(t, strings.Join([]string{b.String(), c.String(), a.String()}, "\n"))
	stages := plan.Competitions[0].Entries[0].Stages
	if len(stages) != 1 || stages[0].RingCounts != [7]int{10, 0, 0, 0, 0, 0, 0} {
		t.Fatalf("stages = %+v", stages)
	}
}

func TestFoldFinalUnlockMeansUnscored(t *testing.T) {
	unlock := lockedAt("06", 2)
	unlock.event = "stage_unlocked"
	plan, _ := run(t, lockedAt("05", 1).String()+"\n"+unlock.String())
	if len(plan.Competitions) != 0 {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestFoldRelockAfterUnlockIsScored(t *testing.T) {
	unlock := lockedAt("06", 2)
	unlock.event = "stage_unlocked"
	relock := lockedAt("07", 3)
	plan, _ := run(t, lockedAt("05", 1).String()+"\n"+unlock.String()+"\n"+relock.String())
	if len(plan.Competitions[0].Entries[0].Stages) != 1 {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestFoldContextComesFromLatestLine(t *testing.T) {
	old := lockedAt("05", 1)
	old.stage = 1
	newer := lockedAt("06", 2)
	newer.stage = 2
	newer.compName = "Renamed"
	newer.category = "Veteran"
	plan, _ := run(t, old.String()+"\n"+newer.String())
	comp := plan.Competitions[0]
	entry := comp.Entries[0]
	if comp.Name != "Renamed" || entry.Category != "Veteran" || len(entry.Stages) != 2 {
		t.Fatalf("plan = %+v", plan)
	}
	if entry.Stages[0].Number != 1 || entry.Stages[1].Number != 2 {
		t.Fatalf("stages not ordered: %+v", entry.Stages)
	}
}

func TestFoldContextFromLatestUnlockWhenStagesRemain(t *testing.T) {
	s1 := lockedAt("05", 1)
	s2 := lockedAt("06", 2)
	s2.stage = 2
	unlock := lockedAt("07", 3)
	unlock.event = "stage_unlocked"
	unlock.stage = 2
	unlock.category = "Veteran"
	plan, _ := run(t, strings.Join([]string{s1.String(), s2.String(), unlock.String()}, "\n"))
	entry := plan.Competitions[0].Entries[0]
	if entry.Category != "Veteran" || len(entry.Stages) != 1 {
		t.Fatalf("entry = %+v", entry)
	}
}

func TestSummaryListsCompetitionsEntriesAndStages(t *testing.T) {
	plan, stats := run(t, lockedAt("05", 1).String()+"\nnot json")
	var b strings.Builder
	WriteSummary(&b, plan, stats)
	out := b.String()
	for _, want := range []string{"Club Match", "2026-10-19", "Ann Lee", "lane 2", "stage 1", "1 malformed", "dry run"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q:\n%s", want, out)
		}
	}
}
