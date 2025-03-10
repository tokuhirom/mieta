package main

import (
	"github.com/gdamore/tcell/v2"
	_ "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/tokuhirom/mieta/mieta/config"
	"github.com/tokuhirom/mieta/mieta/files_view"
	"github.com/tokuhirom/mieta/mieta/help_view"
	"github.com/tokuhirom/mieta/mieta/search_view"
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

	config := config.LoadConfig()
	run(rootDir, config)
}

func run(rootDir string, config *config.Config) {
	app := tview.NewApplication()

	pages := tview.NewPages()
	helpView := help_view.NewHelpView(pages, config)
	mainView := files_view.NewFilesView(rootDir, config, app, pages)
	pages.AddPage("files", mainView.Flex, true, true)
	pages.AddPage("help", helpView.Flex, true, false)
	searchView := search_view.NewSearchView(app, config, mainView, pages, rootDir)
	pages.AddPage("search", searchView.Flex, true, false)

	pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			log.Printf("Hide help page")
			pages.HidePage("help")
		}
		return event
	})

	pages.SetChangedFunc(func() {
		p := pages.GetPageNames(true)
		for _, name := range p {
			log.Printf("Shown page: %s", name)
			if name == "files" {
				app.SetFocus(mainView.TreeView)
			} else if name == "search" {
				app.SetFocus(searchView.InputField)
			} else if name == "help" {
				app.SetFocus(helpView.CloseButton)
			}
		}
	})

	if err := app.SetRoot(pages, true).SetFocus(mainView.TreeView).Run(); err != nil {
		panic(err)
	}
}
