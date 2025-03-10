package search

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
)

// RgSearchDriver implements the SearchDriver interface for ripgrep (rg)
type RgSearchDriver struct {
	extraOpts []string
}

// NewRgSearchDriver creates a new RgSearchDriver with optional extra options
func NewRgSearchDriver(extraOpts []string) *RgSearchDriver {
	return &RgSearchDriver{
		extraOpts: extraOpts,
	}
}

// Name returns the name of the search driver
func (d *RgSearchDriver) Name() string {
	return "rg"
}

// IsAvailable checks if rg is available on the system
func (d *RgSearchDriver) IsAvailable() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}

// BuildCommand constructs the command to execute the search
func (d *RgSearchDriver) BuildCommand(options SearchOptions) (*exec.Cmd, error) {
	args := []string{
		"--line-number", // Show line numbers
		"--no-heading",  // Don't group matches by file
	}

	// Add case sensitivity option
	if options.IgnoreCase {
		args = append(args, "-i")
	}

	// Add literal search option (disable regex)
	if !options.UseRegex {
		args = append(args, "-F")
	}

	// Add any extra options from config
	args = append(args, d.extraOpts...)

	// Add search pattern and directory
	args = append(args, options.Query, options.RootDir)

	return exec.Command("rg", args...), nil
}

// ParseResults parses the output from the rg command
func (d *RgSearchDriver) ParseResults(stdout io.Reader) (<-chan SearchResult, error) {
	resultChan := make(chan SearchResult)

	// Regular expression to parse search results
	// This pattern works with rg output format: filename:line_number:matched_content
	re := regexp.MustCompile(`^([^:]+):(\d+):(.*)$`)

	go func() {
		defer close(resultChan)

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			matches := re.FindStringSubmatch(line)

			if len(matches) >= 4 {
				filePath := matches[1]
				lineNumber, _ := strconv.Atoi(matches[2])
				matchedLine := matches[3]

				resultChan <- SearchResult{
					FilePath:    filePath,
					LineNumber:  lineNumber,
					MatchedLine: matchedLine,
				}
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading search results: %v\n", err)
		}
	}()

	return resultChan, nil
}
