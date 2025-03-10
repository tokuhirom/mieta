package search_view

import "github.com/rivo/tview"

// search_view.go に追加

// SearchFocusInput は入力フィールドにフォーカスを移します
func SearchFocusInput(view *SearchView) {
	view.Application.SetFocus(view.InputField)
}

// SearchPreviousItem は前の検索結果アイテムを表示します
func SearchPreviousItem(view *SearchView) {
	view.ShowPreviousItem()
}

// SearchNextItem は次の検索結果アイテムを表示します
func SearchNextItem(view *SearchView) {
	view.ShowNextItem()
}

// SearchExitView は検索ビューを終了します
func SearchExitView(view *SearchView) {
	view.ShowFilesView()
}

// SearchScrollLeft はプレビューを左にスクロールします
func SearchScrollLeft(view *SearchView) {
	row, col := view.ContentView.GetScrollOffset()
	if col > 0 {
		view.ContentView.ScrollTo(row, col-1)
	}
}

// SearchScrollDown はプレビューを下にスクロールします
func SearchScrollDown(view *SearchView) {
	row, col := view.ContentView.GetScrollOffset()
	view.ContentView.ScrollTo(row+1, col)
}

// SearchScrollUp はプレビューを上にスクロールします
func SearchScrollUp(view *SearchView) {
	row, col := view.ContentView.GetScrollOffset()
	if row > 0 {
		view.ContentView.ScrollTo(row-1, col)
	}
}

// SearchScrollRight はプレビューを右にスクロールします
func SearchScrollRight(view *SearchView) {
	row, col := view.ContentView.GetScrollOffset()
	view.ContentView.ScrollTo(row, col+1)
}

// SearchEdit は現在選択されているファイルをエディタで開きます
func SearchEdit(view *SearchView) {
	view.Edit()
}

// SearchScrollToEnd はプレビューの最後までスクロールします
func SearchScrollToEnd(view *SearchView) {
	view.ContentView.ScrollToEnd()
}

// SearchDecreaseLeftWidth は左パネルの幅を減らします
func SearchDecreaseLeftWidth(view *SearchView) {
	leftFlex := view.Flex.GetItem(0).(*tview.Flex)
	_, _, width, _ := leftFlex.GetRect()
	view.Flex.ResizeItem(leftFlex, width-2, 1)
}

// SearchIncreaseLeftWidth は左パネルの幅を増やします
func SearchIncreaseLeftWidth(view *SearchView) {
	leftFlex := view.Flex.GetItem(0).(*tview.Flex)
	_, _, width, _ := leftFlex.GetRect()
	view.Flex.ResizeItem(leftFlex, width+2, 1)
}

// SearchToggleRegex は正規表現検索の有効/無効を切り替えます
func SearchToggleRegex(view *SearchView) {
	view.UseRegex = !view.UseRegex
	view.updateStatusBar()
}

// SearchToggleCase は大文字小文字の区別の有効/無効を切り替えます
func SearchToggleCase(view *SearchView) {
	view.IgnoreCase = !view.IgnoreCase
	view.updateStatusBar()
}
