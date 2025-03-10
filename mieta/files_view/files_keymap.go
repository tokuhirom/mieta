package files_view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/tokuhirom/mieta/mieta/config"
	"log"
	"sort"
	"strings"
)

// FilesViewHandler は *FilesView を引数に取る関数型のエイリアスです
type FilesViewHandler func(view *FilesView)

func GetFilesKeymap(config *config.Config) (map[tcell.Key]FilesViewHandler, map[rune]FilesViewHandler) {
	keycodeKeymaps := map[tcell.Key]FilesViewHandler{}

	runeKeymaps := map[rune]FilesViewHandler{
		'j': FilesScrollDown,
		'k': FilesScrollUp,
		'q': FilesQuit,
		'?': FilesShowHelp,
		'w': FilesMoveUp,
		's': FilesMoveDown,
		'S': FilesShowSearch,
		'e': FilesEdit,
		'a': FilesNavigateUp,
		'd': FilesExpand,
		' ': FilesScrollPageDown,
		'H': FilesDecreaseTreeWidth,
		'L': FilesIncreaseTreeWidth,
		'f': FilesEnterFindMode,
	}

	FilesFunctions := map[string]FilesViewHandler{
		"FilesScrollDown":        FilesScrollDown,
		"FilesScrollUp":          FilesScrollUp,
		"FilesQuit":              FilesQuit,
		"FilesShowHelp":          FilesShowHelp,
		"FilesMoveUp":            FilesMoveUp,
		"FilesMoveDown":          FilesMoveDown,
		"FilesShowSearch":        FilesShowSearch,
		"FilesEdit":              FilesEdit,
		"FilesNavigateUp":        FilesNavigateUp,
		"FilesExpand":            FilesExpand,
		"FilesScrollPageDown":    FilesScrollPageDown,
		"FilesDecreaseTreeWidth": FilesDecreaseTreeWidth,
		"FilesIncreaseTreeWidth": FilesIncreaseTreeWidth,
		"FilesEnterFindMode":     FilesEnterFindMode,
		"FilesExitFindMode":      FilesExitFindMode,
	}

	// 設定ファイルからのカスタムキーマップ
	keyName2keyCode := make(map[string]tcell.Key)
	for keyCode, keyName := range tcell.KeyNames {
		keyName2keyCode[strings.ToLower(keyName)] = keyCode
	}

	// FilesKeyMap が設定されていれば処理
	for key, funcName := range config.FilesKeyMap {
		fun, ok := FilesFunctions[funcName]
		if !ok {
			// 利用可能な関数名のリストを作成
			availableFunctions := make([]string, 0, len(FilesFunctions))
			for fname := range FilesFunctions {
				availableFunctions = append(availableFunctions, fname)
			}
			// ソートして読みやすくする
			sort.Strings(availableFunctions)
			log.Fatalf("Unknown functions for [keymap.Files]: %s\nAvailable functions: %s",
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
