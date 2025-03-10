package main

import (
	"github.com/gdamore/tcell/v2"
	_ "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/tokuhirom/mieta/mieta"
	"io"
	"log"
	"os"
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
	} else {
		log.SetOutput(io.Discard)
	}

	// コマンドライン引数でディレクトリを指定
	var rootDir string
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	} else {
		rootDir, _ = os.Getwd()
	}

	config := mieta.LoadConfig()
	run(rootDir, config)
}

func run(rootDir string, config *mieta.Config) {
	app := tview.NewApplication()

	pages := tview.NewPages()
	helpView := mieta.NewHelpView(pages, config)
	mainView := mieta.NewMainView(rootDir, config, app, pages, helpView)
	pages.AddPage("background", mainView.Flex, true, true)
	pages.AddPage("help", helpView.Flex, true, false)
	searchView := mieta.NewSearchView(app, config, mainView, pages, rootDir)
	pages.AddPage("search", searchView.Flex, true, false)

	pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			log.Printf("Hide help page")
			pages.HidePage("help")
		}
		return event
	})

	if err := app.SetRoot(pages, true).SetFocus(mainView.TreeView).Run(); err != nil {
		panic(err)
	}
}
