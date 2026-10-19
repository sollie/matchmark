package restore

import (
	"fmt"
	"io"
)

func WriteSummary(w io.Writer, plan Plan, stats Stats) {
	_, _ = fmt.Fprintf(w, "read %d lines: %d score events, %d non-score skipped, %d malformed skipped\n",
		stats.Lines, stats.Events, stats.NonScore, stats.Malformed)
	entries, stages := 0, 0
	for _, c := range plan.Competitions {
		_, _ = fmt.Fprintf(w, "\ncompetition %d: %s (%s, %s)\n", c.ID, c.Name, c.Date, c.CourseOfFire.Key)
		for _, e := range c.Entries {
			entries++
			_, _ = fmt.Fprintf(w, "  entry %d: %s %s, lane %d, %s, %d scored stage(s)\n",
				e.ID, e.Shooter.FirstName, e.Shooter.LastName, e.Lane, e.Category, len(e.Stages))
			for _, s := range e.Stages {
				stages++
				_, _ = fmt.Fprintf(w, "    stage %d: %v locked by %s\n", s.Number, s.RingCounts, s.LockedBy)
			}
		}
	}
	_, _ = fmt.Fprintf(w, "\nwould restore %d competition(s), %d entries, %d scored stage(s)\ndry run: nothing was written\n",
		len(plan.Competitions), entries, stages)
}
