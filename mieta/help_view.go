package mieta

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const HelpMessage = `Help

# MIETA Help

## Navigation
- w/s: Move up/down in tree or list
- j/k: Scroll preview up/down
- a: Navigate up/collapse directory
- d: Expand directory
- Space: Scroll preview down one page
- H/L: Adjust panel widths

## Search
- /: Open search view
- f: Find files by name
- Ctrl-R: Toggle regex search
- Ctrl-I: Toggle case sensitivity
- h/j/k/l: Navigate in preview
- G: Go to end of preview

## Other
- q: Quit
- ?: Show/hide help
`

type HelpView struct {
	Flex        *tview.Flex
	CloseButton *tview.Button
}

func NewHelpView(pages *tview.Pages) *HelpView {
	// modal always show the text as align-center.
	// it's hard coded. so, we need to create a new flex layout manually.
	// we can't use tview.NewModal() because it's not flexible.
	width := 40
	height := 10

	textView := tview.NewTextView().
		SetText(HelpMessage)
	textView.SetBorder(true)
	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case rune(tcell.KeyEscape):
			pages.HidePage("help")
		}
		return event
	})

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

	return &HelpView{
		Flex:        flex,
		CloseButton: closeButton,
	}
}
