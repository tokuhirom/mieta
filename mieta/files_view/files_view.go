package files_view

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/chroma/quick"
	"github.com/fsnotify/fsnotify"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"github.com/tokuhirom/mieta/mieta"
	"github.com/tokuhirom/mieta/mieta/config"
	"github.com/tokuhirom/mieta/mieta/git"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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

	// 読み込み中のディレクトリを追跡するためのマップとそのロック
	loadingDirs      map[string]bool
	loadingDirsMutex sync.Mutex
	gitTracker       *git.GitTracker

	watcher      *fsnotify.Watcher
	watchedDirs  map[string]bool
	watcherMutex sync.Mutex
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

	gitTarcker := git.NewGitTracker()
	err := gitTarcker.Initialize()
	if err != nil {
		log.Printf("Error initializing git tracker: %v", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Error creating fsnotify watcher: %v", err)
	}

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

		gitTracker:  gitTarcker,
		loadingDirs: make(map[string]bool),

		watcher:     watcher,
		watchedDirs: make(map[string]bool),
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

	{
		filesView.watcherMutex.Lock()
		defer filesView.watcherMutex.Unlock()
		filesView.startWatching(rootDir)
	}

	// fsnotifyイベント処理用のgoroutineを起動
	go filesView.watchEvents()

	return filesView
}

// loadDirectoryContents loads the contents of a directory into a tree node
func (m *FilesView) loadDirectoryContents(node *tview.TreeNode, path string) error {
	// ロックを取得してディレクトリの読み込み状態を確認
	m.loadingDirsMutex.Lock()
	if m.loadingDirs[path] {
		// 既に読み込み中なら何もしない
		m.loadingDirsMutex.Unlock()
		return nil
	}

	// 読み込み中としてマーク
	m.loadingDirs[path] = true
	m.loadingDirsMutex.Unlock()

	// 読み込み中の表示
	loadingNode := tview.NewTreeNode("[yellow]Loading...")

	go func() {
		m.Application.QueueUpdateDraw(func() {
			// 子ノードがすでにあるか確認
			if len(node.GetChildren()) == 0 {
				node.AddChild(loadingNode)
			}
		})

		// 処理が終了したらマップから削除するための遅延処理
		defer func() {
			m.loadingDirsMutex.Lock()
			delete(m.loadingDirs, path)
			m.loadingDirsMutex.Unlock()
		}()

		files, err := os.ReadDir(path)
		if err != nil {
			m.Application.QueueUpdateDraw(func() {
				// ノードがまだ有効かチェック
				for _, child := range node.GetChildren() {
					if child == loadingNode {
						node.RemoveChild(loadingNode)
						errorNode := tview.NewTreeNode("[red]Error: " + err.Error())
						node.AddChild(errorNode)
						break
					}
				}
			})
			return
		}

		// ファイルをバッチ処理
		const batchSize = 50
		var batch []*tview.TreeNode

		for _, file := range files {
			filePath := filepath.Join(path, file.Name())
			isDir := file.IsDir()

			var childNode *tview.TreeNode
			if isDir {
				childNode = tview.NewTreeNode("📁" + file.Name() + "/")
			} else {
				childNode = tview.NewTreeNode("📄" + file.Name())
			}

			childNode.SetReference(&FileNode{
				Path:  filePath,
				IsDir: isDir,
			})

			ignored := m.gitTracker.IsIgnored(filePath)
			if ignored {
				childNode.SetColor(tcell.ColorDarkGray)
			}

			batch = append(batch, childNode)

			if len(batch) >= batchSize || file == files[len(files)-1] {
				nodesToAdd := make([]*tview.TreeNode, len(batch))
				copy(nodesToAdd, batch)

				m.Application.QueueUpdateDraw(func() {
					// ノードがまだ有効かチェック
					stillValid := false
					for _, child := range node.GetChildren() {
						if child == loadingNode {
							stillValid = true
							break
						}
					}

					if stillValid {
						for _, n := range nodesToAdd {
							node.AddChild(n)
						}
					}
				})

				batch = batch[:0]
			}
		}

		// 読み込み完了後にローディングノードを削除
		m.Application.QueueUpdateDraw(func() {
			for _, child := range node.GetChildren() {
				if child == loadingNode {
					node.RemoveChild(loadingNode)
					break
				}
			}
		})
	}()

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

	// ノードが既に展開されているか確認
	if node.IsExpanded() {
		// 既に展開されている場合は、子ノードがあるか確認
		if len(node.GetChildren()) == 0 {
			// 子ノードがなければ読み込み
			if err := m.loadDirectoryContents(node, path); err != nil {
				log.Printf("Error loading directory: %v", err)
			}
		}
	} else {
		// 展開されていない場合は展開
		node.Expand()

		// 子ノードがなければ読み込み
		if len(node.GetChildren()) == 0 {
			if err := m.loadDirectoryContents(node, path); err != nil {
				log.Printf("Error loading directory: %v", err)
			}
		}
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

func (m *FilesView) startWatching(path string) {
	// 既に監視中なら何もしない
	if m.watchedDirs[path] {
		return
	}

	// ディレクトリを監視対象に追加
	err := m.watcher.Add(path)
	if err != nil {
		log.Printf("Error watching directory %s: %v", path, err)
		return
	}

	m.watchedDirs[path] = true
	log.Printf("Started watching directory: %s", path)

	// サブディレクトリも再帰的に監視
	files, err := os.ReadDir(path)
	if err != nil {
		log.Printf("Error reading directory %s: %v", path, err)
		return
	}

	log.Printf("Found %d files", len(files))
	for _, file := range files {
		if file.IsDir() {
			subPath := filepath.Join(path, file.Name())
			m.startWatching(subPath)
		}
	}

	log.Printf("Started watching directory: %s", path)
}

func (m *FilesView) watchEvents() {
	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}

			// イベント処理
			m.handleFsEvent(event)

		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func (m *FilesView) handleFsEvent(event fsnotify.Event) {
	if event.Op&fsnotify.Chmod == fsnotify.Chmod {
		// git generates too much Chmod events. ignore it.
		return
	}

	log.Printf("FS event: %v", event)

	// ディレクトリの作成イベント
	if event.Op&fsnotify.Create == fsnotify.Create {
		fileInfo, err := os.Stat(event.Name)
		if err != nil {
			log.Printf("Error getting file info for %s: %v", event.Name, err)
			return
		}

		// 新しいディレクトリが作成された場合は監視対象に追加
		if fileInfo.IsDir() {
			m.startWatching(event.Name)
		}

		// UIツリーに新しいノードを追加
		m.Application.QueueUpdateDraw(func() {
			m.addNodeForPath(event.Name, fileInfo.IsDir())
		})
	}

	// ファイル/ディレクトリの削除イベント
	if event.Op&fsnotify.Remove == fsnotify.Remove {
		m.Application.QueueUpdateDraw(func() {
			m.removeNodeForPath(event.Name)
		})

		// 監視リストから削除
		m.watcherMutex.Lock()
		delete(m.watchedDirs, event.Name)
		m.watcherMutex.Unlock()
	}

	// ファイルの変更イベント
	if event.Op&fsnotify.Write == fsnotify.Write {
		// 現在表示中のファイルが変更された場合は再読み込み
		if m.CurrentLoadingFile == event.Name {
			go m.loadFileContent(m.Config, event.Name)
		}
	}
}

// パスに対応するノードをツリーに追加する関数
func (m *FilesView) addNodeForPath(path string, isDir bool) {
	// パスの親ディレクトリを特定
	parentPath := filepath.Dir(path)
	fileName := filepath.Base(path)

	// 親ノードを探す
	parentNode := m.findNodeByPath(m.TreeView.GetRoot(), parentPath)
	if parentNode == nil {
		log.Printf("Cannot find parent node for %s", path)
		return
	}

	// 既に同名のノードがあるか確認
	for _, child := range parentNode.GetChildren() {
		ref := child.GetReference()
		if ref == nil {
			continue
		}

		fileNode := ref.(*FileNode)
		if fileNode.Path == path {
			// 既に存在する場合は何もしない
			return
		}
	}

	// 新しいノードを作成
	var newNode *tview.TreeNode
	if isDir {
		newNode = tview.NewTreeNode("📁" + fileName + "/")
	} else {
		newNode = tview.NewTreeNode("📄" + fileName)
	}

	newNode.SetReference(&FileNode{
		Path:  path,
		IsDir: isDir,
	})

	// Gitで無視されているかチェック
	ignored := m.gitTracker.IsIgnored(path)
	if ignored {
		newNode.SetColor(tcell.ColorDarkGray)
	}

	// 親ノードに追加
	parentNode.AddChild(newNode)
}

func (m *FilesView) removeNodeForPath(path string) {
	// 親パスを特定
	parentPath := filepath.Dir(path)

	// 親ノードを探す
	parentNode := m.findNodeByPath(m.TreeView.GetRoot(), parentPath)
	if parentNode == nil {
		return
	}

	// 子ノードを探して削除
	for _, child := range parentNode.GetChildren() {
		ref := child.GetReference()
		if ref == nil {
			continue
		}

		fileNode := ref.(*FileNode)
		if fileNode.Path == path {
			parentNode.RemoveChild(child)
			return
		}
	}
}

// パスからノードを探す関数
func (m *FilesView) findNodeByPath(node *tview.TreeNode, path string) *tview.TreeNode {
	ref := node.GetReference()
	if ref == nil {
		return nil
	}

	fileNode := ref.(*FileNode)
	if fileNode.Path == path {
		return node
	}

	// 子ノードを再帰的に探索
	for _, child := range node.GetChildren() {
		if found := m.findNodeByPath(child, path); found != nil {
			return found
		}
	}

	return nil
}

// アプリケーション終了時にウォッチャーをクローズするメソッドを追加
func (m *FilesView) Close() {
	if m.watcher != nil {
		m.watcher.Close()
	}
}
