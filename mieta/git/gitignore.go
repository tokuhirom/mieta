package git

import (
	"errors"
	"os/exec"
)

// Git コマンドの存在を確認
func isGitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// 指定したファイルが Git に無視されているかを一発で判定
func IsEffectivelyIgnored(filePath string) (bool, error) {
	// Git コマンドが存在しない場合はエラーを返す
	if !isGitAvailable() {
		return false, errors.New("git command not found")
	}

	// まず、ファイルが Git に追跡されているかを確認
	cmd := exec.Command("git", "ls-files", "--error-unmatch", filePath)
	err := cmd.Run()
	if err == nil {
		// Git に追跡されている場合、.gitignore の影響は受けない
		return false, nil
	}

	// 追跡されていない場合、.gitignore によって無視されているかを確認
	cmd = exec.Command("git", "check-ignore", "-q", filePath)
	err = cmd.Run()
	if err == nil {
		// 終了コード 0 は .gitignore によって無視されている
		return true, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		// 終了コード 1 は無視されていない
		return false, nil
	}
	return false, err
}
