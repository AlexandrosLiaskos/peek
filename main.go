package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

const maxNameLen = 45

var (
	// Panel headers
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#005c2e")).Bold(true)
	borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#003d1a"))

	// Dir names
	dirNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff66")).Bold(true)
	dotDirStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#006633")).Bold(true)

	// File names
	fileNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00cc55"))
	dotFileStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#006633"))

	// Subtitle: size · ext
	subStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#004d26")).Italic(true)

	// Symlinks
	symNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffaa"))

	// Count / footer
	countStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#003d1a"))

	// Error
	errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff3334"))
)

type entry struct {
	name  string
	isDir bool
	isSym bool
	size  int64
	dot   bool
	ext   string
}

func main() {
	showAll := false
	filesOnly := false
	target := "."

	for _, arg := range os.Args[1:] {
		switch arg {
		case "-a", "--all":
			showAll = true
		case "-f", "--files":
			filesOnly = true
		case "-h", "--help":
			fmt.Println("Usage: peek [options] [path]")
			fmt.Println("  -a, --all     show hidden files")
			fmt.Println("  -f, --files   files only")
			fmt.Println("  -h, --help    this message")
			return
		default:
			if !strings.HasPrefix(arg, "-") {
				target = arg
			}
		}
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		fmt.Fprintln(os.Stderr, errStyle.Render("  error: "+err.Error()))
		os.Exit(1)
	}

	var dirs, files []entry
	for _, e := range entries {
		name := e.Name()
		isDot := strings.HasPrefix(name, ".")

		if isDot && !showAll {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		isDir := e.IsDir()
		isSym := e.Type()&os.ModeSymlink != 0

		if isSym {
			resolved, err := filepath.EvalSymlinks(filepath.Join(target, name))
			if err == nil {
				ri, err := os.Stat(resolved)
				if err == nil {
					isDir = ri.IsDir()
				}
			}
		}

		ext := ""
		if !isDir {
			ext = strings.TrimPrefix(filepath.Ext(name), ".")
		}

		it := entry{
			name:  name,
			isDir: isDir,
			isSym: isSym,
			size:  info.Size(),
			dot:   isDot,
			ext:   ext,
		}

		if isDir && !filesOnly {
			dirs = append(dirs, it)
		} else if !isDir {
			files = append(files, it)
		}
	}

	sortEntries := func(items []entry) {
		sort.Slice(items, func(i, j int) bool {
			return strings.ToLower(items[i].name) < strings.ToLower(items[j].name)
		})
	}
	sortEntries(dirs)
	sortEntries(files)

	if len(dirs) == 0 && len(files) == 0 {
		fmt.Println(countStyle.Render("  empty"))
		return
	}

	// Terminal width
	width := 80
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		width = w
	}

	// Files-only mode: single panel
	if filesOnly || len(dirs) == 0 {
		renderSingle(files, "files", width)
		printFooter(len(dirs), len(files))
		return
	}

	// No files: single panel dirs
	if len(files) == 0 {
		renderSingle(dirs, "dirs", width)
		printFooter(len(dirs), len(files))
		return
	}

	// Side-by-side
	gap := 4
	panelW := (width - gap) / 2

	leftLines := buildDirPanel(dirs, panelW)
	rightLines := buildFilePanel(files, panelW)

	// Pad to equal height
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}
	for len(leftLines) < maxLines {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxLines {
		rightLines = append(rightLines, "")
	}

	gapStr := strings.Repeat(" ", gap)
	fmt.Println()
	for i := 0; i < maxLines; i++ {
		left := padToWidth(leftLines[i], panelW)
		fmt.Println(left + gapStr + rightLines[i])
	}
	fmt.Println()
	printFooter(len(dirs), len(files))
}

func buildDirPanel(dirs []entry, panelW int) []string {
	var lines []string

	// Header
	label := headerStyle.Render("dirs")
	lineW := panelW - visLen(label) - 2
	if lineW < 2 {
		lineW = 2
	}
	lines = append(lines, "  "+label+" "+borderStyle.Render(strings.Repeat("─", lineW)))
	lines = append(lines, "")

	nameMax := panelW - 4
	if nameMax > maxNameLen {
		nameMax = maxNameLen
	}

	for _, d := range dirs {
		name := truncate(d.name, nameMax)
		switch {
		case d.isSym:
			lines = append(lines, "  "+symNameStyle.Render(name))
		case d.dot:
			lines = append(lines, "  "+dotDirStyle.Render(name))
		default:
			lines = append(lines, "  "+dirNameStyle.Render(name))
		}
	}

	return lines
}

func buildFilePanel(files []entry, panelW int) []string {
	var lines []string

	// Header
	label := headerStyle.Render("files")
	lineW := panelW - visLen(label) - 2
	if lineW < 2 {
		lineW = 2
	}
	lines = append(lines, "  "+label+" "+borderStyle.Render(strings.Repeat("─", lineW)))
	lines = append(lines, "")

	nameMax := panelW - 4
	if nameMax > maxNameLen {
		nameMax = maxNameLen
	}

	for _, f := range files {
		name := truncate(f.name, nameMax)
		switch {
		case f.isSym:
			lines = append(lines, "  "+symNameStyle.Render(name))
		case f.dot:
			lines = append(lines, "  "+dotFileStyle.Render(name))
		default:
			lines = append(lines, "  "+fileNameStyle.Render(name))
		}

		// Size subtitle
		parts := []string{humanSize(f.size)}
		if f.ext != "" {
			parts = append(parts, f.ext)
		}
		lines = append(lines, "  "+subStyle.Render("  "+strings.Join(parts, " · ")))
	}

	return lines
}

func renderSingle(items []entry, label string, width int) {
	lineW := width - visLen(label) - 4
	if lineW < 2 {
		lineW = 2
	}

	fmt.Println()
	fmt.Println("  " + headerStyle.Render(label) + " " + borderStyle.Render(strings.Repeat("─", lineW)))
	fmt.Println()

	nameMax := width - 6
	if nameMax > maxNameLen {
		nameMax = maxNameLen
	}

	isFile := label == "files"

	for _, it := range items {
		name := truncate(it.name, nameMax)
		switch {
		case it.isSym:
			fmt.Println("  " + symNameStyle.Render(name))
		case it.dot && it.isDir:
			fmt.Println("  " + dotDirStyle.Render(name))
		case it.isDir:
			fmt.Println("  " + dirNameStyle.Render(name))
		case it.dot:
			fmt.Println("  " + dotFileStyle.Render(name))
		default:
			fmt.Println("  " + fileNameStyle.Render(name))
		}

		if isFile {
			parts := []string{humanSize(it.size)}
			if it.ext != "" {
				parts = append(parts, it.ext)
			}
			fmt.Println("  " + subStyle.Render("  "+strings.Join(parts, " · ")))
		}
	}
}

func printFooter(dirCount, fileCount int) {
	parts := []string{}
	if dirCount > 0 {
		s := fmt.Sprintf("%d dir", dirCount)
		if dirCount > 1 {
			s += "s"
		}
		parts = append(parts, s)
	}
	if fileCount > 0 {
		s := fmt.Sprintf("%d file", fileCount)
		if fileCount > 1 {
			s += "s"
		}
		parts = append(parts, s)
	}
	fmt.Println("  " + countStyle.Render(strings.Join(parts, ", ")))
	fmt.Println()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// padToWidth pads a string (which may contain ANSI codes) to a visual width
func padToWidth(s string, width int) string {
	vl := visLen(s)
	if vl >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vl)
}

// visLen returns the visible length of a string, stripping ANSI escape codes
func visLen(s string) int {
	n := 0
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		n++
	}
	return n
}

func humanSize(b int64) string {
	if b == 0 {
		return "0 B"
	}
	units := []string{"B", "K", "M", "G", "T"}
	i := int(math.Log(float64(b)) / math.Log(1024))
	if i >= len(units) {
		i = len(units) - 1
	}
	val := float64(b) / math.Pow(1024, float64(i))
	if i == 0 {
		return fmt.Sprintf("%d B", b)
	}
	if val >= 10 {
		return fmt.Sprintf("%d %s", int(val), units[i])
	}
	return fmt.Sprintf("%.1f %s", val, units[i])
}
