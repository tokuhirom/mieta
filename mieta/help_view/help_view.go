package help_view

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/tokuhirom/mieta/mieta/config"
	"github.com/tokuhirom/mieta/mieta/files_view"
	"github.com/tokuhirom/mieta/mieta/search_view"
)

type HelpView struct {
	Flex        *tview.Flex
	CloseButton *tview.Button
	TextView    *tview.TextView
	Pages       *tview.Pages
}

func helpFoo(prefix string, keymap map[string]string) string {
	buf := ""
	for key, val := range keymap {
		if key == " " {
			key = "` `"
		}
		buf += fmt.Sprintf("\n%8v: %v", key, val[len(prefix):])
	}
	return buf
}

func generateHelpMessage(config *config.Config) string {
	buf := ""

	keymap, _, _ := files_view.GetFilesKeymap(config)
	buf += "# Files\n" + helpFoo("Files", keymap)

	keymap, _, _ = search_view.GetSearchKeymap(config)
	buf += "\n\n# Search\n" + helpFoo("Search", keymap)

	keymap, _, _ = GetHelpKeymap(config)
	buf += "\n\n# Help\n" + helpFoo("Help", keymap)

	return buf
}

func NewHelpView(pages *tview.Pages, config *config.Config) *HelpView {
	// modal always show the text as align-center.
	// it's hard coded. so, we need to create a new flex layout manually.
	// we can't use tview.NewModal() because it's not flexible.
	width := 40
	height := 10

	helpMessage := generateHelpMessage(config)

	textView := tview.NewTextView().
		SetText(helpMessage)
	textView.SetBorder(true)

	closeButton := tview.NewButton("OK")
	closeButton.SetSelectedFunc(func() {
		pages.HidePage("help")
	})

	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(textView, height, 1, true).
			AddItem(closeButton, 1, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)

	helpView := &HelpView{
		Flex:        flex,
		TextView:    textView,
		Pages:       pages,
		CloseButton: closeButton,
	}

	_, keycodeKeymap, runeKeymap := GetHelpKeymap(config)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		handler, ok := keycodeKeymap[event.Key()]
		if ok {
			handler(helpView)
			return nil
		}

		if event.Key() == tcell.KeyRune {
			handler, ok := runeKeymap[event.Rune()]
			if ok {
				handler(helpView)
				return nil
			}
		}

		return event
	})

	return helpView
}
