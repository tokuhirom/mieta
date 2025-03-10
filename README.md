# MIETA - CLI Tool for Directory Tree and File Preview

## Overview
MIETA is a CLI tool that visually displays directory structures in the terminal and allows you to preview the contents of selected files. As the name suggests, it allows you to "see" (MIETA) code and files easily. (The name is, of course, a nod to MIEL.)

![img.png](img.png)

## Install

```bash
go install github.com/tokuhirom/mieta@latest
```

## Main Features

### Directory Tree Display
-  Displays directory structure in a tree format
-  Automatically excludes `.git` directories
-  Tree display is asynchronous, ensuring the UI is not blocked even with large directories

### File Preview
-  Shows the contents of selected files with syntax highlighting
-  Supports various programming languages (Python, Go, Terraform, YAML, PHP, Perl, Kotlin, Java, JavaScript, TypeScript, HTML, CSS, Markdown, JSON, Bash, Ruby, Rust, C, C++, C#, etc.)
-  Displays appropriate error messages for binary files or permission errors
-  Supports image preview for common formats (JPG, PNG, GIF, SVG)

### Text Search
-  Full text search across files using powerful search tools (ag/The Silver Searcher or rg/ripgrep)
-  Configurable search options (case sensitivity, regex support)
-  Results displayed with context and highlighted matches
-  Navigate directly to matching lines in files
-  Extensible search driver architecture for adding new search tools

### External Editor Integration
-  Open files directly in your preferred external editor
-  Automatically opens to the current line in preview or matched line in search results
-  Configurable through config file or EDITOR environment variable
-  Supports common editors (vim, emacs, nano, VS Code) with proper line number handling

### Keyboard Navigation
-  `w`/`s`: Move up and down within the tree or search results
-  `j`/`k`: Scroll by page in the preview
-  `shift+h`/`shift+l`: Adjust the width of the tree and preview panes
-  `f`: Find files by name
-  `S`: Open search view for full text search
-  `e`: Open current file in external editor
-  `q`: Exit the application

### Layout
-  Displays the directory tree on the left and file preview on the right
-  The width ratio is dynamically adjustable

### Configuration
-  TOML configuration file for customizing behavior
-  Configurable syntax highlighting themes
-  Customizable search tools and options

## Command Line Arguments
-  Allows specifying a directory path at startup
-  Uses the current directory if no path is specified

## Technical Features

### Framework
-  TUI application built using the tview framework

### Asynchronous Processing
-  Directory scanning and search operations are performed asynchronously
-  Ensures responsive UI even during heavy operations

### Error Handling
-  Displays appropriate error messages when files cannot be opened
-  Handles binary files, permission errors, and other IO errors

### Customizable Layout
-  Dynamically adjustable pane widths
-  Flexbox-based layout

## Configuration

MIETA uses a TOML configuration file located at `~/.config/mieta/config.toml`. If the file doesn't exist, a default configuration will be created on first run.

Example configuration:

```toml
# MIETA configuration file

# Syntax highlighting style
# Available styles: monokai, github, vs, xcode, dracula, nord, solarized-dark, solarized-light
chroma_style = "monokai"

# Maximum file size (in bytes) for syntax highlighting
highlight_limit = 1000000

# External editor command
# If not specified, uses EDITOR environment variable
editor = "vim"

# Search settings
[search]
# Default search driver: "ag" or "rg"
driver = "ag"

# Search options
# extra_opts = ["--hidden", "--follow"]
```

## Keyboard Shortcuts

### Navigation
- `w`/`s`: Move up/down in tree or search results
- `j`/`k`: Scroll preview up/down
- `a`: Navigate up/collapse directory
- `d`: Expand directory
- `Space`: Scroll preview down one page
- `H`/`L`: Adjust panel widths

### Files
- `e`: Open current file in external editor
- `f`: Find files by name in the current tree

### Search
- `/`: Open search view for full text search
- `S`: Switch to search mode
- `Ctrl-R`: Toggle regex search
- `Ctrl-I`: Toggle case sensitivity
- `h`/`j`/`k`/`l`: Navigate in preview
- `G`: Go to end of preview

### Other
- `q`: Quit
- `?`: Show/hide help

## Debug Mode

For debugging purposes, you can set the `MIETA_DEBUG` environment variable to a file path where logs will be written:

```bash
MIETA_DEBUG=/tmp/mieta.log mieta /path/to/directory
```

## Future Expansion Possibilities
-  Filtering: Filter files by specific extensions or name patterns
-  Custom Keybindings: Support for user-defined keybindings
-  Git Integration: Show git status in the file tree

## Contributing

Contributions are welcome! Feel free to open issues or submit pull requests on GitHub.

## License

```
The MIT License (MIT)

Copyright © 2025 Tokuhiro Matsuno, https://64p.org/ <tokuhirom@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the “Software”), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```
