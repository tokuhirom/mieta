package mieta

import (
	"github.com/BurntSushi/toml"
	"github.com/gdamore/tcell/v2"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

	// 外部エディタの設定
	Editor string `toml:"editor"`

	// 検索関連の設定
	Search SearchConfig `toml:"search"`

	// Keymaps
	HelpKeyMap map[string]string `toml:"keymap.help"`
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

	// エディタの設定がない場合は環境変数 EDITOR を使用
	if config.Editor == "" {
		config.Editor = os.Getenv("EDITOR")
		// それでも設定がない場合はデフォルト値を設定
		if config.Editor == "" {
			// プラットフォームに応じてデフォルト値を設定
			if _, err := os.Stat("/usr/bin/vim"); err == nil {
				config.Editor = "vim"
			} else if _, err := os.Stat("/usr/bin/nano"); err == nil {
				config.Editor = "nano"
			} else {
				// 最終的なフォールバック
				config.Editor = "vi"
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

type KeyOrRune struct {
	Key  tcell.Key
	Rune int32
}

type HelpViewHandler func(view *HelpView)

func (c *Config) GetHelpKeymap() (map[tcell.Key]HelpViewHandler, map[rune]HelpViewHandler) {
	keycodeKeymaps := map[tcell.Key]HelpViewHandler{
		tcell.KeyEscape: HelpHidePage,
	}

	runeKeymaps := map[rune]HelpViewHandler{
		'j': HelpScrollDown,
		'k': HelpScrollUp,
	}

	helpFunctions := map[string]HelpViewHandler{
		"HelpScrollDown": HelpScrollDown,
		"HelpScrollUp":   HelpScrollUp,
		"HelpHidePage":   HelpHidePage,
	}

	keyName2keyCode := make(map[string]tcell.Key)
	for keyCode, keyName := range tcell.KeyNames {
		keyName2keyCode[strings.ToLower(keyName)] = keyCode
	}

	for key, funcName := range c.HelpKeyMap {
		fun, ok := helpFunctions[funcName]
		if !ok {
			// 利用可能な関数名のリストを作成
			availableFunctions := make([]string, 0, len(helpFunctions))
			for fname := range helpFunctions {
				availableFunctions = append(availableFunctions, fname)
			}
			// ソートして読みやすくする
			sort.Strings(availableFunctions)
			log.Fatalf("Unknown functions for [keymap.help]: %s\nAvailable functions: %v", funcName, availableFunctions)
		}

		if len(key) == 1 {
			// handle as rune
			runeKeymaps[rune(key[0])] = fun
		} else {
			keyCode, ok := keyName2keyCode[key]
			if !ok {
				// 利用可能なキー名のリストを作成
				availableKeys := make([]string, 0, len(keyName2keyCode))
				for keyName := range keyName2keyCode {
					availableKeys = append(availableKeys, keyName)
				}
				// ソートして読みやすくする
				sort.Strings(availableKeys)

				log.Fatalf("Unknown keyname %s in configuration file.\nAvailable key names are: %s",
					key, strings.Join(availableKeys, ", "))
			}
			keycodeKeymaps[keyCode] = fun
		}
	}

	return keycodeKeymaps, runeKeymaps
}
