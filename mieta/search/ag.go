package search

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
)

// AgSearchDriver implements the SearchDriver interface for The Silver Searcher (ag)
type AgSearchDriver struct {
	extraOpts []string
}

// NewAgSearchDriver creates a new AgSearchDriver with optional extra options
func NewAgSearchDriver(extraOpts []string) *AgSearchDriver {
	return &AgSearchDriver{
		extraOpts: extraOpts,
	}
}

// Name returns the name of the search driver
func (d *AgSearchDriver) Name() string {
	return "ag"
}

// IsAvailable checks if ag is available on the system
func (d *AgSearchDriver) IsAvailable() bool {
	_, err := exec.LookPath("ag")
	return err == nil
}

// BuildCommand constructs the command to execute the search
func (d *AgSearchDriver) BuildCommand(options SearchOptions) (*exec.Cmd, error) {
	args := []string{}

	// Add case sensitivity option
	if options.IgnoreCase {
		args = append(args, "-i")
	}

	// Add literal search option (disable regex)
	if !options.UseRegex {
		args = append(args, "-Q")
	}

	// Add any extra options from config
	args = append(args, d.extraOpts...)

	// Add search pattern and directory
	args = append(args, options.Query, options.RootDir)

	return exec.Command("ag", args...), nil
}

// ParseResults parses the output from the ag command
func (d *AgSearchDriver) ParseResults(stdout io.Reader) (<-chan SearchResult, error) {
	resultChan := make(chan SearchResult)

	// Regular expression to parse search results
	// This pattern works with ag output format: filename:line_number:matched_content
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
			// Just log the error, we can't return it from this goroutine
			fmt.Printf("Error reading search results: %v\n", err)
		}
	}()

	return resultChan, nil
}
