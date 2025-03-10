package mieta

import (
	"fmt"
	"github.com/rivo/tview"
	"github.com/tokuhirom/mieta/mieta/config"
	"log"
	"os"
	"os/exec"
	"strings"
)

// OpenInEditor opens the file in an external editor
func OpenInEditor(app *tview.Application, config *config.Config, filePath string, lineNumber int) {
	// エディタコマンドを取得
	editorCmd := config.Editor
	if editorCmd == "" {
		log.Fatalf("No editor configured")
		return
	}

	// 一時的にアプリケーションを中断
	app.Suspend(func() {
		// エディタコマンドを構築
		var cmd *exec.Cmd

		// エディタによって行番号の指定方法が異なるため、一般的なエディタに対応
		switch {
		case strings.Contains(editorCmd, "vim") || strings.Contains(editorCmd, "vi"):
			// Vim: +行番号 ファイル
			cmd = exec.Command(editorCmd, fmt.Sprintf("+%d", lineNumber), filePath)
		case strings.Contains(editorCmd, "emacs"):
			// Emacs: +行番号:列番号 ファイル
			cmd = exec.Command(editorCmd, fmt.Sprintf("+%d:1", lineNumber), filePath)
		case strings.Contains(editorCmd, "nano"):
			// Nano: +行番号,列番号 ファイル
			cmd = exec.Command(editorCmd, fmt.Sprintf("+%d,%d", lineNumber, 1), filePath)
		case strings.Contains(editorCmd, "code") || strings.Contains(editorCmd, "vscode"):
			// VS Code: -g ファイル:行番号:列番号
			cmd = exec.Command(editorCmd, "-g", fmt.Sprintf("%s:%d:%d", filePath, lineNumber, 1))
		default:
			// その他のエディタはファイルのみを指定
			cmd = exec.Command(editorCmd, filePath)
		}

		// 標準入出力をターミナルに接続
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// エディタを起動
		log.Printf("Opening file in editor: %s %v", editorCmd, cmd.Args)
		if err := cmd.Run(); err != nil {
			log.Printf("Error opening editor: %v", err)
		}
	})
}

func GetCurrentLineNumber(textView *tview.TextView) int {
	// スクロールオフセットを取得
	row, _ := textView.GetScrollOffset()

	// 表示領域の高さを取得
	_, _, _, height := textView.GetRect()

	// 中央の行を計算（表示領域の中央に表示されている行）
	centerLine := row + height/2

	// 行数が少ない場合は調整
	totalLines := textView.GetOriginalLineCount()
	if centerLine >= totalLines {
		centerLine = totalLines - 1
	}
	if centerLine < 0 {
		centerLine = 0
	}

	// 1-indexed に変換（エディタは通常1行目から始まる）
	return centerLine + 1
}
