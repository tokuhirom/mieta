package mieta

import (
	"github.com/BurntSushi/toml"
	"log"
	"os"
	"path/filepath"
)

type SearchConfig struct {
	// デフォルトの検索ドライバー
	Driver    string   `toml:"driver"`
	ExtraOpts []string `toml:"extra_opts"`
}

type Config struct {
	// シンタックスハイライトのスタイル
	ChromaStyle string `toml:"chroma_style"`

	// ハイライト処理を行うファイルサイズの上限（バイト）
	HighlightLimit int `toml:"highlight_limit"`

	// 検索関連の設定
	Search SearchConfig `toml:"search"`
}

// LoadConfig は設定ファイルを読み込みます
func LoadConfig() *Config {
	config := &Config{}

	// デフォルト値の設定
	config.ChromaStyle = "monokai"
	config.HighlightLimit = 1000000
	config.Search.Driver = "ag"

	// ユーザーホームディレクトリの設定ファイルを試す
	homeDir, err := os.UserHomeDir()
	if err == nil {
		// ~/.config/mieta/config.toml を探す
		configDir := filepath.Join(homeDir, ".config", "mieta")
		configPath := filepath.Join(configDir, "config.toml")

		if _, err := os.Stat(configPath); err == nil {
			// 設定ファイルが存在する場合は読み込む
			if _, err := toml.DecodeFile(configPath, config); err != nil {
				log.Printf("設定ファイルの読み込みエラー: %v", err)
			} else {
				log.Printf("設定ファイルを読み込みました: %s", configPath)
			}
		} else {
			// 設定ファイルが存在しない場合はデフォルト設定ファイルを作成
			if err := os.MkdirAll(configDir, 0755); err == nil {
				defaultConfig := `# MIETA 設定ファイル

# シンタックスハイライトのスタイル
# 利用可能なスタイル: monokai, github, vs, xcode, dracula, nord, solarized-dark, solarized-light
chroma_style = "monokai"

# ハイライト処理を行うファイルサイズの上限（バイト）
highlight_limit = 1000000

# 検索関連の設定
[search]
# 使用する検索ドライバー: "ag" または "rg"
driver = "ag"

# The Silver Searcher (ag) の設定
[search.ag]
# 追加のコマンドラインオプション
extra_opts = []

# ripgrep (rg) の設定
[search.rg]
# 追加のコマンドラインオプション
extra_opts = []
`
				if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
					log.Printf("デフォルト設定ファイルの作成に失敗しました: %v", err)
				} else {
					log.Printf("デフォルト設定ファイルを作成しました: %s", configPath)
				}
			}
		}
	}

	return config
}

// GetSearchDriver は設定に基づいて適切な検索ドライバーを返します
func (c *Config) GetSearchDriver() (string, []string) {
	driver := c.Search.Driver

	if !(driver == "ag" || driver == "rg") {
		log.Fatalf("Unknown searchdriver: %v", driver)
	}

	return driver, c.Search.ExtraOpts
}
