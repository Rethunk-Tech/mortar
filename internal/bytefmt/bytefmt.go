// Package bytefmt words a byte count for text Go writes (CLI tables, error sentences). Units are binary steps labelled
// KB, MB, GB, as the window's formatBytes labels them.
package bytefmt

import "fmt"

// Format is n as whole bytes under 1 KB ("512 B") and otherwise to one decimal in the largest fitting unit ("1.5 MB").
func Format(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
