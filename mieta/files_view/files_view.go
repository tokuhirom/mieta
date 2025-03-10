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
	"strings"
	"unicode/utf8"
)

const (
	TreeMode = iota
	FindingMode
)

type FilesView struct {
	Application      *tview.Application
	Mode             int
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

	previewImageView := tview.NewImage()
	previewImageView.SetBorder(true)
	previewImageView.SetBorderColor(tcell.ColorDarkSlateGray)

	previewPages := tview.NewPages()
	previewPages.AddPage("text", previewTextView, true, true)
	previewPages.AddPage("image", previewImageView, true, false)

	// Layout
	flex := tview.NewFlex().
		AddItem(treeView, 30, 1, true).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(previewPages, 0, 2, false)

	filesView := &FilesView{
		Application:      app,
		Config:           config,
		Pages:            pages,
		Flex:             flex,
		TreeView:         treeView,
		PreviewPages:     previewPages,
		PreviewTextView:  previewTextView,
		PreviewImageView: previewImageView,
		RootDir:          rootDir,
	}

	_, keycodeKeymap, runeKeymap := GetFilesKeymap(config)

	// キーバインド設定
	treeView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch filesView.Mode {
		case FindingMode:
			log.Printf("FindingMode: %v", event.Key())
			switch event.Key() {
			case tcell.KeyEscape:
				treeView.SetTitle("")
				filesView.Mode = TreeMode
				treeView.SetCurrentNode(filesView.NodeBeforeFinding)
				filesView.FindingKeyword = ""
			case tcell.KeyEnter:
				treeView.SetTitle("")
				filesView.Mode = TreeMode
				filesView.FindingKeyword = ""
			case tcell.KeyDEL:
				if len(filesView.FindingKeyword) > 0 {
					filesView.FindingKeyword = filesView.FindingKeyword[:len(filesView.FindingKeyword)-1]
					filesView.TreeView.SetTitle(filesView.FindingKeyword)
					filesView.findByKeyword()
				}
			case tcell.KeyRune:
				filesView.FindingKeyword += string(event.Rune())
				filesView.TreeView.SetTitle(filesView.FindingKeyword)
				filesView.findByKeyword()
			}
		case TreeMode:
			// キーコードに対応するハンドラを実行
			if handler, ok := keycodeKeymap[event.Key()]; ok {
				handler(filesView)
				return nil
			}

			// ルーンに対応するハンドラを実行
			if event.Key() == tcell.KeyRune {
				if handler, ok := runeKeymap[event.Rune()]; ok {
					handler(filesView)
					return nil
				}
			}
			//if event.Key() == tcell.KeyRune {
			//	switch event.Rune() {
			//	case 'f':
			//		// goto find file mode
			//		treeView.SetTitle("🔎")
			//		filesView.Mode = FindingMode
			//		filesView.NodeBeforeFinding = treeView.GetCurrentNode()
			//		filesView.FindingKeyword = ""
			//	case 'w':
			//		treeView.Move(-1)
			//	case 's':
			//		treeView.Move(1)
			//	case 'S':
			//		pages.ShowPage("search")
			//	case 'e':
			//		filesView.Edit()
			//		return nil
			//	case 'a':
			//		filesView.NavigateUp()
			//	case 'd':
			//		filesView.expand()
			//	case ' ':
			//		row, col := previewTextView.GetScrollOffset()
			//		_, _, _, height := previewTextView.GetRect()
			//		if previewTextView.GetOriginalLineCount() <= row+height {
			//			// Try to find the next node
			//			// TODO: This is not working correctly
			//			//currentNode := treeView.GetCurrentNode()
			//			//if currentNode.GetNextSibling() != nil {
			//			//	treeView.SetCurrentNode(currentNode.GetNextSibling())
			//			//} else {
			//			//	previewTextView.ScrollTo(row+9, col)
			//			//}
			//		} else {
			//			previewTextView.ScrollTo(row+9, col)
			//		}
			//	case 'H':
			//		_, _, width, _ := treeView.GetRect()
			//		flex.ResizeItem(treeView, width-2, 1)
			//	case 'L':
			//		_, _, width, _ := treeView.GetRect()
			//		flex.ResizeItem(treeView, width+2, 1)
			//	}
			//}
		}
		return nil
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

func (m *FilesView) findByKeyword() {
	keyword := strings.ToLower(m.FindingKeyword)
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
