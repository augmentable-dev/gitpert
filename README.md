[![GoDoc](https://godoc.org/github.com/augmentable-dev/gitpert?status.svg)](https://godoc.org/github.com/augmentable-dev/gitpert)
[![Go Report Card](https://goreportcard.com/badge/github.com/augmentable-dev/gitpert)](https://goreportcard.com/report/github.com/augmentable-dev/gitpert)
[![TODOs](https://badgen.net/https/api.tickgit.com/badgen/github.com/augmentable-dev/gitpert)](https://www.tickgit.com/browse?repo=github.com/augmentable-dev/gitpert)

# gitpert

`gitpert` measures the "pertinence" of git authors as a time-decayed measure of LOC added and removed to a repository (or a set of files in a repository).
It's meant to help identify who the most relevant contributors are based on commit recency, frequency and impact.

- **impact** in this context is lines of code added plus lines of code removed by a commit. Vendored dependency files are ignored (as best as possible).
- **decay rate** determines how long it takes for the impact of a commit to halve, based on how recently the commit was made. If the decay rate is 10 days, a commit that added 100 lines of code, authored 10 days ago, will be scored at 50. It is a half-life, and can be supplied as a config parameter.
- **score** is the sum of the time decayed impact of every commit in a repository, for a given author.

The net effect *should* be a ranked list of authors (contributors) where those who have more recently and more frequently contributed "larger" commits surface to the top.
An author who committed the initial code many years ago (maybe higher impact) will likely rank lower than an author who has contributed less impactfully, but much more recently (depending on the decay rate and absolute numbers, of course).

This could be useful for identifying who the best person to review a new code change might be, or who the best person to ask questions or seek help from might be. Scoring can be done at the repository level, and also for individual files (the most pertinent author for a repository might not be the most pertinent for a directory or file within that repository).


## Installation

### Homebrew

```
brew tap augmentable-dev/gitpert
brew install gitpert
```

## Usage

Running

```
gitpert
```

in a git repository will output a ranking table like so:

```
Rank   Email                                      Name                      Score   Impact   Commits
1      user1@users.noreply.github.com             User One                  191     6064     92
2      user2@gmail.com                            User Two                  171     931      27
3      user3@gmail.com                            User Three                40      4366     38
...
```

use the `-d` flag to set the decay rate (in days) `gitpert -d 90` will output a ranking table where score is based on impact that halves at commits made 90 days ago.

```
gitpert -h
```

will have more specific and up to date usage info.

## FAQ

### What about git-blame?
`git-blame` will tell you about the last modification to lines in a file (the author and revision), and is certainly useful. This tool hopes to provide a higher level view of the net effect of authorship in a repository as a function of time.

### Why are changes to "vendored" dependencies ignored?
Authoring a large commit because it adds or removes many dependencies (think the `vendor/` directory in golang projects), while in most contexts an important contribution, gives an outsized "impact" to that commit which probably shouldn't be evaluated in the same way as "typical" LOC.

### Should LOC added be weighed the same as LOC removed?
Maybe. This could be worth exposing as a config parameter. One could argue that a LOC added should weigh some amount more than a LOC removed.

### How are commits aggregated to an author?
By email address. Authors using multiple email addresses (say, because of GitHub's `...@users.noreply.github.com`) will be tracked as distinct authors. An option to enumerate which emails should be treated as belonging to the same author should be implemented.
