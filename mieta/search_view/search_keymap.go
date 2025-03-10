package search_view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/tokuhirom/mieta/mieta/config"
	"github.com/tokuhirom/mieta/mieta/keymap"
)

// config.go に追加

// SearchViewHandler は *search_view.SearchView を引数に取る関数型のエイリアスです
type SearchViewHandler func(view *SearchView)

var SearchFunctions = map[string]SearchViewHandler{
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

var DefaultKeyMap = map[string]string{
	"Esc":    "SearchExitView",
	"Up":     "SearchPreviousItem",
	"Down":   "SearchNextItem",
	"Ctrl-R": "SearchToggleRegex",
	"Ctrl-I": "SearchToggleCase",

	"S": "SearchFocusInput",
	"w": "SearchPreviousItem",
	"s": "SearchNextItem",
	"q": "SearchExitView",
	"h": "SearchScrollLeft",
	"j": "SearchScrollDown",
	"k": "SearchScrollUp",
	"l": "SearchScrollRight",
	"e": "SearchEdit",
	"G": "SearchScrollToEnd",
	"H": "SearchDecreaseLeftWidth",
	"L": "SearchIncreaseLeftWidth",
}

func GetSearchKeymap(config *config.Config) (map[string]string, map[tcell.Key]SearchViewHandler, map[rune]SearchViewHandler) {
	return keymap.ProcessKeymap("search", DefaultKeyMap, config.SearchKeyMap, SearchFunctions)
}
