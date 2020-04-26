package score

import (
	"math"
	"testing"
	"time"

	"github.com/augmentable-dev/gitpert/pkg/gitlog"
)

func TestScoreSimple(t *testing.T) {
	commits := []*gitlog.Commit{
		&gitlog.Commit{
			SHA: "002",
			Author: gitlog.Event{
				Name:  "A",
				Email: "a",
				When:  time.Now(),
			},
			Stats: map[string]gitlog.Stat{
				"file": gitlog.Stat{Additions: 100},
			},
		},
		&gitlog.Commit{
			SHA: "001",
			Author: gitlog.Event{
				Name:  "A",
				Email: "a",
				When:  time.Now().Add(-30 * 24 * time.Hour),
			},
			Stats: map[string]gitlog.Stat{
				"file": gitlog.Stat{Additions: 100},
			},
		},
	}
	aAggs := AuthorAggregates(commits, 30)

	if len(aAggs) != 1 {
		t.Fatalf("expected 1 unique author, got: %d", len(aAggs))
	}

	if aAggs[0].Commits != 2 {
		t.Fatalf("expected 2 commits from the one author, got: %d", aAggs[0].Commits)
	}

	if aAggs[0].Impact != 200 {
		t.Fatalf("expected an impact of 200 from the one author, got: %d", aAggs[0].Impact)
	}

	if int(math.Round(aAggs[0].Score)) != 150 {
		t.Fatalf("expected a score of 150 from the one author, got: %d", int(math.Round(aAggs[0].Score)))
	}
}
