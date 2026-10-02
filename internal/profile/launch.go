package profile

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var launchEnvPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=.*$`)

// LaunchPrefixArgs parses a launch prefix with shell quoting but no shell evaluation.
func LaunchPrefixArgs(prefix string) ([]string, error) {
	var args []string
	var current strings.Builder
	var quote rune
	escaped := false
	haveValue := false
	for _, r := range prefix {
		switch {
		case escaped:
			current.WriteRune(r)
			haveValue = true
			escaped = false
		case r == '\\' && quote != '\'':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
				haveValue = true
			}
		case r == '\'' || r == '"':
			quote = r
			haveValue = true
		case unicode.IsSpace(r):
			if haveValue {
				args = append(args, current.String())
				current.Reset()
				haveValue = false
			}
		default:
			current.WriteRune(r)
			haveValue = true
		}
	}
	if escaped || quote != 0 {
		return nil, errors.New("launch prefix has an unfinished quote or escape")
	}
	if haveValue {
		args = append(args, current.String())
	}
	return args, nil
}

// LaunchEnvironment validates and converts one NAME=value variable per line.
func LaunchEnvironment(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var env []string
	for lineNumber, line := range strings.Split(raw, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if !launchEnvPattern.MatchString(line) {
			return nil, fmt.Errorf("invalid launch environment variable on line %d", lineNumber+1)
		}
		env = append(env, line)
	}
	return env, nil
}

// SetLaunchSettings replaces a profile's direct-launch prefix and environment.
func (s *Store) SetLaunchSettings(game, id, prefix, env string) (Profile, error) {
	if _, err := LaunchPrefixArgs(prefix); err != nil {
		return Profile{}, err
	}
	if _, err := LaunchEnvironment(env); err != nil {
		return Profile{}, err
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		p.LaunchPrefix = prefix
		p.LaunchEnv = env
		return nil
	})
}

// LaunchSettings returns the direct-launch prefix and environment for a profile.
func (s *Store) LaunchSettings(game, id string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.read(game, id)
	if err != nil {
		return "", "", err
	}
	return p.LaunchPrefix, p.LaunchEnv, nil
}
