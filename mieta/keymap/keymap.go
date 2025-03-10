package keymap

import (
	"github.com/gdamore/tcell/v2"
	"log"
	"sort"
	"strings"
)

func GetKeyName2KeyCode() map[string]tcell.Key {
	keyName2keyCode := make(map[string]tcell.Key)
	for keyCode, keyName := range tcell.KeyNames {
		keyName2keyCode[strings.ToLower(keyName)] = keyCode
	}
	return keyName2keyCode
}

func ProcessKeymap[T any](
	defaultKeymap map[string]string,
	userKeymap map[string]string,
	functionMap map[string]T,
) (map[string]string, map[tcell.Key]T, map[rune]T) {
	keycodeKeymaps := map[tcell.Key]T{}
	runeKeymaps := map[rune]T{}

	keyName2keyCode := GetKeyName2KeyCode()

	mergedKeyMap := make(map[string]string)
	for k, v := range defaultKeymap {
		mergedKeyMap[k] = v
	}
	for k, v := range userKeymap {
		mergedKeyMap[k] = v
	}

	// FilesKeyMap が設定されていれば処理
	for key, funcName := range mergedKeyMap {
		fun, ok := functionMap[funcName]
		if !ok {
			// 利用可能な関数名のリストを作成
			availableFunctions := make([]string, 0, len(functionMap))
			for fname := range functionMap {
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

	return mergedKeyMap, keycodeKeymaps, runeKeymaps
}
