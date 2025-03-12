package git

import (
	"bufio"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitTracker は Git リポジトリのファイル追跡状態を管理するクラス
type GitTracker struct {
	ignoredFiles map[string]bool
}

// NewGitTracker は GitTracker の新しいインスタンスを作成
func NewGitTracker() *GitTracker {
	tracker := &GitTracker{
		ignoredFiles: make(map[string]bool),
	}

	return tracker
}

// Initialize は Git リポジトリからファイル情報を初期化
func (g *GitTracker) Initialize() error {
	// Git コマンドが利用可能か確認
	_, err := exec.LookPath("git")
	if err != nil {
		log.Printf("git not found in $PATH")
		return nil
	}

	cmd := exec.Command("git", "ls-files", "--others", "--ignored", "--exclude-standard")
	output, err := cmd.Output()
	if err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			relPath := scanner.Text()
			abs, err := filepath.Abs(relPath)
			if err == nil {
				log.Printf("Found ignored file: %s", abs)
				g.ignoredFiles[abs] = true
			} else {
				log.Printf("Error for Abs: %s, %v", relPath, err)
			}
		}
	}

	return nil
}

// IsIgnored はファイルが Git で無視されているかを判定
func (g *GitTracker) IsIgnored(filePath string) bool {
	// 追跡されているファイルは無視されない
	if g.ignoredFiles[filePath] {
		log.Printf("Ignored file: %s", filePath)
		return true
	}

	// .git ディレクトリは特別扱い
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		log.Printf("Abs error: %s", filePath)
		return false
	}
	if strings.Contains(absPath, "/.git/") || strings.HasSuffix(absPath, "/.git") {
		log.Printf(".git: %s", filePath)
		return true
	}

	log.Printf("not ignored: %s", filePath)
	return false
}
