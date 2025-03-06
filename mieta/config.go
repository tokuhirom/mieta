package mieta

type Config struct {
	HighlightLimit int
	MaxLines       *int
	ChromaStyle    string
}

func LoadConfig() *Config {
	maxLines := 100
	return &Config{
		HighlightLimit: 100 * 1024,
		ChromaStyle:    "dracula",
		MaxLines:       &maxLines,
	}
}
