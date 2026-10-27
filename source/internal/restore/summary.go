package restore

import (
	"fmt"
	"io"
)

func WriteSummary(w io.Writer, plan Plan, stats Stats) {
	_, _ = fmt.Fprintf(w, "read %d lines: %d score events, %d non-score skipped, %d malformed skipped\n",
		stats.Lines, stats.Events, stats.NonScore, stats.Malformed)
	entries, stages := 0, 0
	for _, c := range plan.Matches {
		_, _ = fmt.Fprintf(w, "\nmatch %d: %s (%s, %s)\n", c.ID, c.Name, c.Date, c.CourseOfFire.Key)
		for _, e := range c.Entries {
			entries++
			_, _ = fmt.Fprintf(w, "  entry %d: %s %s, lane %d, %s, %d scored stage(s)\n",
				e.ID, e.Shooter.FirstName, e.Shooter.LastName, e.Lane, e.Category, len(e.Stages))
			for _, s := range e.Stages {
				stages++
				_, _ = fmt.Fprintf(w, "    stage %d: %v %s\n", s.Number, s.RingCounts, s.Status)
			}
		}
	}
	_, _ = fmt.Fprintf(w, "\nwould restore %d match(s), %d entries, %d scored stage(s)\ndry run: nothing was written (use --apply to restore)\n",
		len(plan.Matches), entries, stages)
}

type Applied struct {
	Matches   int
	Entries   int
	Stages    int
	Conflicts []string
}

func WriteApplied(w io.Writer, stats Stats, applied Applied) {
	_, _ = fmt.Fprintf(w, "read %d lines: %d score events, %d non-score skipped, %d malformed skipped\n",
		stats.Lines, stats.Events, stats.NonScore, stats.Malformed)
	_, _ = fmt.Fprintf(w, "restored %d match(s), %d entries, %d scored stage(s)\n", applied.Matches, applied.Entries, applied.Stages)
	if len(applied.Conflicts) > 0 {
		_, _ = fmt.Fprintf(w, "%d conflict(s) left unchanged:\n", len(applied.Conflicts))
		for _, c := range applied.Conflicts {
			_, _ = fmt.Fprintf(w, "  %s\n", c)
		}
	}
	_, _ = fmt.Fprintln(w, "note: unscored entries, devices and pairings are not restored; re-pair the devices before scoring continues")
}
