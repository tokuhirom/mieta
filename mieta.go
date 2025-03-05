package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/rivo/tview"
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
		SetText("Select a file to view its content")

	// Load directory items
	go loadDirectory(listView, rootDir, textView)

	listView.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		path := secondaryText
		fileInfo, err := os.Stat(path)
		if err != nil {
			log.Printf("%s, %v", path, err)
			return
		}
		if fileInfo.IsDir() {
			textView.SetText("")
		} else {
			loadFileContent(textView, path)
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

// loadDirectory loads directory content into a list view
func loadDirectory(listView *tview.List, path string, textView *tview.TextView) {
	listView.Clear()

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
		listView.AddItem(file.Name(), filePath, 0, func() {
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				log.Printf("Cannot load %v", err)
				return
			}
			if !fileInfo.IsDir() {
				loadFileContent(textView, filePath)
			}
		})
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
