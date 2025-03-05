package main

import (
	"fmt"
	_ "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func main() {
	app := tview.NewApplication()

	// コマンドライン引数でディレクトリを指定
	var rootDir string
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	} else {
		rootDir, _ = os.Getwd()
	}

	// Create list view
	listView := tview.NewList()
	listView.ShowSecondaryText(false)
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetText(fmt.Sprintf("Select a file to view its content: %s", rootDir))

	// Load directory items
	loadDirectory(listView, rootDir, textView, "")

	listView.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		path := secondaryText
		fileInfo, err := os.Stat(path)
		if err != nil {
			log.Println(err)
			return
		}
		if !fileInfo.IsDir() {
			loadFileContent(textView, path)
		} else {
			textView.SetText("")
		}
	})

	// Layout
	flex := tview.NewFlex().
		AddItem(listView, 30, 1, true).
		AddItem(textView, 0, 2, false)

	if err := app.SetRoot(flex, true).SetFocus(listView).Run(); err != nil {
		panic(err)
	}
}

// loadDirectory loads directory content into a list view with tree-like formatting
func loadDirectory(listView *tview.List, path string, textView *tview.TextView, prefix string) {
	listView.Clear()

	walkDirectory(listView, path, textView, prefix)
}

func walkDirectory(listView *tview.List, path string, textView *tview.TextView, prefix string) {
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
			displayName = fmt.Sprintf("%s  %s", prefix, file.Name())
		}
		listView.AddItem(displayName, filePath, 0, func() {
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				log.Println(err)
				return
			}
			if fileInfo.IsDir() {
				loadDirectory(listView, filePath, textView, prefix+"  ")
			} else {
				loadFileContent(textView, filePath)
			}
		})
		if file.IsDir() {
			walkDirectory(listView, filePath, textView, prefix+"  ")
		}
	}
}

// loadFileContent loads and displays file content in the text view
func loadFileContent(textView *tview.TextView, path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		textView.SetText(fmt.Sprintf("[red]Error loading file: %v[/red]", err))
		return
	}
	textView.SetText(string(data))
}
