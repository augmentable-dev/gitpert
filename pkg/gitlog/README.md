[![go.dev](https://badgen.net/badge/go.dev/reference/blue)](https://pkg.go.dev/github.com/augmentable-dev/gitpert/pkg/gitlog?tab=doc)

## gitlog

Package for parsing the output of `git log` for the purposes of `gitpert`.
Can be used for more general purpose `git log` parsing, with (likely) some cleanup.

Essentially calls `git log --numstat --format=fuller --no-merges --no-decorate --date=iso8601-strict` with [`os/exec`](https://golang.org/pkg/os/exec/) and parses the output.
