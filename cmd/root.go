package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/augmentable-dev/gitpert/pkg/gitlog"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"github.com/src-d/enry/v2"
)

func handleError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var (
	decayDays int
	remote    bool
)

func init() {
	rootCmd.Flags().IntVarP(&decayDays, "decay-rate", "d", 30, "determines how long it takes for the impact of a commit to halve, based on how recently the commit was made")
	rootCmd.Flags().BoolVarP(&remote, "remote", "r", false, "whether or not this is a remote repository")
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

		if remote {
			dir, err := ioutil.TempDir("", "gitpert_remote_repo")
			handleError(err)
			defer os.RemoveAll(dir)

			_, err = git.PlainClone(dir, false, &git.CloneOptions{
				URL: repoPath,
				// Progress: os.Stdout,
			})
			handleError(err)

			repoPath = dir
		}

		var fileName string
		if len(args) > 1 {
			fileName = args[1]
		}

		commits, err := gitlog.Exec(context.Background(), repoPath, fileName, []string{})
		handleError(err)

		// TODO (patrickdevivo) at some point this entire scoring logic should be brought out into a subpackage with some tests
		// this could also make it possibe for other projects to import the implementation.
		decayHours := 24 * decayDays

		type authorAggregate struct {
			email   string
			name    string
			commits int
			impact  int
			score   float64
		}
		authors := map[string]*authorAggregate{}
		var authorEmails []string
		var firstCommit *gitlog.Commit
		for _, commit := range commits {
			if firstCommit == nil {
				firstCommit = commit
			}
			authorEmail := commit.Author.Email
			if _, ok := authors[authorEmail]; !ok {
				authors[authorEmail] = &authorAggregate{
					email: authorEmail,
					name:  commit.Author.Name,
				}
				authorEmails = append(authorEmails, authorEmail)
			}

			agg := authors[authorEmail]
			hoursAgo := firstCommit.Author.When.Sub(commit.Author.When).Hours()
			agg.commits++

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
			agg.impact += additions + deletions

			agg.score += float64(additions+deletions) * math.Exp2(-hoursAgo/float64(decayHours))
		}

		sort.SliceStable(authorEmails, func(i, j int) bool {
			return authors[authorEmails[j]].score < authors[authorEmails[i]].score
		})

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
		fmt.Fprintf(w, "Rank\tEmail\tName\tScore\tImpact\tCommits\n")
		for rank, authorEmail := range authorEmails {
			agg := authors[authorEmail]
			// only print the top 10
			if rank > 9 {
				break
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
