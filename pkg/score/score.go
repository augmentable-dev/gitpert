package score

import (
	"math"
	"sort"

	"github.com/augmentable-dev/gitpert/pkg/gitlog"
	"github.com/src-d/enry/v2"
)

// AuthorAggregate is a summary of an author's stats (given a git history)
type AuthorAggregate struct {
	Email   string
	Name    string
	Commits int
	Impact  int
	Score   float64
}

// AuthorAggregates calculates the aggregate scores of authors, given a list of commits
func AuthorAggregates(commits []*gitlog.Commit, decayDays int) []*AuthorAggregate {
	decayHours := 24 * decayDays

	authors := map[string]*AuthorAggregate{}
	var authorEmails []string
	var firstCommit *gitlog.Commit
	for _, commit := range commits {
		if firstCommit == nil {
			firstCommit = commit
		}
		authorEmail := commit.Author.Email
		if _, ok := authors[authorEmail]; !ok {
			authors[authorEmail] = &AuthorAggregate{
				Email: authorEmail,
				Name:  commit.Author.Name,
			}
			authorEmails = append(authorEmails, authorEmail)
		}

		agg := authors[authorEmail]
		hoursAgo := firstCommit.Author.When.Sub(commit.Author.When).Hours()
		agg.Commits++

		var additions int
		var deletions int
		for file, stat := range commit.Stats {
			// ignore diffs in vendor files
			// TODO perhaps it's worth allowing for the user to supply file path patterns be ignored?
			if enry.IsVendor(file) {
				continue
			}
			additions += stat.Additions
			deletions += stat.Deletions
		}
		agg.Impact += additions + deletions

		agg.Score += float64(additions+deletions) * math.Exp2(-hoursAgo/float64(decayHours))
	}

	sort.SliceStable(authorEmails, func(i, j int) bool {
		return authors[authorEmails[j]].Score < authors[authorEmails[i]].Score
	})

	res := make([]*AuthorAggregate, len(authorEmails))
	for rank, authorEmail := range authorEmails {
		res[rank] = authors[authorEmail]
	}

	return res
}
