package cmd

import (
	"io"
	"os"
)

func readInputFromStdin() (string, bool, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return "", false, err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return "", false, nil
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", false, err
	}
	return string(data), true, nil
}
