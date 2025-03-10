package help_view

func HelpHidePage(view *HelpView) {
	view.Pages.HidePage("help")
}

func HelpScrollUp(view *HelpView) {
	row, col := view.TextView.GetScrollOffset()
	view.TextView.ScrollTo(row-1, col)
}

func HelpScrollDown(view *HelpView) {
	row, col := view.TextView.GetScrollOffset()
	view.TextView.ScrollTo(row+1, col)
}
