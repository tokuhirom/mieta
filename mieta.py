from textual.app import App, ComposeResult
from textual.containers import Container
from textual.widgets import Footer, Header, Button, DirectoryTree, TextArea, ListView, ListItem, Label

import os

def scan_directory(path):
    result = []
    
    def _scan(dir_path, prefix=""):
        try:
            entries = sorted(os.listdir(dir_path))
        except PermissionError:
            return  # Skip directories that cannot be accessed

        for i, entry in enumerate(entries):
            if entry == '.git':
                continue

            full_path = os.path.join(dir_path, entry)
            connector = "└─ " if i == len(entries) - 1 else "├─ "
            display_name = f"{prefix}{connector}{entry}"
            result.append((display_name, full_path))

            if os.path.isdir(full_path):
                extension = "    " if i == len(entries) - 1 else "│  "
                _scan(full_path, prefix + extension)

    _scan(path)
    return result


class StopwatchApp(App):
    """A Textual app to manage stopwatches."""

    CSS_PATH = "layout.tcss"
    BINDINGS = [
            ("q", "quit", "Quit"),
            ]

    def compose(self) -> ComposeResult:
        """Create child widgets for the app."""
        yield Header()
        list_items = self.get_list_items()
        lv =ListView(
            *list_items
        )
        def p(s, q):
            print(s, q)
        lv.on_list_view_selected = p
        lv.focus()
        ta = TextArea()
        ta.read_only = True
        self.ta = ta
        self.text_container = Container()
        yield Container(
            lv,
            self.text_container,
            id='top'
        )
        yield Footer()

    def action_toggle_dark(self) -> None:
        """An action to toggle dark mode."""
        self.theme = (
            "textual-dark" if self.theme == "textual-light" else "textual-light"
        )

    # TODO highlight の移動が ws で出来る必要あり
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
                    with open(filename, 'r', encoding='utf-8') as f:
                        editor = TextArea.code_editor(f.read(), language=language)
                        editor.read_only = True
                        editor.show_line_numbers = False
                        self.text_container.mount(editor)
                except UnicodeDecodeError:
                    # テキストファイルとして開けない場合
                    self.text_container.mount(Label(f"[red]Cannot open binary file: {filename}[/red]"))
                except PermissionError:
                    # 権限がない場合
                    self.text_container.mount(Label(f"[red]Permission denied: {filename}[/red]"))
                except IOError as e:
                    # その他のIO関連のエラー
                    self.text_container.mount(Label(f"[red]Error opening file: {filename}[/red]\n{str(e)}"))
                
            except Exception as e:
                # その他の予期しないエラー
                self.text_container.mount(Label(f"[red]Unexpected error: {str(e)}[/red]"))
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
        scanned_files = scan_directory('.')

        items = []
        for display_name, actual_name in scanned_files:
            item = ListItem(Label(display_name))
            item.file = actual_name
            items.append(item)

        return items


if __name__ == "__main__":
    app = StopwatchApp()
    app.run()
