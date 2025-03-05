from textual.app import App, ComposeResult
from textual.containers import Container
from textual.widgets import (
    Footer,
    Header,
    Button,
    DirectoryTree,
    TextArea,
    ListView,
    ListItem,
    Label,
)
from textual.binding import Binding

import os


def scan_directory(path, limit=None):
    result = []
    limited = False

    def _scan(dir_path, prefix=""):
        try:
            entries = sorted(os.listdir(dir_path))
        except PermissionError:
            return  # Skip directories that cannot be accessed

        for i, entry in enumerate(entries):
            if entry == ".git":
                continue

            full_path = os.path.join(dir_path, entry)
            connector = "└─ " if i == len(entries) - 1 else "├─ "
            display_name = f"{prefix}{connector}{entry}"
            result.append((display_name, full_path))

            if limit and len(result) >= limit:
                limited = True
                break

            if os.path.isdir(full_path):
                extension = "    " if i == len(entries) - 1 else "│  "
                _scan(full_path, prefix + extension)

    _scan(path)
    return result, limited


class StopwatchApp(App):
    """A Textual app to manage stopwatches."""

    def __init__(self, directory='.', limit=1000, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.directory = directory
        self.limit = limit

    CSS_PATH = "layout.tcss"
    BINDINGS = [
        ("q", "quit", "Quit"),
        ("w", "move_up", "Move Up"),
        ("s", "move_down", "Move Down"),
        ("j", "scroll_down", "Scroll Down"),
        ("k", "scroll_up", "Scroll Up"),
        ("z", "narrow_tree", "Narrow Tree"),
        ("x", "widen_tree", "Widen Tree"),
    ]

    def compose(self) -> ComposeResult:
        """Create child widgets for the app."""
        yield Header()
        list_items, limited = self.get_list_items()
        self.lv = ListView(*list_items)
        self.lv.focus()

        self.tree_width = 30  # デフォルトのツリー幅

        ta = TextArea()
        ta.read_only = True
        self.ta = ta
        self.text_container = Container()
        if limited:
            yield Label("[red]too much items in {self.directory}... some files are snipped[/red]")
        yield Container(self.lv, self.text_container, id="top")
        yield Footer()

    def action_toggle_dark(self) -> None:
        """An action to toggle dark mode."""
        self.theme = (
            "textual-dark" if self.theme == "textual-light" else "textual-light"
        )

    def action_move_up(self) -> None:
        """wキーでリストを上に移動"""
        if self.lv.index > 0:
            self.lv.index -= 1

    def action_move_down(self) -> None:
        """sキーでリストを下に移動"""
        if self.lv.index < len(self.lv.children) - 1:
            self.lv.index += 1

    def action_scroll_down(self) -> None:
        """jキーでプレビューを下にスクロール"""
        # プレビューエリアにフォーカスを当てる
        for child in self.text_container.children:
            if isinstance(child, TextArea):
                child.action_cursor_page_down()

    def action_scroll_up(self) -> None:
        """kキーでプレビューを上にスクロール"""
        # プレビューエリアにフォーカスを当てる
        for child in self.text_container.children:
            if isinstance(child, TextArea):
                child.action_cursor_page_up()

    def on_list_view_highlighted(self, event: ListView.Highlighted) -> None:
        """リストビューでアイテムがハイライトされたときに呼ばれるメソッド"""
        # コンテナの中身をクリア
        self.text_container.remove_children()

        # ハイライトされたアイテムのファイルパスを取得
        filename = event.item.file

        # ファイルかディレクトリかを確認
        if os.path.isfile(filename):
            try:
                # ファイル拡張子を取得して言語を推測
                _, ext = os.path.splitext(filename)
                language = self.get_language_from_extension(ext)

                # ファイルを開いてみて、開けるかどうか確認
                try:
                    with open(filename, "r", encoding="utf-8") as f:
                        self.text_container.mount(Label(f"{filename}"))
                        editor = TextArea.code_editor(f.read(), language=language)
                        editor.read_only = True
                        editor.show_line_numbers = False
                        self.text_container.mount(editor)
                except UnicodeDecodeError:
                    # テキストファイルとして開けない場合
                    self.text_container.mount(
                        Label(f"[red]Cannot open binary file: {filename}[/red]")
                    )
                except PermissionError:
                    # 権限がない場合
                    self.text_container.mount(
                        Label(f"[red]Permission denied: {filename}[/red]")
                    )
                except IOError as e:
                    # その他のIO関連のエラー
                    self.text_container.mount(
                        Label(f"[red]Error opening file: {filename}[/red]\n{str(e)}")
                    )

            except Exception as e:
                # その他の予期しないエラー
                self.text_container.mount(
                    Label(f"[red]Unexpected error: {str(e)}[/red]")
                )
        else:
            # ディレクトリの場合は情報メッセージを表示
            self.text_container.mount(Label(f"Directory: {filename}"))

    def get_language_from_extension(self, ext: str) -> str:
        """ファイル拡張子から言語を推測する"""
        language_map = {
            ".py": "python",
            ".tf": "terraform",
            ".yml": "yaml",
            ".yaml": "yaml",
            ".go": "go",
            ".php": "php",
            ".pl": "perl",
            ".kt": "kotlin",
            ".java": "java",
            ".js": "javascript",
            ".ts": "typescript",
            ".html": "html",
            ".css": "css",
            ".md": "markdown",
            ".json": "json",
            ".sh": "bash",
            ".rb": "ruby",
            ".rs": "rust",
            ".c": "c",
            ".cpp": "cpp",
            ".h": "c",
            ".hpp": "cpp",
            ".cs": "csharp",
        }
        return language_map.get(ext.lower(), None)

    def get_list_items(self):
        scanned_files, limited = scan_directory(self.directory, self.limit)

        items = []
        for display_name, actual_name in scanned_files:
            item = ListItem(Label(display_name))
            item.file = actual_name
            items.append(item)

        return items, limited

    def on_mount(self) -> None:
        """マウント時に初期レイアウトを設定"""
        self.update_layout()

    def update_layout(self) -> None:
        """レイアウトを更新する"""
        # CSSを動的に更新して幅を変更
        self.lv.styles.width = f"{self.tree_width}fr"
        self.text_container.styles.width = f"{100 - self.tree_width}fr"

    def action_narrow_tree(self) -> None:
        """Shift+Hでツリー表示部を狭くする"""
        if self.tree_width > 10:  # 最小幅を設定
            self.tree_width -= 5
            self.update_layout()

    def action_widen_tree(self) -> None:
        """Shift+Lでツリー表示部を広くする"""
        if self.tree_width < 70:  # 最大幅を設定
            self.tree_width += 5
            self.update_layout()

if __name__ == "__main__":
    import sys
    directory = sys.argv[1] if len(sys.argv) > 1 else '.'
    app = StopwatchApp(directory)
    app.run()
