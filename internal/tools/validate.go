package tools

import (
	"fmt"
	"os"
	"strings"
)

func validateTool(t Tool) error {
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(t.Executable) == "" {
		return fmt.Errorf("executable is required")
	}
	info, err := os.Stat(t.Executable)
	if err != nil {
		return fmt.Errorf("executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("executable must be a regular file")
	}
	return nil
}
