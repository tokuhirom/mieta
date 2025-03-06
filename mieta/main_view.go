package mieta

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/chroma/quick"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"log"
	"os"
	"path/filepath"
	"unicode/utf8"
)

type MainView struct {
	Config   *Config
	Flex     *tview.Flex
	ListView *tview.List
}

func NewMainView(rootDir string, config *Config, app *tview.Application, pages *tview.Pages, helpView *HelpView) *MainView {
	// Create list view
	listView := tview.NewList()
	listView.SetBorder(true)
	listView.ShowSecondaryText(false)
	listView.SetBorderColor(tcell.ColorDarkSlateGray)

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetText(fmt.Sprintf("Select a file to view its content: %s", rootDir))
	textView.SetBorder(true)
	textView.SetBorderColor(tcell.ColorDarkSlateGray)
	textView.SetBorderPadding(0, 0, 1, 1)
	if config.MaxLines != nil {
		textView.SetMaxLines(*config.MaxLines)
	}

	// Load directory items
	go loadDirectory(config, listView, rootDir, textView, "")

	listView.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		path := secondaryText
		fileInfo, err := os.Stat(path)
		if err != nil {
			log.Println(err)
			return
		}
		if !fileInfo.IsDir() {
			loadFileContent(config, textView, path)
		} else {
			textView.SetText("")
		}
	})

	// Layout
	flex := tview.NewFlex().
		AddItem(listView, 30, 1, true).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(textView, 0, 2, false)

	// キーバインド設定
	listView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'w':
			index := listView.GetCurrentItem()
			if index > 0 {
				listView.SetCurrentItem(index - 1)
			}
		case 's':
			index := listView.GetCurrentItem()
			if index < listView.GetItemCount()-1 {
				listView.SetCurrentItem(index + 1)
			}
		case 'j':
			row, col := textView.GetScrollOffset()
			textView.ScrollTo(row+9, col)
		case 'k':
			row, col := textView.GetScrollOffset()
			textView.ScrollTo(row-9, col)
		case '?':
			pages.ShowPage("help")
			app.SetFocus(helpView.CloseButton)
		case ' ':
			row, col := textView.GetScrollOffset()
			x, y, width, height := textView.GetRect()
			log.Printf("row: %d, col: %d, lines: %d, height: %d, (%d,%d,%d,%d)",
				row, col, textView.GetOriginalLineCount(),
				textView.GetFieldHeight(),
				x, y, width, height,
			)
			if textView.GetOriginalLineCount() <= row+height {
				index := listView.GetCurrentItem()
				if index < listView.GetItemCount()-1 {
					listView.SetCurrentItem(index + 1)
				}
			} else {
				textView.ScrollTo(row+9, col)
			}
		case 'H':
			_, _, width, _ := listView.GetRect()
			flex.ResizeItem(listView, width-2, 1)
		case 'L':
			_, _, width, _ := listView.GetRect()
			flex.ResizeItem(listView, width+2, 1)
		}
		return event
	})

	return &MainView{
		Config:   config,
		Flex:     flex,
		ListView: listView,
	}
}

// loadDirectory loads directory content into a list view with tree-like formatting
func loadDirectory(config *Config, listView *tview.List, path string, textView *tview.TextView, prefix string) {
	listView.Clear()

	walkDirectory(config, listView, path, textView, prefix)
}

func walkDirectory(config *Config, listView *tview.List, path string, textView *tview.TextView, prefix string) {
	files, err := os.ReadDir(path)
	if err != nil {
		log.Println(err)
		return
	}

	for _, file := range files {
		if file.Name() == ".git" {
			continue
		}
		filePath := filepath.Join(path, file.Name())

		var displayName string
		if file.IsDir() {
			displayName = fmt.Sprintf("%s+ %s", prefix, file.Name())
		} else {
			displayName = fmt.Sprintf("%s- %s", prefix, file.Name())
		}
		listView.AddItem(displayName, filePath, 0, func() {
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				log.Println(err)
				return
			}
			if fileInfo.IsDir() {
				//loadDirectory(listView, filePath, textView, prefix+"  ")
			} else {
				loadFileContent(config, textView, filePath)
			}
		})
		if file.IsDir() {
			walkDirectory(config, listView, filePath, textView, prefix+"  ")
		}
	}
}

// loadFileContent loads and displays file content in the text view with syntax highlighting
func loadFileContent(config *Config, textView *tview.TextView, path string) {
	log.Printf("Loading %s", path)
	textView.SetText(fmt.Sprintf("Loading %s", path))
	content, err := os.ReadFile(path)
	if err != nil {
		textView.SetText(fmt.Sprintf("[red]Error loading file: %v", err))
		return
	}
	log.Printf("Finished reading %s(%d bytes)", path, len(content))

	if !utf8.Valid(content) {
		textView.SetText("[red]Binary")
		return
	}

	highlightLimit := config.HighlightLimit
	if len(content) > highlightLimit {
		log.Printf("File is too large to highlight: %s(%d bytes > %d bytes)", path,
			len(content), highlightLimit)
		textView.SetText(string(content))
		return
	}

	// Detect syntax highlighting based on file extension
	fileExt := filepath.Ext(path)

	var highlighted bytes.Buffer
	if err := quick.Highlight(&highlighted, string(content), fileExt, "terminal", config.ChromaStyle); err == nil {
		textView.SetText(tview.TranslateANSI(highlighted.String()))
	} else {
		textView.SetText(string(content))
	}
	log.Printf("Loaded %s", path)
}
