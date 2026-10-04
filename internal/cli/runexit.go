package cli

import "github.com/Rethunk-Tech/mortar/internal/launch"

func runEndedLabel(x *launch.Exit) string {
	if x == nil {
		return ""
	}
	return launch.DescribeExit(*x)
}
