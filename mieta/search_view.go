package mieta

import (
	"bufio"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/tokuhirom/mieta/mieta/search"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// SearchResult represents a search result or error message
type SearchResult struct {
	FilePath    string
	LineNumber  int
	MatchedLine string
	IsError     bool // エラーメッセージかどうかを示すフラグ
}

// SearchView represents the search functionality view
type SearchView struct {
	Application    *tview.Application
	Config         *Config
	MainView       *MainView
	Pages          *tview.Pages
	Flex           *tview.Flex
	InputField     *tview.InputField
	ResultList     *tview.List
	ContentView    *tview.TextView
	RootDir        string
	UseRegex       bool
	IgnoreCase     bool
	SearchResults  []SearchResult
	searchMutex    sync.Mutex
	currentCommand *os.Process
	searchDriver   search.SearchDriver
}

// NewSearchView creates a new search view
func NewSearchView(app *tview.Application, config *Config, mainView *MainView, pages *tview.Pages, rootDir string) *SearchView {
	// Create input field for search query
	inputField := tview.NewInputField().
		SetLabel("Search: ").
		SetFieldWidth(0)
	inputField.SetBorder(true)
	inputField.SetBorderColor(tcell.ColorDarkSlateGray)

	// Create list for search results
	resultList := tview.NewList().
		ShowSecondaryText(true)
	resultList.SetBorder(true)
	resultList.SetBorderColor(tcell.ColorDarkSlateGray)
	resultList.SetTitle("Results")

	// Create text view for file content
	contentView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true)
	contentView.SetBorder(true)
	contentView.SetBorderColor(tcell.ColorDarkSlateGray)
	contentView.SetTitle("Preview")
	contentView.SetBorderPadding(0, 0, 1, 1)

	// Create status bar for search options
	statusBar := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	statusBar.SetText("[yellow]Ctrl-R[white]: Toggle Regex | [yellow]Ctrl-I[white]: Toggle Case Sensitivity | [yellow]Esc[white]: Exit Search")

	// Create layout for input and list
	leftFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(inputField, 3, 0, true).
		AddItem(resultList, 0, 1, false).
		AddItem(statusBar, 1, 0, false)

	// Create main layout
	flex := tview.NewFlex().
		AddItem(leftFlex, 0, 1, true).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(contentView, 0, 2, false)

	driverName, extraOpts := config.GetSearchDriver()
	searchDriver := search.GetSearchDriver(driverName, extraOpts)

	searchView := &SearchView{
		Application:  app,
		Config:       config,
		MainView:     mainView,
		Pages:        pages,
		Flex:         flex,
		InputField:   inputField,
		ResultList:   resultList,
		ContentView:  contentView,
		RootDir:      rootDir,
		UseRegex:     false,
		IgnoreCase:   true,
		searchDriver: searchDriver,
	}

	// Set up input field behavior
	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			query := inputField.GetText()
			if query != "" {
				searchView.executeSearch(query)
				app.SetFocus(resultList)
			}
		}
	})

	// Set up input field key capture
	inputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			searchView.UseRegex = !searchView.UseRegex
			searchView.updateStatusBar()
			return nil
		case tcell.KeyCtrlI:
			searchView.IgnoreCase = !searchView.IgnoreCase
			searchView.updateStatusBar()
			return nil
		case tcell.KeyEscape:
			pages.SwitchToPage("background")
			app.SetFocus(mainView.TreeView)
			return nil
		}
		return event
	})

	// Set up result list behavior
	resultList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(searchView.SearchResults) {
			result := searchView.SearchResults[index]
			if result.IsError {
				// エラーメッセージの場合はそのまま表示
				searchView.ContentView.Clear()
				searchView.ContentView.SetTitle("Error")
				searchView.ContentView.SetText(fmt.Sprintf("[red]%s", result.MatchedLine))
			} else {
				// 通常の検索結果の場合はファイル内容を表示
				searchView.loadFileContent(result.FilePath, result.LineNumber)
			}
			app.SetFocus(contentView)
		}
	})

	// Set up result list key capture
	resultList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			pages.SwitchToPage("background")
			app.SetFocus(mainView.TreeView)
			return nil
		}

		switch event.Rune() {
		case 'w':
			index := resultList.GetCurrentItem()
			if index > 0 {
				resultList.SetCurrentItem(index - 1)
			}
			return nil
		case 's':
			index := resultList.GetCurrentItem()
			if index < resultList.GetItemCount()-1 {
				resultList.SetCurrentItem(index + 1)
			}
			return nil
		case 'q':
			pages.SwitchToPage("background")
			app.SetFocus(mainView.TreeView)
			return nil
		}
		return event
	})

	// Set up content view key capture
	contentView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'h':
			row, col := contentView.GetScrollOffset()
			if col > 0 {
				contentView.ScrollTo(row, col-1)
			}
			return nil
		case 'j':
			row, col := contentView.GetScrollOffset()
			contentView.ScrollTo(row+1, col)
			return nil
		case 'k':
			row, col := contentView.GetScrollOffset()
			if row > 0 {
				contentView.ScrollTo(row-1, col)
			}
			return nil
		case 'l':
			row, col := contentView.GetScrollOffset()
			contentView.ScrollTo(row, col+1)
			return nil
		case 'G':
			contentView.ScrollToEnd()
			return nil
		case 'q':
			pages.SwitchToPage("background")
			app.SetFocus(mainView.TreeView)
			return nil
		}

		if event.Key() == tcell.KeyEscape {
			pages.SwitchToPage("background")
			app.SetFocus(mainView.TreeView)
			return nil
		}
		return event
	})

	searchView.updateStatusBar()
	return searchView
}

// updateStatusBar updates the status bar text based on current settings
func (s *SearchView) updateStatusBar() {
	regexStatus := "OFF"
	if s.UseRegex {
		regexStatus = "ON"
	}

	caseStatus := "Ignore"
	if !s.IgnoreCase {
		caseStatus = "Match"
	}

	driverName := s.searchDriver.Name()

	statusText := fmt.Sprintf("[yellow]Driver[white]: %s | [yellow]Ctrl-R[white]: Regex (%s) | [yellow]Ctrl-I[white]: Case (%s) | [yellow]Esc[white]: Exit",
		driverName, regexStatus, caseStatus)

	// Find the status bar (third item in the left flex)
	leftFlex := s.Flex.GetItem(0).(*tview.Flex)
	statusBar := leftFlex.GetItem(2).(*tview.TextView)
	statusBar.SetText(statusText)
}

// executeSearch runs the search command and processes results
func (s *SearchView) executeSearch(query string) {
	// Clear previous results
	s.ResultList.Clear()
	s.ContentView.Clear()
	s.ContentView.SetTitle("Preview")

	s.searchMutex.Lock()
	s.SearchResults = nil

	// Cancel any existing search
	if s.currentCommand != nil {
		if err := s.currentCommand.Kill(); err != nil {
			log.Printf("Failed to kill previous search: %v", err)
		}
	}
	s.searchMutex.Unlock()

	// Check if the search driver is available
	if !s.searchDriver.IsAvailable() {
		s.addErrorResult(fmt.Sprintf("Search driver '%s' is not available. Please install it or choose another driver.", s.searchDriver.Name()))
		return
	}

	// Build the search command
	options := search.SearchOptions{
		Query:      query,
		RootDir:    s.RootDir,
		UseRegex:   s.UseRegex,
		IgnoreCase: s.IgnoreCase,
	}

	cmd, err := s.searchDriver.BuildCommand(options)
	if err != nil {
		s.addErrorResult(fmt.Sprintf("Failed to build search command: %v", err))
		return
	}

	log.Printf("Executing search: %s %v", cmd.Path, cmd.Args)

	// stdout と stderr をマージするためのパイプを作成
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		s.addErrorResult(fmt.Sprintf("Failed to create stdout pipe: %v", err))
		return
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		s.addErrorResult(fmt.Sprintf("Failed to create stderr pipe: %v", err))
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		s.addErrorResult(fmt.Sprintf("Failed to start search command: %v", err))
		return
	}

	s.searchMutex.Lock()
	s.currentCommand = cmd.Process
	s.searchMutex.Unlock()

	// stdout と stderr を非同期で処理
	// マルチリーダーを使ってログにも出力
	stdoutReader := io.TeeReader(stdoutPipe, &logWriter{prefix: "STDOUT"})
	stderrReader := io.TeeReader(stderrPipe, &logWriter{prefix: "STDERR"})

	// 結果チャネルを作成
	resultChan := make(chan SearchResult)

	// stdout を処理するゴルーチン
	go func() {
		defer close(resultChan)

		// stdout からの検索結果を処理
		results, err := s.searchDriver.ParseResults(stdoutReader)
		if err != nil {
			resultChan <- SearchResult{
				MatchedLine: fmt.Sprintf("Error parsing search results: %v", err),
				IsError:     true,
			}
			return
		}

		// 検索結果を結果チャネルに送信
		for result := range results {
			resultChan <- SearchResult{
				FilePath:    result.FilePath,
				LineNumber:  result.LineNumber,
				MatchedLine: result.MatchedLine,
				IsError:     false,
			}
		}

		// stderr からのエラーメッセージを処理
		scanner := bufio.NewScanner(stderrReader)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				resultChan <- SearchResult{
					MatchedLine: line,
					IsError:     true,
				}
			}
		}
	}()

	// 結果を処理するゴルーチン
	go func() {
		var results []SearchResult
		resultCount := 0

		for result := range resultChan {
			results = append(results, result)
			resultCount++

			// バッチ処理でUIを更新（パフォーマンス向上のため）
			if resultCount%10 == 0 || resultCount == 1 {
				localResults := make([]SearchResult, len(results))
				copy(localResults, results)

				s.Application.QueueUpdateDraw(func() {
					s.searchMutex.Lock()
					defer s.searchMutex.Unlock()

					s.SearchResults = localResults

					// これがまだアクティブな検索の場合のみ更新
					if s.ResultList.GetItemCount() < len(localResults) {
						for i := s.ResultList.GetItemCount(); i < len(localResults); i++ {
							r := localResults[i]
							if r.IsError {
								// エラーメッセージは赤色で表示
								s.ResultList.AddItem(
									"[red]Error",
									"[red]"+r.MatchedLine,
									0,
									nil,
								)
							} else {
								// 通常の検索結果
								relPath, _ := filepath.Rel(s.RootDir, r.FilePath)
								s.ResultList.AddItem(
									fmt.Sprintf("%s:%d", relPath, r.LineNumber),
									r.MatchedLine,
									0,
									nil,
								)
							}
						}
						s.ResultList.SetTitle(fmt.Sprintf("Results (%d)", len(localResults)))
					}
				})
			}
		}

		// 最終更新
		if len(results) > 0 {
			s.Application.QueueUpdateDraw(func() {
				s.searchMutex.Lock()
				defer s.searchMutex.Unlock()

				s.SearchResults = results
				s.ResultList.SetTitle(fmt.Sprintf("Results (%d)", len(results)))
			})
		} else {
			// 結果がない場合
			s.Application.QueueUpdateDraw(func() {
				s.addErrorResult("No results found")
			})
		}

		// コマンドの終了を待つ
		if err := cmd.Wait(); err != nil {
			// プロセスが強制終了された場合は無視
			if !strings.Contains(err.Error(), "killed") {
				log.Printf("Search command error: %v\n%v", err, cmd)
			}
		}
	}()
}

// addErrorResult adds an error message to the result list
func (s *SearchView) addErrorResult(message string) {
	s.searchMutex.Lock()
	defer s.searchMutex.Unlock()

	// エラーメッセージを検索結果に追加
	errorResult := SearchResult{
		MatchedLine: message,
		IsError:     true,
	}
	s.SearchResults = append(s.SearchResults, errorResult)

	// リストに表示
	s.ResultList.AddItem(
		"[red]Error",
		"[red]"+message,
		0,
		nil,
	)
	s.ResultList.SetTitle(fmt.Sprintf("Results (%d)", len(s.SearchResults)))
}

// loadFileContent loads and displays file content with the matched line highlighted
func (s *SearchView) loadFileContent(path string, lineNumber int) {
	s.ContentView.Clear()
	s.ContentView.SetTitle(path)

	file, err := os.Open(path)
	if err != nil {
		s.ContentView.SetText(fmt.Sprintf("[red]Error opening file: %v", err))
		return
	}
	defer file.Close()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		s.ContentView.SetText(fmt.Sprintf("[red]Error reading file: %v", err))
		return
	}

	lines := strings.Split(string(content), "\n")

	// Display file content with highlighted line
	var builder strings.Builder
	for i, line := range lines {
		lineNum := i + 1
		if lineNum == lineNumber {
			builder.WriteString(fmt.Sprintf("[yellow]%4d: %s[white]\n", lineNum, line))
		} else {
			builder.WriteString(fmt.Sprintf("%4d: %s\n", lineNum, line))
		}
	}

	s.ContentView.SetText(builder.String())

	// Scroll to the matched line
	if lineNumber > 0 && lineNumber <= len(lines) {
		// Calculate position to center the matched line
		_, _, _, height := s.ContentView.GetRect()
		targetLine := lineNumber - (height / 2)
		if targetLine < 0 {
			targetLine = 0
		}
		s.ContentView.ScrollTo(targetLine, 0)
	}
}

// logWriter is a simple io.Writer that logs output to the application log
type logWriter struct {
	prefix string
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	log.Printf("%s: %s", w.prefix, string(p))
	return len(p), nil
}
