package files_view

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/chroma/quick"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"github.com/tokuhirom/mieta/mieta"
	"github.com/tokuhirom/mieta/mieta/config"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

type FilesView struct {
	Application      *tview.Application
	Config           *config.Config
	Pages            *tview.Pages
	Flex             *tview.Flex
	TreeView         *tview.TreeView
	PreviewPages     *tview.Pages
	PreviewImageView *tview.Image
	PreviewTextView  *tview.TextView
	// 検索モードに入る前に選択されていたアイテム
	NodeBeforeFinding *tview.TreeNode
	// 検索キーワード
	FindingKeyword     string
	CurrentLoadingFile string
	RootDir            string
	LeftPane           *tview.Flex
	FileNameSearchBox  *tview.InputField
	PreviewTextWrapper *tview.Flex
	InlineSearchBox    *tview.InputField
	MaxHighlightId     int
	CurrentHighlightId int
}

type FileNode struct {
	Path  string
	IsDir bool
}

func NewFilesView(rootDir string, config *config.Config, app *tview.Application, pages *tview.Pages) *FilesView {
	// Create tree view
	root := tview.NewTreeNode(filepath.Base(rootDir))
	root.SetReference(&FileNode{
		Path:  rootDir,
		IsDir: true,
	})

	treeView := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)
	treeView.SetBorder(true)
	treeView.SetBorderColor(tcell.ColorDarkSlateGray)

	previewTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true)
	previewTextView.SetBorder(true)
	previewTextView.SetBorderColor(tcell.ColorDarkSlateGray)
	previewTextView.SetBorderPadding(0, 0, 1, 1)

	previewTextWrapper := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(previewTextView, 0, 1, false)

	previewImageView := tview.NewImage()
	previewImageView.SetBorder(true)
	previewImageView.SetBorderColor(tcell.ColorDarkSlateGray)

	previewPages := tview.NewPages()
	previewPages.AddPage("text", previewTextWrapper, true, true)
	previewPages.AddPage("image", previewImageView, true, false)

	fileNameSearchBox := tview.NewInputField().
		SetLabel("🔎: ")

	inlineSearchBox := tview.NewInputField().
		SetLabel("🔎: ")

	inlineSearchBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			// leve from the input search mode
			previewTextWrapper.RemoveItem(inlineSearchBox)
			app.SetFocus(treeView)
			return nil
		case tcell.KeyEsc:
			// leave from the input search mode
			log.Printf("Escaped from the input search mode")
			previewTextWrapper.RemoveItem(inlineSearchBox)
			app.SetFocus(treeView)
			return nil
		default:
			return event
		}
	})

	leftPane := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(treeView, 0, 1, true)

	// Layout
	flex := tview.NewFlex().
		AddItem(leftPane, 30, 1, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(previewPages, 0, 2, false)

	filesView := &FilesView{
		Application:        app,
		Config:             config,
		Pages:              pages,
		Flex:               flex,
		LeftPane:           leftPane,
		FileNameSearchBox:  fileNameSearchBox,
		TreeView:           treeView,
		PreviewPages:       previewPages,
		PreviewTextWrapper: previewTextWrapper,
		InlineSearchBox:    inlineSearchBox,
		PreviewTextView:    previewTextView,
		PreviewImageView:   previewImageView,
		RootDir:            rootDir,
	}

	inlineSearchBox.SetChangedFunc(func(text string) {
		filesView.SearchByKeyword(text)
	})

	_, keycodeKeymap, runeKeymap := GetFilesKeymap(config)

	fileNameSearchBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			fileNameSearchBox.SetText("")
			// remove search box from the leftPane
			leftPane.RemoveItem(fileNameSearchBox)
			app.SetFocus(treeView)
			return nil
		case tcell.KeyEscape:
			treeView.SetCurrentNode(filesView.NodeBeforeFinding)
			fileNameSearchBox.SetText("")
			leftPane.RemoveItem(fileNameSearchBox)
			app.SetFocus(treeView)
			return nil
		default:
			return event
		}
	})
	fileNameSearchBox.SetChangedFunc(func(text string) {
		filesView.findByKeyword(text)
	})

	// キーバインド設定
	treeView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if handler, ok := keycodeKeymap[event.Key()]; ok {
			handler(filesView)
			return nil
		}

		if event.Key() == tcell.KeyRune {
			if handler, ok := runeKeymap[event.Rune()]; ok {
				handler(filesView)
				return nil
			}
		}

		return event
	})

	treeView.SetChangedFunc(func(node *tview.TreeNode) {
		log.Printf("ChangedFunc: %v", node.GetText())

		reference := node.GetReference()
		if reference == nil {
			return
		}

		fileNode := reference.(*FileNode)
		path := fileNode.Path

		if !fileNode.IsDir {
			// Load file content
			filesView.CurrentLoadingFile = path
			previewTextView.SetTitle(path)
			previewTextView.SetText("[blue]Loading...")
			previewPages.SwitchToPage("text")
			go filesView.loadFileContent(config, path)
		} else {
			previewTextView.SetTitle(path)
			previewTextView.SetText("[yellow]Directory...")
			previewPages.SwitchToPage("text")
		}
	})

	// Initial loading of the root directory
	if err := filesView.loadDirectoryContents(root, rootDir); err != nil {
		log.Printf("Error loading root directory: %v", err)
	}

	return filesView
}

// loadDirectoryContents loads the contents of a directory into a tree node
func (m *FilesView) loadDirectoryContents(node *tview.TreeNode, path string) error {
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// Sort files and directories
	for _, file := range files {
		filePath := filepath.Join(path, file.Name())
		isDir := file.IsDir()

		// Create a new node
		var childNode *tview.TreeNode
		if isDir {
			childNode = tview.NewTreeNode("📁" + file.Name() + "/")
		} else {
			childNode = tview.NewTreeNode("📄" + file.Name())
		}

		// Set reference data
		childNode.SetReference(&FileNode{
			Path:  filePath,
			IsDir: isDir,
		})

		// Add to parent
		node.AddChild(childNode)
	}

	return nil
}

// loadFileContent loads and displays file content in the text view with syntax highlighting
func (m *FilesView) loadFileContent(config *config.Config, path string) {
	fileExt := filepath.Ext(path)
	if fileExt == ".jpg" || fileExt == ".jpeg" || fileExt == ".png" || fileExt == ".gif" || fileExt == ".svg" {
		log.Printf("Loading image: %s", path)
		m.loadImage(path, fileExt)
	} else {
		m.loadTextFile(config, path)
	}
}

func (m *FilesView) ShowPreviewImage(path string, image *image.Image) {
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

func (m *FilesView) ShowPreviewText(path string, text string) {
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

func (m *FilesView) loadImage(path string, fileExt string) {
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

func (m *FilesView) loadTextFile(config *config.Config, path string) {
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

func (m *FilesView) findByKeyword(keyword string) {
	keyword = strings.ToLower(keyword)
	log.Printf("Finding: %s", keyword)
	if keyword == "" {
		return
	}

	// Function to check if a string contains all characters of the keyword in order
	matches := func(text, keyword string) bool {
		text = strings.ToLower(text)
		j := 0
		for i := 0; i < len(text) && j < len(keyword); i++ {
			if text[i] == keyword[j] {
				j++
			}
		}
		return j == len(keyword)
	}

	// Function to search nodes recursively
	var searchNode func(node *tview.TreeNode) *tview.TreeNode
	searchNode = func(node *tview.TreeNode) *tview.TreeNode {
		// Check current node
		text := node.GetText()
		if matches(text, keyword) {
			return node
		}

		// Check children
		for _, child := range node.GetChildren() {
			if found := searchNode(child); found != nil {
				return found
			}
		}

		return nil
	}

	// Start search from root
	if found := searchNode(m.TreeView.GetRoot()); found != nil {
		m.TreeView.SetCurrentNode(found)
	} else {
		// If no match found, revert to original node
		log.Printf("Not Found: %s", keyword)
		m.TreeView.SetCurrentNode(m.NodeBeforeFinding)
	}
}

func (m *FilesView) expand() {
	node := m.TreeView.GetCurrentNode()
	reference := node.GetReference()
	if reference == nil {
		return
	}

	fileNode := reference.(*FileNode)
	path := fileNode.Path

	if !fileNode.IsDir {
		return
	}

	node.ClearChildren()
	node.Expand()

	if err := m.loadDirectoryContents(node, path); err != nil {
		log.Printf("Error loading directory: %v", err)
	}
}

func (m *FilesView) Edit() {
	node := m.TreeView.GetCurrentNode()
	reference := node.GetReference()
	if reference == nil {
		return
	}

	fileNode := reference.(*FileNode)
	if fileNode.IsDir {
		return
	}

	// Open in external editor
	lineNumber := mieta.GetCurrentLineNumber(m.PreviewTextView)
	mieta.OpenInEditor(m.Application, m.Config, fileNode.Path, lineNumber)
}

func (m *FilesView) SearchByKeyword(keyword string) {
	// TextArea の中で現在表示位置の先頭から検索する｡それが終わったら先頭から現在表示位置の前までを検索する｡
	// マッチした行があればハイライトする｡

	keyword = strings.ToLower(keyword)
	log.Printf("Searching for keyword: %s", keyword)
	if keyword == "" {
		return
	}

	// Get the current text from the PreviewTextView
	text := m.PreviewTextView.GetText(true)
	lines := strings.Split(text, "\n")

	// Function to check if a line contains the keyword
	matches := func(line, keyword string) bool {
		return strings.Contains(strings.ToLower(line), keyword)
	}

	// Get the current scroll position
	startRow, _ := m.PreviewTextView.GetScrollOffset()

	re := regexp.MustCompile(`(?i)(` + regexp.QuoteMeta(keyword) + ")")

	firstHighlight := ""
	highlightId := 0

	// Search from the current position to the end
	for i := startRow; i < len(lines); i++ {
		if matches(lines[i], keyword) {
			log.Printf("Hit: '%v'", lines[i])
			highlightKey := fmt.Sprintf("highlight-%d", highlightId)
			if firstHighlight == "" {
				firstHighlight = highlightKey
			}
			highlightId += 1
			lines[i] = re.ReplaceAllString(lines[i], fmt.Sprintf(`[yellow::u]["%s"]$1[""][-:-:-:-]`, highlightKey))
		}
	}

	// Search from the beginning to the current position
	for i := 0; i < startRow; i++ {
		if matches(lines[i], keyword) {
			highlightKey := fmt.Sprintf("highlight-%d", highlightId)
			if firstHighlight == "" {
				firstHighlight = highlightKey
			}
			highlightId += 1
			lines[i] = re.ReplaceAllString(lines[i], fmt.Sprintf(`[yellow::u]["%s"]$1[""][-:-:-:-]`, highlightKey))
			log.Printf("Hit: %s", lines[i])
		}
	}

	m.PreviewTextView.SetRegions(true)
	m.MaxHighlightId = highlightId - 1
	m.CurrentHighlightId = 0

	if m.MaxHighlightId > 0 {
		highlightKey := fmt.Sprintf("highlight-%d", m.CurrentHighlightId)
		m.PreviewTextView.SetText(strings.Join(lines, "\n"))
		m.PreviewTextView.Highlight(highlightKey)
		m.PreviewTextView.ScrollToHighlight()
	}
}

func (m *FilesView) findNext() {
	if m.MaxHighlightId > m.CurrentHighlightId {
		m.CurrentHighlightId += 1
	} else {
		m.CurrentHighlightId = 0
	}

	log.Printf("findNext: %d", m.CurrentHighlightId)

	highlightKey := fmt.Sprintf("highlight-%d", m.CurrentHighlightId)
	m.PreviewTextView.Highlight(highlightKey)
	m.PreviewTextView.ScrollToHighlight()
}

func (m *FilesView) findPrev() {
	if m.CurrentHighlightId > 0 {
		m.CurrentHighlightId -= 1
	} else {
		m.CurrentHighlightId = m.MaxHighlightId
	}

	log.Printf("findPrev: %d", m.CurrentHighlightId)

	highlightKey := fmt.Sprintf("highlight-%d", m.CurrentHighlightId)
	m.PreviewTextView.Highlight(highlightKey)
	m.PreviewTextView.ScrollToHighlight()
}
