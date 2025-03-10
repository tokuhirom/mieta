package search

import (
	"io"
	"os/exec"
)

// SearchOptions contains common options for search operations
type SearchOptions struct {
	Query      string
	RootDir    string
	UseRegex   bool
	IgnoreCase bool
}

// SearchDriver defines the interface for search implementations
type SearchDriver interface {
	// Name returns the name of the search driver
	Name() string

	// IsAvailable checks if the search tool is available on the system
	IsAvailable() bool

	// BuildCommand constructs the command to execute the search
	BuildCommand(options SearchOptions) (*exec.Cmd, error)

	// ParseResults parses the output from the search command
	// Returns a channel that will receive search results
	ParseResults(stdout io.Reader) (<-chan SearchResult, error)
}

// SearchResult represents a single search result
type SearchResult struct {
	FilePath    string
	LineNumber  int
	MatchedLine string
}

// GetSearchDriver returns the appropriate search driver based on the name
func GetSearchDriver(name string, extraOpts []string) SearchDriver {
	switch name {
	case "ag":
		return NewAgSearchDriver(extraOpts)
	case "rg":
		return NewRgSearchDriver(extraOpts)
	default:
		// Default to ag if the specified driver is not supported
		return NewAgSearchDriver(extraOpts)
	}
}
