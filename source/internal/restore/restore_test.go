package restore

import (
	"fmt"
	"strings"
	"testing"
)

type line struct {
	status    string
	emittedAt string
	seq       int
	compName  string
	entryID   int
	category  string
	segment   int
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
	if l.segment == 0 {
		l.segment = 1
	}
	if l.counts == "" {
		l.counts = "[1,2,3,4,0,0,0]"
	}
	if l.status == "" {
		l.status = StatusAccepted
	}
	return fmt.Sprintf(`{"time":"x","level":"INFO","msg":"segment_status","schema_version":4,"event":"segment_status","status":"%s","emitted_at":"%s","seq":%d,`+
		`"competition":{"id":3,"name":"Nationals"},`+
		`"match":{"id":7,"name":"%s","date":"2026-10-19","course_of_fire":{"key":"150","discipline":"WA1500","total_shots":60}},`+
		`"entry":{"id":%d,"lane":2,"category":"%s","shooter":{"waid":"W1","first_name":"Ann","last_name":"Lee","club":"SK"}},`+
		`"segment":{"number":%d,"expected_shots":10},"ring_counts":%s,"locked_by":"bob","scorer_name":"bob","locked_at":"2026-10-19T10:00:00Z"}`,
		l.status, l.emittedAt, l.seq, l.compName, l.entryID, l.category, l.segment, l.counts)
}

func lockedAt(t string, seq int) line {
	return line{emittedAt: "2026-10-19T10:00:" + t + ".000Z", seq: seq}
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
		`{"msg":"segment_status","schema_version":4,"event":"segment_status","status":"accepted","ring_counts":[1]}`,
		"",
		lockedAt("01", 1).String(),
	}, "\n")
	plan, stats := run(t, input)
	if stats.Lines != 4 || stats.Events != 1 || stats.NonScore != 1 || stats.Malformed != 2 {
		t.Fatalf("stats = %+v", stats)
	}
	if len(plan.Matches) != 1 {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestParseRejectsUnknownSchemaVersion(t *testing.T) {
	bad := strings.Replace(lockedAt("01", 1).String(), `"schema_version":4`, `"schema_version":5`, 1)
	_, _, err := Parse(strings.NewReader(bad))
	if err == nil || !strings.Contains(err.Error(), "schema_version 5") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseRejectsVersionTwoLines(t *testing.T) {
	bad := strings.Replace(lockedAt("01", 1).String(), `"schema_version":4`, `"schema_version":2`, 1)
	if _, _, err := Parse(strings.NewReader(bad)); err == nil || !strings.Contains(err.Error(), "schema_version 2") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseRejectsVersionOneLines(t *testing.T) {
	v1 := `{"msg":"segment_locked","schema_version":1,"event":"segment_locked","emitted_at":"2026-10-19T10:00:01.000Z"}`
	if _, _, err := Parse(strings.NewReader(v1)); err == nil || !strings.Contains(err.Error(), "schema_version 1") {
		t.Fatalf("err = %v", err)
	}
}

func TestFoldKeepsOpenSegmentWithCounts(t *testing.T) {
	open := lockedAt("06", 2)
	open.status = StatusOpen
	plan, _ := run(t, lockedAt("05", 1).String()+"\n"+open.String())
	segments := plan.Matches[0].Entries[0].Segments
	if len(segments) != 1 || segments[0].Status != StatusOpen || segments[0].RingCounts != [7]int{1, 2, 3, 4, 0, 0, 0} {
		t.Fatalf("segments = %+v", segments)
	}
}

func TestParseRejectsUnknownStatus(t *testing.T) {
	bad := strings.Replace(lockedAt("01", 1).String(), `"status":"accepted"`, `"status":"bogus"`, 1)
	if _, stats := run(t, bad); stats.Malformed != 1 {
		t.Fatalf("stats = %+v", stats)
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
	segments := plan.Matches[0].Entries[0].Segments
	if len(segments) != 1 || segments[0].RingCounts != [7]int{10, 0, 0, 0, 0, 0, 0} {
		t.Fatalf("segments = %+v", segments)
	}
}

func TestFoldFinalOpenLineWins(t *testing.T) {
	unlock := lockedAt("06", 2)
	unlock.status = StatusOpen
	plan, _ := run(t, lockedAt("05", 1).String()+"\n"+unlock.String())
	if got := plan.Matches[0].Entries[0].Segments[0].Status; got != StatusOpen {
		t.Fatalf("status = %q", got)
	}
}

func TestFoldRelockAfterUnlockIsAccepted(t *testing.T) {
	unlock := lockedAt("06", 2)
	unlock.status = StatusOpen
	relock := lockedAt("07", 3)
	plan, _ := run(t, lockedAt("05", 1).String()+"\n"+unlock.String()+"\n"+relock.String())
	if got := plan.Matches[0].Entries[0].Segments[0].Status; got != StatusAccepted {
		t.Fatalf("status = %q", got)
	}
}

func TestFoldContextComesFromLatestLine(t *testing.T) {
	old := lockedAt("05", 1)
	old.segment = 1
	newer := lockedAt("06", 2)
	newer.segment = 2
	newer.compName = "Renamed"
	newer.category = "Veteran"
	plan, _ := run(t, old.String()+"\n"+newer.String())
	comp := plan.Matches[0]
	entry := comp.Entries[0]
	if comp.Name != "Renamed" || entry.Category != "Veteran" || len(entry.Segments) != 2 {
		t.Fatalf("plan = %+v", plan)
	}
	if entry.Segments[0].Number != 1 || entry.Segments[1].Number != 2 {
		t.Fatalf("segments not ordered: %+v", entry.Segments)
	}
}

func TestSummaryListsMatchesEntriesAndSegments(t *testing.T) {
	plan, stats := run(t, lockedAt("05", 1).String()+"\nnot json")
	var b strings.Builder
	WriteSummary(&b, plan, stats)
	out := b.String()
	for _, want := range []string{"Club Match", "2026-10-19", "Ann Lee", "lane 2", "segment 1", "1 malformed", "dry run"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q:\n%s", want, out)
		}
	}
}

func TestParseTreatsInvalidFieldsAsMalformed(t *testing.T) {
	good := lockedAt("01", 1).String()
	tests := []struct{ name, from, to string }{
		{"unknown course", `"key":"150"`, `"key":"nope"`},
		{"locked_by empty", `"locked_by":"bob"`, `"locked_by":""`},
		{"locked_by zero", `"locked_by":"bob"`, `"locked_by":0`},
		{"locked_by negative", `"locked_by":"bob"`, `"locked_by":-2`},
		{"emitted_at layout", `.000Z`, `Z`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, stats := run(t, strings.Replace(good, tt.from, tt.to, 1))
			if stats.Malformed != 1 || stats.Events != 0 {
				t.Fatalf("stats = %+v", stats)
			}
		})
	}
	ok := strings.Replace(good, `"locked_by":"bob"`, `"locked_by":3`, 1)
	if _, stats := run(t, ok); stats.Events != 1 {
		t.Fatalf("stats = %+v", stats)
	}
}
