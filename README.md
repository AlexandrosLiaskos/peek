# peek

Side-by-side directory listing — Go + [Charm Lipgloss](https://github.com/charmbracelet/lipgloss).

Dirs on the left, files on the right. Green-on-black, human-readable sizes.

## Usage

```
peek              # list current directory
peek path/to/dir  # list specific directory
peek -a           # include hidden files
peek -f           # files only
```

Also wired as `ls`, `lsa`, `l` aliases.

## Install

```
go install github.com/AlexandrosLiaskos/peek@latest
```

Or build from source:

```
go build -o peek.exe .
```

## License

MIT
