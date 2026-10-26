package devsvc

import (
	"context"
	"strconv"
	"strings"
)

// gitState reads the branch, changed files and unpushed commits of the repository in dir; nil when it is not one.
func (s *Service) gitState(ctx context.Context, dir string) *GitState {
	// A project folder inside another repository has none of its own; that repository's state is not the project's.
	if prefix, err := s.run(ctx, dir, "git", "rev-parse", "--show-prefix"); err != nil || strings.TrimSpace(string(prefix)) != "" {
		return nil
	}
	out, err := s.run(ctx, dir, "git", "status", "--porcelain=v2", "--branch")
	if err != nil {
		return nil
	}
	state := parseStatus(string(out))
	// Before the first commit there is nothing to count.
	if n, err := s.run(ctx, dir, "git", "rev-list", "--count", "HEAD", "--not", "--remotes"); err == nil {
		state.Unpushed, _ = strconv.Atoi(strings.TrimSpace(string(n)))
	}
	return &state
}

// parseStatus reads `git status --porcelain=v2 --branch`: the branch from its header, a changed file per other line.
func parseStatus(out string) GitState {
	var state GitState
	for line := range strings.SplitSeq(out, "\n") {
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			state.Branch = strings.TrimPrefix(line, "# branch.head ")
		case line == "" || strings.HasPrefix(line, "#"):
		default:
			state.Changed++
		}
	}
	return state
}
