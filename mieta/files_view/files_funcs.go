package files_view

import "log"

// FilesScrollDown は preview を下にスクロールします
func FilesScrollDown(view *FilesView) {
	row, col := view.PreviewTextView.GetScrollOffset()
	view.PreviewTextView.ScrollTo(row+9, col)
}

// FilesScrollUp は preview を上にスクロールします
func FilesScrollUp(view *FilesView) {
	row, col := view.PreviewTextView.GetScrollOffset()
	view.PreviewTextView.ScrollTo(row-9, col)
}

// FilesQuit はアプリケーションを終了します
func FilesQuit(view *FilesView) {
	view.Application.Stop()
}

// FilesShowHelp はヘルプページを表示します
func FilesShowHelp(view *FilesView) {
	view.Pages.ShowPage("help")
}

// FilesMoveUp はツリービューで上に移動します
func FilesMoveUp(view *FilesView) {
	view.TreeView.Move(-1)
}

// FilesMoveDown はツリービューで下に移動します
func FilesMoveDown(view *FilesView) {
	view.TreeView.Move(1)
}

// FilesShowSearch は検索ページを表示します
func FilesShowSearch(view *FilesView) {
	view.Pages.ShowPage("search")
}

// FilesEdit は現在のファイルをエディタで開きます
func FilesEdit(view *FilesView) {
	view.Edit()
}

// FilesNavigateUp はディレクトリを上に移動します
func FilesNavigateUp(view *FilesView) {
	node := view.TreeView.GetCurrentNode()
	if node == nil {
		log.Printf("No node selected")
		view.TreeView.SetCurrentNode(view.TreeView.GetRoot())
		return
	}
	reference := node.GetReference()
	if reference == nil {
		return
	}

	fileNode := reference.(*FileNode)
	if fileNode.IsDir && node.IsExpanded() {
		// The current node is a directory and expanded, ust collapse it.
		node.Collapse()
		return
	}

	// move to the parent node
	path := view.TreeView.GetPath(view.TreeView.GetCurrentNode())
	if len(path) > 2 {
		view.TreeView.SetCurrentNode(path[len(path)-2])
	}
}

// FilesExpand はディレクトリを展開します
func FilesExpand(view *FilesView) {
	view.expand()
}

// FilesScrollPageDown はプレビューを1ページ下にスクロールします
func FilesScrollPageDown(view *FilesView) {
	row, col := view.PreviewTextView.GetScrollOffset()
	_, _, _, height := view.PreviewTextView.GetRect()
	if view.PreviewTextView.GetOriginalLineCount() <= row+height {
		// ここでは次のノードを探す処理は実装しない（コメントアウトされていたため）
	} else {
		view.PreviewTextView.ScrollTo(row+9, col)
	}
}

// FilesDecreaseTreeWidth はツリービューの幅を減らします
func FilesDecreaseTreeWidth(view *FilesView) {
	_, _, width, _ := view.TreeView.GetRect()
	view.Flex.ResizeItem(view.TreeView, width-2, 1)
}

// FilesIncreaseTreeWidth はツリービューの幅を増やします
func FilesIncreaseTreeWidth(view *FilesView) {
	_, _, width, _ := view.TreeView.GetRect()
	view.Flex.ResizeItem(view.TreeView, width+2, 1)
}

// FilesEnterFindMode は検索モードに入ります
func FilesEnterFindMode(view *FilesView) {
	view.TreeView.SetTitle("🔎")
	view.Mode = FindingMode
	view.NodeBeforeFinding = view.TreeView.GetCurrentNode()
	view.FindingKeyword = ""
}

// FilesExitFindMode は検索モードを終了します
func FilesExitFindMode(view *FilesView) {
	view.TreeView.SetTitle("")
	view.Mode = TreeMode
	view.TreeView.SetCurrentNode(view.NodeBeforeFinding)
	view.FindingKeyword = ""
}
