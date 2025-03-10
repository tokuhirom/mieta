package help_view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/tokuhirom/mieta/mieta/config"
	"github.com/tokuhirom/mieta/mieta/keymap"
)

type HelpViewHandler func(view *HelpView)

var HelpFunctions = map[string]HelpViewHandler{
	"HelpScrollDown": HelpScrollDown,
	"HelpScrollUp":   HelpScrollUp,
	"HelpHidePage":   HelpHidePage,
}

var DefaultKeyMap = map[string]string{
	"Esc":   "HelpHidePage",
	"Enter": "HelpHidePage",
	"j":     "HelpScrollDown",
	"k":     "HelpScrollUp",
}

func GetHelpKeymap(config *config.Config) (map[string]string, map[tcell.Key]HelpViewHandler, map[rune]HelpViewHandler) {
	return keymap.ProcessKeymap("help", DefaultKeyMap, config.HelpKeyMap, HelpFunctions)
}
