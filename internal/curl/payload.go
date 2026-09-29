package curl

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadPayload checks if val begins with '@'.
// If it does, it reads and returns the contents of the specified file.
// If it does not, it returns val unchanged.
// Returns an error if the file path is empty, does not exist, or cannot be read.
func LoadPayload(val string) (string, error) {
	if !strings.HasPrefix(val, "@") {
		return val, nil
	}

	target := strings.TrimPrefix(val, "@")
	target = strings.Trim(target, `"'`)
	if target == "" {
		return "", errors.New("empty file path after '@'")
	}

	if strings.HasPrefix(target, "~/") || target == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			if target == "~" {
				target = home
			} else {
				target = filepath.Join(home, strings.TrimPrefix(target, "~/"))
			}
		}
	}

	data, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("failed to read data file '%s': %w", target, err)
	}

	return string(data), nil
}
