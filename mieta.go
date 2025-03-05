package main

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/chroma/quick"
	"github.com/gdamore/tcell/v2"
	_ "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"unicode/utf8"
)

func main() {
	// Set up logging to a file if MIETA_DEBUG is set
	if logFilePath := os.Getenv("MIETA_DEBUG"); logFilePath != "" {
		logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		defer func(logFile *os.File) {
			err := logFile.Close()
			if err != nil {
				log.Fatalf("Failed to close log file: %v", err)
			}
		}(logFile)
		log.SetOutput(logFile)
	}

	// コマンドライン引数でディレクトリを指定
	var rootDir string
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	} else {
		rootDir, _ = os.Getwd()
	}

	mieta := NewMieta()
	mieta.Run(rootDir)
}

type Mieta struct {
	HighlightLimit int
	MaxLines       *int
}

func NewMieta() *Mieta {
	maxLines := 100
	return &Mieta{
		HighlightLimit: 100 * 1024,
		MaxLines:       &maxLines,
	}
}

func (m *Mieta) Run(rootDir string) {
	app := tview.NewApplication()

	// Create list view
	listView := tview.NewList()
	listView.ShowSecondaryText(false)
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetText(fmt.Sprintf("Select a file to view its content: %s", rootDir))
	if m.MaxLines != nil {
		textView.SetMaxLines(*m.MaxLines)
	}

	// Load directory items
	go m.loadDirectory(listView, rootDir, textView, "")

	listView.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		path := secondaryText
		fileInfo, err := os.Stat(path)
		if err != nil {
			log.Println(err)
			return
		}
		if !fileInfo.IsDir() {
			m.loadFileContent(textView, path)
		} else {
			textView.SetText("")
		}
	})

	// Layout
	flex := tview.NewFlex().
		AddItem(listView, 30, 1, true).
		AddItem(tview.NewBox(), 2, 0, false).
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
		case 'q':
			app.Stop()
		case rune(tcell.KeyCtrlH):
			_, _, width, _ := listView.GetRect()
			flex.ResizeItem(listView, width-2, 1)
		case rune(tcell.KeyCtrlL):
			_, _, width, _ := listView.GetRect()
			flex.ResizeItem(listView, width+2, 1)
		}
		return event
	})

	if err := app.SetRoot(flex, true).SetFocus(listView).Run(); err != nil {
		panic(err)
	}
}

// loadDirectory loads directory content into a list view with tree-like formatting
func (m *Mieta) loadDirectory(listView *tview.List, path string, textView *tview.TextView, prefix string) {
	listView.Clear()

	m.walkDirectory(listView, path, textView, prefix)
}

func (m *Mieta) walkDirectory(listView *tview.List, path string, textView *tview.TextView, prefix string) {
	files, err := ioutil.ReadDir(path)
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
				m.loadFileContent(textView, filePath)
			}
		})
		if file.IsDir() {
			m.walkDirectory(listView, filePath, textView, prefix+"  ")
		}
	}
}

// loadFileContent loads and displays file content in the text view with syntax highlighting
func (m *Mieta) loadFileContent(textView *tview.TextView, path string) {
	log.Printf("Loading %s", path)
	textView.SetText(fmt.Sprintf("Loading %s", path))
	content, err := ioutil.ReadFile(path)
	if err != nil {
		textView.SetText(fmt.Sprintf("[red]Error loading file: %v", err))
		return
	}
	log.Printf("Finished reading %s(%d bytes)", path, len(content))

	if !utf8.Valid(content) {
		textView.SetText("[red]Binary")
		return
	}

	// 100KB よりも大きいファイルは重くなるのでハイライトしない
	highlightLimit := m.HighlightLimit
	if len(content) > highlightLimit {
		log.Printf("File is too large to highlight: %s(%d bytes > %d bytes)", path,
			len(content), highlightLimit)
		textView.SetText(string(content))
		return
	}

	// Detect syntax highlighting based on file extension
	fileExt := filepath.Ext(path)

	var highlighted bytes.Buffer
	if err := quick.Highlight(&highlighted, string(content), fileExt, "terminal", "monokai"); err == nil {
		textView.SetText(tview.TranslateANSI(highlighted.String()))
	} else {
		textView.SetText(string(content))
	}
	log.Printf("Loaded %s", path)
}
