package cmd

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Read target URLs from stdin (if piped) or a --targets-file
func readURLs() ([]string, error) {
	targetsFile := viper.GetString("targets-file")

	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	stdinPiped := (stdinInfo.Mode() & os.ModeCharDevice) == 0

	if stdinPiped && targetsFile != "" {
		return nil, errors.New("cannot use both piped stdin and --targets-file")
	}

	if stdinPiped {
		return scanURLs(os.Stdin)
	}

	if targetsFile != "" {
		f, err := os.Open(targetsFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return scanURLs(f)
	}

	return nil, nil
}

// Reads lines from r, skipping empty lines and comments (lines starting with #)
func scanURLs(r io.Reader) ([]string, error) {
	var urls []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}
	return urls, scanner.Err()
}
