import curses
from rich.console import Console
from rich.tree import Tree
from rich.syntax import Syntax
import os

def build_tree(base_path, parent_tree):
    for entry in os.listdir(base_path):
        full_path = os.path.join(base_path, entry)
        if entry == '.git':
            parent_tree.add(f"[dim].git[/dim] ...")
        elif os.path.isdir(full_path):
            subtree = parent_tree.add(f"[bold]{entry}[/bold]")
            build_tree(full_path, subtree)
        else:
            parent_tree.add(entry)

def main(stdscr):
    console = Console()
    base_path = '.'  # Starting directory
    tree = Tree(base_path)
    build_tree(base_path, tree)

    # Initialize curses
    curses.curs_set(0)
    stdscr.nodelay(1)
    stdscr.timeout(100)

    # Variables to track state
    selected_index = 0
    file_list = [node.label for node in tree.children]
    preview_offset = 0

    while True:
        stdscr.clear()

        # Display the tree
        for idx, line in enumerate(file_list):
            if idx == selected_index:
                stdscr.addstr(idx, 0, line, curses.A_REVERSE)
            else:
                stdscr.addstr(idx, 0, line)

        # Display the preview of the selected file
        if os.path.isfile(file_list[selected_index]):
            with open(file_list[selected_index], 'r') as f:
                code = f.read()
            syntax = Syntax(code, "auto", theme="monokai", line_numbers=True)
            console.print(syntax, overflow="ignore", new_line_start=False)

        # Input handling
        key = stdscr.getch()
        if key == ord('q'):
            break
        elif key == ord('w'):
            selected_index = max(0, selected_index - 1)
        elif key == ord('s'):
            selected_index = min(len(file_list) - 1, selected_index + 1)
        elif key == ord('j'):
            preview_offset += 1
        elif key == ord('k'):
            preview_offset = max(0, preview_offset - 1)

        stdscr.refresh()

if __name__ == "__main__":
    curses.wrapper(main)
