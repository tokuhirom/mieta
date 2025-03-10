package search_view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/tokuhirom/mieta/mieta/config"
	"log"
	"sort"
	"strings"
)

// config.go に追加

// SearchViewHandler は *search_view.SearchView を引数に取る関数型のエイリアスです
type SearchViewHandler func(view *SearchView)

func GetSearchKeymap(config *config.Config) (map[tcell.Key]SearchViewHandler, map[rune]SearchViewHandler) {
	keycodeKeymaps := map[tcell.Key]SearchViewHandler{
		tcell.KeyEscape: SearchExitView,
		tcell.KeyUp:     SearchPreviousItem,
		tcell.KeyDown:   SearchNextItem,
		tcell.KeyCtrlR:  SearchToggleRegex,
		tcell.KeyCtrlI:  SearchToggleCase,
	}

	runeKeymaps := map[rune]SearchViewHandler{
		'S': SearchFocusInput,
		'w': SearchPreviousItem,
		's': SearchNextItem,
		'q': SearchExitView,
		'h': SearchScrollLeft,
		'j': SearchScrollDown,
		'k': SearchScrollUp,
		'l': SearchScrollRight,
		'e': SearchEdit,
		'G': SearchScrollToEnd,
		'H': SearchDecreaseLeftWidth,
		'L': SearchIncreaseLeftWidth,
	}

	searchFunctions := map[string]SearchViewHandler{
		"SearchFocusInput":        SearchFocusInput,
		"SearchPreviousItem":      SearchPreviousItem,
		"SearchNextItem":          SearchNextItem,
		"SearchExitView":          SearchExitView,
		"SearchScrollLeft":        SearchScrollLeft,
		"SearchScrollDown":        SearchScrollDown,
		"SearchScrollUp":          SearchScrollUp,
		"SearchScrollRight":       SearchScrollRight,
		"SearchEdit":              SearchEdit,
		"SearchScrollToEnd":       SearchScrollToEnd,
		"SearchDecreaseLeftWidth": SearchDecreaseLeftWidth,
		"SearchIncreaseLeftWidth": SearchIncreaseLeftWidth,
		"SearchToggleRegex":       SearchToggleRegex,
		"SearchToggleCase":        SearchToggleCase,
	}

	// 設定ファイルからのカスタムキーマップ
	keyName2keyCode := make(map[string]tcell.Key)
	for keyCode, keyName := range tcell.KeyNames {
		keyName2keyCode[strings.ToLower(keyName)] = keyCode
	}

	// SearchKeyMap が設定されていれば処理
	for key, funcName := range config.SearchKeyMap {
		fun, ok := searchFunctions[funcName]
		if !ok {
			// 利用可能な関数名のリストを作成
			availableFunctions := make([]string, 0, len(searchFunctions))
			for fname := range searchFunctions {
				availableFunctions = append(availableFunctions, fname)
			}
			// ソートして読みやすくする
			sort.Strings(availableFunctions)
			log.Fatalf("Unknown functions for [keymap.search]: %s\nAvailable functions: %s",
				funcName, strings.Join(availableFunctions, ", "))
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
