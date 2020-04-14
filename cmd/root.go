package cmd

import (
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/spf13/cobra"
	"github.com/src-d/enry/v2"
)

func handleError(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

var (
	decayDays int
	full      bool
	remote    bool
)

func init() {
	rootCmd.Flags().IntVarP(&decayDays, "decay-rate", "d", 30, "determines how long it takes for the impact of a commit to halve, based on how recently the commit was made")
	rootCmd.Flags().BoolVarP(&remote, "remote", "r", false, "whether or not this is a remote repository")
	rootCmd.Flags().BoolVarP(&full, "full", "f", false, "include all commits when calculating scores")
}

var rootCmd = &cobra.Command{
	Use:   "gitpert",
	Short: "gitpert ranks committers ",
	Args:  cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {

		// if first argument exists, it's the repoPath
		var repoPath string
		if len(args) > 0 {
			repoPath = args[0]
		} else { // otherwise, use the working directory
			p, err := os.Getwd()
			handleError(err)
			repoPath = p
		}

		var repo *git.Repository
		// if the remote flag is set, clone the repo (using repoPath) into memory
		if remote {
			r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
				URL:          repoPath,
				SingleBranch: true,
			})
			handleError(err)
			repo = r
		} else { // otherwise, open the specified repo
			r, err := git.PlainOpen(repoPath)
			handleError(err)
			repo = r
		}

		var fileName *string
		if len(args) > 1 {
			fileName = &args[1]
		}

		// TODO (patrickdevivo) at some point this entire scoring logic should be brought out into a subpackage with some tests
		// this could also make it possibe for other projects to import the implementation.
		decayHours := 24 * decayDays

		// this ignores any commits older than 100 half-lives,
		var since time.Time
		if !full {
			since = time.Now().Add(-(time.Duration(decayHours) * time.Hour * 10))
		}
		commitIter, err := repo.Log(&git.LogOptions{
			Order:    git.LogOrderCommitterTime,
			FileName: fileName,
			Since:    &since,
		})
		handleError(err)
		defer commitIter.Close()

		type authorAggregate struct {
			email   string
			name    string
			commits int
			impact  int
			score   float64
		}
		authors := map[string]*authorAggregate{}
		var authorEmails []string
		commitIter.ForEach(func(commit *object.Commit) error {
			authorEmail := commit.Author.Email
			if _, ok := authors[authorEmail]; !ok {
				authors[authorEmail] = &authorAggregate{
					email: authorEmail,
					name:  commit.Author.Name,
				}
				authorEmails = append(authorEmails, authorEmail)
			}

			agg := authors[authorEmail]
			hoursAgo := time.Now().Sub(commit.Author.When).Hours()
			agg.commits++

			// TODO this is a bit hacky, we're absorbing any panics that occur
			// in particular, it's meant to capture an index out of range error occurring
			// under some conditions in the underlying git/diff dependency. Maybe another reason to use native git...
			defer func() {
				if err := recover(); err != nil {
					agg.score += math.Exp2(-hoursAgo / float64(decayHours))
				}
			}()

			fileStats, err := commit.Stats()
			handleError(err)

			var additions int
			var deletions int
			for _, stat := range fileStats {
				// ignore diffs in vendor files
				// TODO perhaps it's worth allowing for the user to supply file path patterns be ignored?
				if enry.IsVendor(stat.Name) {
					continue
				}
				additions += stat.Addition
				deletions += stat.Deletion
			}
			agg.impact += additions + deletions

			agg.score += float64(additions+deletions) * math.Exp2(-hoursAgo/float64(decayHours))
			return nil
		})

		sort.SliceStable(authorEmails, func(i, j int) bool {
			return authors[authorEmails[j]].score < authors[authorEmails[i]].score
		})

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
		fmt.Fprintf(w, "Rank\tEmail\tName\tScore\tImpact\tCommits\n")
		for rank, authorEmail := range authorEmails {
			agg := authors[authorEmail]
			// ignore scores less than 0
			if agg.score < 1 {
				continue
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\t%d\n", rank+1, authorEmail, agg.name, int(math.Round(agg.score)), agg.impact, agg.commits)
		}
		w.Flush()
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
