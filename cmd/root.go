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
	remote    bool
	decayDays int
)

func init() {
	rootCmd.Flags().BoolVarP(&remote, "remote", "r", false, "whether or not this is a remote repository")
	rootCmd.Flags().IntVarP(&decayDays, "drop-off", "d", 30, "drop off duration in days")
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
				URL: repoPath,
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
		// since := time.Now().Add(-(time.Duration(decayHours) * time.Hour * 10))
		commitIter, err := repo.Log(&git.LogOptions{
			Order:    git.LogOrderCommitterTime,
			FileName: fileName,
			// Since:    &since,
		})
		handleError(err)
		defer commitIter.Close()

		type authorAggregate struct {
			email   string
			name    string
			commits []*object.Commit
			impact  int
			score   float64
		}
		authors := map[string]*authorAggregate{}
		var authorEmails []string
		commitIter.ForEach(func(commit *object.Commit) error {
			authorEmail := commit.Author.Email
			if _, ok := authors[authorEmail]; !ok {
				authors[authorEmail] = &authorAggregate{
					email:   authorEmail,
					name:    commit.Author.Name,
					commits: make([]*object.Commit, 0),
				}
				authorEmails = append(authorEmails, authorEmail)
			}

			agg := authors[authorEmail]
			agg.commits = append(authors[authorEmail].commits, commit)

			fileStats, err := commit.Stats()
			handleError(err)

			var additions int
			var deletions int
			for _, stat := range fileStats {
				// ignore diffs in vendor files
				if enry.IsVendor(stat.Name) {
					continue
				}
				additions += stat.Addition
				deletions += stat.Deletion
			}
			agg.impact += additions + deletions

			hoursAgo := time.Now().Sub(commit.Author.When).Hours()
			agg.score += float64(additions+deletions) * math.Exp2(-hoursAgo/float64(decayHours))
			return nil
		})

		sort.SliceStable(authorEmails, func(i, j int) bool {
			return authors[authorEmails[j]].score < authors[authorEmails[i]].score
		})

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
		for rank, authorEmail := range authorEmails {
			agg := authors[authorEmail]
			if agg.score < 1 {
				continue
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d commits\t%d\t%f\n", rank+1, authorEmail, agg.name, int(math.Round(agg.score)), len(agg.commits), agg.impact, float64(agg.impact)/agg.score)
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
