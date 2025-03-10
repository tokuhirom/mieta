package help_view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/tokuhirom/mieta/mieta/config"
	"log"
	"sort"
	"strings"
)

type HelpViewHandler func(view *HelpView)

func GetHelpKeymap(config *config.Config) (map[tcell.Key]HelpViewHandler, map[rune]HelpViewHandler) {
	keycodeKeymaps := map[tcell.Key]HelpViewHandler{
		tcell.KeyEscape: HelpHidePage,
		tcell.KeyEnter:  HelpHidePage,
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

	for key, funcName := range config.HelpKeyMap {
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
