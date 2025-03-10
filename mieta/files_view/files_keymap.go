package files_view

import (
	"github.com/gdamore/tcell/v2"
	"github.com/tokuhirom/mieta/mieta/config"
	"github.com/tokuhirom/mieta/mieta/keymap"
)

type FilesViewHandler func(view *FilesView)

var FilesFunctions = map[string]FilesViewHandler{
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
}

var DefaultKeyMap = map[string]string{
	"j": "FilesScrollDown",
	"k": "FilesScrollUp",
	"q": "FilesQuit",
	"?": "FilesShowHelp",
	"w": "FilesMoveUp",
	"s": "FilesMoveDown",
	"S": "FilesShowSearch",
	"e": "FilesEdit",
	"a": "FilesNavigateUp",
	"d": "FilesExpand",
	" ": "FilesScrollPageDown",
	"H": "FilesDecreaseTreeWidth",
	"L": "FilesIncreaseTreeWidth",
	"f": "FilesEnterFindMode",
}

func GetFilesKeymap(config *config.Config) (map[string]string, map[tcell.Key]FilesViewHandler, map[rune]FilesViewHandler) {
	return keymap.ProcessKeymap("files", DefaultKeyMap, config.FilesKeyMap, FilesFunctions)
}
