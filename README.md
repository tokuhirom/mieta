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

### Keyboard Navigation
-  `w`/`s`: Move up and down within the tree
-  `j`/`k`: Scroll by page in the preview
-  `shift+h`/`shift+l`: Adjust the width of the tree and preview panes
-  `q`: Exit the application

### Layout
-  Displays the directory tree on the left and file preview on the right
-  The width ratio is dynamically adjustable

### Command Line Arguments
-  Allows specifying a directory path at startup
-  Uses the current directory if no path is specified

## Technical Features

### Framework
-  TUI application built using the tview framework

### Asynchronous Processing
-  Directory scanning is performed asynchronously using workers
-  Ensures responsive UI

### Error Handling
-  Displays appropriate error messages when files cannot be opened
-  Handles binary files, permission errors, and other IO errors

### Customizable Layout
-  Dynamically adjustable pane widths
-  Flexbox-based layout

## Future Expansion Possibilities
-  Search Functionality: Text search within files
-  Filtering: Filter files by specific extensions or name patterns
-  Theme Customization: Set user-preferred color themes
-  Custom Keybindings: Support for user-defined keybindings

