package mieta

type Config struct {
	HighlightLimit int
	ChromaStyle    string
}

func LoadConfig() *Config {
	return &Config{
		HighlightLimit: 100 * 1024,
		ChromaStyle:    "dracula",
	}
}
