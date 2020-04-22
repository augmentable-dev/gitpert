package gitlog

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Commit represents a parsed commit from git log
type Commit struct {
	SHA       string
	Author    Event
	Committer Event
	Stats     map[string]Stat
}

// Event represents the who and when of a commit event
type Event struct {
	Name  string
	Email string
	When  time.Time
}

// Stat holds the diff stat of a file
type Stat struct {
	Additions int
	Deletions int
}

// Result is a list of commits
type Result []*Commit

func parseLog(reader io.Reader) (Result, error) {
	scanner := bufio.NewScanner(reader)
	res := make(Result, 0)

	// line prefixes for the `fuller` formatted output
	const (
		commit     = "commit "
		author     = "Author: "
		authorDate = "AuthorDate: "

		committer  = "Commit: "
		commitDate = "CommitDate: "
	)

	var currentCommit *Commit
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, commit):
			if currentCommit != nil { // if we're seeing a new commit but already have a current commit, we've finished a commit
				res = append(res, currentCommit)
			}
			currentCommit = &Commit{
				SHA:   strings.TrimPrefix(line, commit),
				Stats: make(map[string]Stat),
			}
		case strings.HasPrefix(line, author):
			s := strings.TrimPrefix(line, author)
			spl := strings.Split(s, " ")
			email := strings.Trim(spl[len(spl)-1], "<>")
			name := strings.Join(spl[:len(spl)-1], " ")
			currentCommit.Author.Email = strings.Trim(email, "<>")
			currentCommit.Author.Name = strings.TrimSpace(name)
		case strings.HasPrefix(line, authorDate):
			authorDateString := strings.TrimPrefix(line, authorDate)
			aD, err := time.Parse(time.RFC3339, authorDateString)
			if err != nil {
				return nil, err
			}
			currentCommit.Author.When = aD
		case strings.HasPrefix(line, committer):
			s := strings.TrimPrefix(line, committer)
			spl := strings.Split(s, " ")
			email := strings.Trim(spl[len(spl)-1], "<>")
			name := strings.Join(spl[:len(spl)-1], " ")
			currentCommit.Committer.Email = strings.Trim(email, "<>")
			currentCommit.Committer.Name = strings.TrimSpace(name)
		case strings.HasPrefix(line, commitDate):
			commitDateString := strings.TrimPrefix(line, commitDate)
			cD, err := time.Parse(time.RFC3339, commitDateString)
			if err != nil {
				return nil, err
			}
			currentCommit.Committer.When = cD
		case strings.HasPrefix(line, " "): // ignore commit message lines
		case strings.TrimSpace(line) == "": // ignore empty lines
		default:
			s := strings.Split(line, "\t")
			var additions int
			var deletions int
			var err error
			if s[0] != "-" {
				additions, err = strconv.Atoi(s[0])
				if err != nil {
					return nil, err
				}
			}
			if s[1] != "-" {
				deletions, err = strconv.Atoi(s[1])
				if err != nil {
					return nil, err
				}
			}
			currentCommit.Stats[s[2]] = Stat{
				Additions: additions,
				Deletions: deletions,
			}
		}
	}
	if currentCommit != nil {
		res = append(res, currentCommit)
	}

	return res, nil
}

// Exec runs the git log command
func Exec(ctx context.Context, repoPath string, filePattern string, additionalFlags []string) (Result, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("could not find git: %w", err)
	}

	args := []string{"log"}

	args = append(args, "--numstat", "--format=fuller", "--no-merges", "--no-decorate", "--date=iso8601-strict", "-w")
	args = append(args, additionalFlags...)
	if filePattern != "" {
		args = append(args, filePattern)
	}

	cmd := exec.CommandContext(ctx, gitPath, args...)
	cmd.Dir = repoPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	res, err := parseLog(stdout)
	if err != nil {
		return nil, err
	}

	errs, err := ioutil.ReadAll(stderr)
	if err != nil {
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		fmt.Println(string(errs))
		return nil, err
	}

	return res, nil
}
