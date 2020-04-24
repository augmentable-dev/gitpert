package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"text/tabwriter"

	"github.com/augmentable-dev/gitpert/pkg/gitlog"
	"github.com/augmentable-dev/gitpert/pkg/score"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
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

		authorScores := score.AuthorAggregates(commits, decayDays)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
		fmt.Fprintf(w, "Rank\tEmail\tName\tScore\tImpact\tCommits\n")
		for rank, agg := range authorScores {
			// ignore scores less than 0
			if agg.Score < 1 {
				continue
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\t%d\n", rank+1, agg.Email, agg.Name, int(math.Round(agg.Score)), agg.Impact, agg.Commits)
		}
		w.Flush()
	},
}

// Execute runs the root command
func Execute() {
	err := rootCmd.Execute()
	handleError(err)
}
