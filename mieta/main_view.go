package mieta

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/chroma/quick"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	ListMode = iota
	FindingMode
)

type MainView struct {
	Application      *tview.Application
	Mode             int
	Config           *Config
	Flex             *tview.Flex
	ListView         *tview.List
	PreviewPages     *tview.Pages
	PreviewImageView *tview.Image
	PreviewTextView  *tview.TextView
	// 検索モードに入る前に選択されていたアイテム
	ItemBeforeFinding int
	// 検索キーワード
	FindingKeyword     string
	CurrentLoadingFile string
}

func NewMainView(rootDir string, config *Config, app *tview.Application, pages *tview.Pages, helpView *HelpView) *MainView {
	// Create list view
	listView := tview.NewList()
	listView.SetBorder(true)
	listView.ShowSecondaryText(false)
	listView.SetBorderColor(tcell.ColorDarkSlateGray)

	previewTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true)
	previewTextView.SetBorder(true)
	previewTextView.SetBorderColor(tcell.ColorDarkSlateGray)
	previewTextView.SetBorderPadding(0, 0, 1, 1)

	previewImageView := tview.NewImage()
	previewImageView.SetBorder(true)
	previewImageView.SetBorderColor(tcell.ColorDarkSlateGray)

	previewPages := tview.NewPages()
	previewPages.AddPage("text", previewTextView, true, true)
	previewPages.AddPage("image", previewImageView, true, false)

	// Layout
	flex := tview.NewFlex().
		AddItem(listView, 30, 1, true).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(previewPages, 0, 2, false)

	mainView := &MainView{
		Application:      app,
		Config:           config,
		Flex:             flex,
		ListView:         listView,
		PreviewPages:     previewPages,
		PreviewTextView:  previewTextView,
		PreviewImageView: previewImageView,
	}

	// キーバインド設定
	listView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch mainView.Mode {
		case FindingMode:
			log.Printf("FindingMode: %v", event.Key())
			switch event.Key() {
			case tcell.KeyEscape:
				listView.SetTitle("")
				mainView.Mode = ListMode
				listView.SetCurrentItem(mainView.ItemBeforeFinding)
				mainView.FindingKeyword = ""
			case tcell.KeyEnter:
				listView.SetTitle("")
				mainView.Mode = ListMode
				mainView.FindingKeyword = ""
			case tcell.KeyDEL:
				if len(mainView.FindingKeyword) > 0 {
					mainView.FindingKeyword = mainView.FindingKeyword[:len(mainView.FindingKeyword)-1]
					mainView.ListView.SetTitle(mainView.FindingKeyword)
					mainView.findByKeyword()
				}
			case tcell.KeyRune:
				mainView.FindingKeyword += string(event.Rune())
				mainView.ListView.SetTitle(mainView.FindingKeyword)
				mainView.findByKeyword()
			}
		case ListMode:
			if event.Key() == tcell.KeyRune {
				switch event.Rune() {
				case 'f':
					// goto find file mode
					listView.SetTitle("🔎")
					mainView.Mode = FindingMode
					mainView.ItemBeforeFinding = listView.GetCurrentItem()
					mainView.FindingKeyword = ""
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
					row, col := previewTextView.GetScrollOffset()
					previewTextView.ScrollTo(row+9, col)
				case 'k':
					row, col := previewTextView.GetScrollOffset()
					previewTextView.ScrollTo(row-9, col)
				case 'q':
					app.Stop()
				case '?':
					pages.ShowPage("help")
					app.SetFocus(helpView.CloseButton)
				case ' ':
					row, col := previewTextView.GetScrollOffset()
					x, y, width, height := previewTextView.GetRect()
					log.Printf("row: %d, col: %d, lines: %d, height: %d, (%d,%d,%d,%d)",
						row, col, previewTextView.GetOriginalLineCount(),
						previewTextView.GetFieldHeight(),
						x, y, width, height,
					)
					if previewTextView.GetOriginalLineCount() <= row+height {
						log.Printf("Scroll to the end... open the next item")
						index := listView.GetCurrentItem()
						if index < listView.GetItemCount()-1 {
							log.Printf("Select new item: %d to %d",
								index, index+1)
							listView.SetCurrentItem(index + 1)
						} else {
							log.Printf("Already at the end")
						}
					} else {
						log.Printf("Already at the end")
						previewTextView.ScrollTo(row+9, col)
					}
				case 'H':
					_, _, width, _ := listView.GetRect()
					flex.ResizeItem(listView, width-2, 1)
				case 'L':
					_, _, width, _ := listView.GetRect()
					flex.ResizeItem(listView, width+2, 1)
				}
			}
		}
		return event
	})

	// Load directory items
	go mainView.loadDirectory(config, listView, rootDir, previewTextView, "")

	listView.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		path := secondaryText
		fileInfo, err := os.Stat(path)
		if err != nil {
			log.Println(err)
			return
		}

		mainView.CurrentLoadingFile = path
		if !fileInfo.IsDir() {
			previewTextView.SetTitle(path)
			previewTextView.SetText("[blue]Loading...")
			previewPages.SwitchToPage("text")
			go mainView.loadFileContent(config, path)
		} else {
			previewTextView.SetText("")
		}
	})

	return mainView
}

// loadDirectory loads directory content into a list view with tree-like formatting
func (m *MainView) loadDirectory(config *Config, listView *tview.List, path string, textView *tview.TextView, prefix string) {
	listView.Clear()

	m.walkDirectory(config, listView, path, textView, prefix)
}

func (m *MainView) walkDirectory(config *Config, listView *tview.List, path string, textView *tview.TextView, prefix string) {
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
			// on selected... do nothing.
		})
		if file.IsDir() {
			m.walkDirectory(config, listView, filePath, textView, prefix+"  ")
		}
	}
}

// loadFileContent loads and displays file content in the text view with syntax highlighting
func (m *MainView) loadFileContent(config *Config, path string) {
	fileExt := filepath.Ext(path)
	if fileExt == ".jpg" || fileExt == ".jpeg" || fileExt == ".png" || fileExt == ".gif" || fileExt == ".svg" {
		log.Printf("Loading image: %s", path)
		m.loadImage(path, fileExt)
	} else {
		m.loadTextFile(config, path)
	}
}

func (m *MainView) ShowPreviewImage(path string, image *image.Image) {
	m.Application.QueueUpdateDraw(func() {
		if m.CurrentLoadingFile == path {
			log.Printf("Displaying image: %s", path)
			m.PreviewPages.SwitchToPage("image")
			m.PreviewImageView.SetTitle(path)
			m.PreviewImageView.SetImage(*image)
		} else {
			log.Printf("Ignoring image: %s", path)
		}
	})
}

func (m *MainView) ShowPreviewText(path string, text string) {
	m.Application.QueueUpdateDraw(func() {
		if m.CurrentLoadingFile == path {
			log.Printf("Displaying text: %s", path)
			m.PreviewTextView.SetTitle(path)
			m.PreviewTextView.SetText(text)
			m.PreviewPages.SwitchToPage("text")
		} else {
			log.Printf("Ignoring text: %s", path)
		}
	})
}

func (m *MainView) loadImage(path string, fileExt string) {
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Failed to open image file: %v", err)
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Failed to close image file: %v", err)
		}
	}(file)

	loadImage := func(image image.Image, err error) {
		if err != nil {
			log.Printf("Failed to decode image: %v", err)
			return
		}

		log.Printf("Displaying image: %s", path)
		m.ShowPreviewImage(path, &image)
	}

	if fileExt == ".jpg" || fileExt == ".jpeg" {
		decoded, err := jpeg.Decode(file)
		loadImage(decoded, err)
	} else if fileExt == ".png" {
		decoded, err := png.Decode(file)
		loadImage(decoded, err)
	} else if fileExt == ".gif" {
		decoded, err := gif.Decode(file)
		loadImage(decoded, err)
	} else if fileExt == ".svg" {
		icon, err := oksvg.ReadIconStream(file)
		if err != nil {
			return
		}
		w, h := int(icon.ViewBox.W), int(icon.ViewBox.H)
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
		raster := rasterx.NewDasher(w, h, scanner)
		icon.Draw(raster, 1.0)
		loadImage(img, nil)
	} else {
		log.Printf("Unsupported image format: %s", fileExt)
	}
}

func (m *MainView) loadTextFile(config *Config, path string) {
	m.PreviewPages.SwitchToPage("text")

	log.Printf("Loading %s", path)
	content, err := os.ReadFile(path)
	if err != nil {
		m.ShowPreviewText(path, fmt.Sprintf("[red]Error loading file: %v", err))
		return
	}
	log.Printf("Finished reading %s(%d bytes)", path, len(content))

	if !utf8.Valid(content) {
		m.ShowPreviewText(path, "[red]Binary")
		return
	}

	highlightLimit := config.HighlightLimit
	if len(content) > highlightLimit {
		log.Printf("File is too large to highlight: %s(%d bytes > %d bytes)", path,
			len(content), highlightLimit)
		m.ShowPreviewText(path, string(content))
		return
	}

	// Detect syntax highlighting based on file extension
	fileExt := filepath.Ext(path)

	var highlighted bytes.Buffer
	if err := quick.Highlight(&highlighted, string(content), fileExt, "terminal", config.ChromaStyle); err == nil {
		m.ShowPreviewText(path, tview.TranslateANSI(highlighted.String()))
	} else {
		m.ShowPreviewText(path, string(content))
	}
}

func (m *MainView) findByKeyword() {
	// mainView.FindingKeyword にマッチする要素を探して選択する｡

	// マッチの方法は､全ての文字が並び順のとおりに含まれていれば良く､
	// 途中で文字が飛んでいても良い｡
	// 例えば "abc" は "afbkc" にもマッチするが "bac" にはマッチしない｡

	// まずは ItemBeforeFinding よりも後を探す｡その後に先頭から探す｡
	// 見つからなかったら ItemBeforeFinding の位置に戻す｡

	keyword := strings.ToLower(m.FindingKeyword)
	log.Printf("Finding: %s", keyword)
	if keyword == "" {
		return
	}

	// Function to check if a string contains all characters of the keyword in order
	matches := func(item, keyword string) bool {
		item = strings.ToLower(item)
		j := 0
		for i := 0; i < len(item) && j < len(keyword); i++ {
			if item[i] == keyword[j] {
				j++
			}
		}
		return j == len(keyword)
	}

	// Search from the current item to the end
	for i := m.ItemBeforeFinding + 1; i < m.ListView.GetItemCount(); i++ {
		mainText, _ := m.ListView.GetItemText(i)
		if matches(mainText, keyword) {
			log.Printf("Found: %s", mainText)
			m.ListView.SetCurrentItem(i)
			return
		}
	}

	// Search from the start to the current item
	for i := 0; i <= m.ItemBeforeFinding; i++ {
		mainText, _ := m.ListView.GetItemText(i)
		if matches(mainText, keyword) {
			log.Printf("Found: %s", mainText)
			m.ListView.SetCurrentItem(i)
			return
		}
	}

	// If no match is found, revert to the original item
	log.Printf("Not Found: %s", keyword)
	m.ListView.SetCurrentItem(m.ItemBeforeFinding)
}
