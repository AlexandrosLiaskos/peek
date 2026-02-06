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

const maxNameLen = 40

var (
	// Box border
	boxBorder = lipgloss.RoundedBorder()

	// Title inside box
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ff66")).
			Bold(true)

	// Dir names
	dirNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff66"))
	dotDirStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#005c2e"))

	// File names
	fileNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00cc55"))
	dotFileStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#005c2e"))

	// Size subtitle
	subStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#003d1a")).Italic(true)

	// Symlinks
	symNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffaa"))

	// Footer
	countStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#003d1a"))

	// Error
	errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff3334"))
)

type entry struct {
	name     string
	isDir    bool
	isSym    bool
	size     int64
	dot      bool
	ext      string
	subDirs  int
	subFiles int
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
		fmt.Fprintln(os.Stderr, errStyle.Render("error: "+err.Error()))
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
			// Count immediate children
			subEntries, err := os.ReadDir(filepath.Join(target, name))
			if err == nil {
				for _, se := range subEntries {
					if !showAll && strings.HasPrefix(se.Name(), ".") {
						continue
					}
					if se.IsDir() {
						it.subDirs++
					} else {
						it.subFiles++
					}
				}
			}
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
	// Sort files by decreasing size
	sort.Slice(files, func(i, j int) bool {
		return files[i].size > files[j].size
	})

	if len(dirs) == 0 && len(files) == 0 {
		fmt.Println(countStyle.Render("  empty"))
		return
	}

	// Terminal width
	width := 80
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		width = w
	}

	gap := 2
	// Border takes 2 chars each side + 2 padding = 6 per panel
	panelOuter := (width - gap) / 2
	innerW := panelOuter - 4 // 2 border + 2 padding

	if innerW < 20 {
		innerW = 20
	}

	nameMax := innerW - 4 // account for padding inside box
	if nameMax > maxNameLen {
		nameMax = maxNameLen
	}

	boxStyle := lipgloss.NewStyle().
		Border(boxBorder).
		BorderForeground(lipgloss.Color("#004d26")).
		Padding(0, 2).
		Width(innerW)

	// Build dir content
	dirContent := buildDirContent(dirs, nameMax)
	// Build file content
	fileContent := buildFileContent(files, nameMax)

	// Single panel modes
	if filesOnly || len(dirs) == 0 {
		wideInner := width - 6
		if wideInner < 20 {
			wideInner = 20
		}
		wideMax := wideInner - 2
		if wideMax > maxNameLen {
			wideMax = maxNameLen
		}
		wideBox := lipgloss.NewStyle().
			Border(boxBorder).
			BorderForeground(lipgloss.Color("#004d26")).
			Padding(0, 2).
			Width(wideInner)
		fc := buildFileContent(files, wideMax)
		panel := wideBox.Render(titleStyle.Render("FILES") + "\n" + fc)
		fmt.Println()
		fmt.Println(panel)
		fmt.Println()
		printFooter(len(dirs), len(files))
		return
	}

	if len(files) == 0 {
		wideInner := width - 6
		if wideInner < 20 {
			wideInner = 20
		}
		wideMax := wideInner - 2
		if wideMax > maxNameLen {
			wideMax = maxNameLen
		}
		wideBox := lipgloss.NewStyle().
			Border(boxBorder).
			BorderForeground(lipgloss.Color("#004d26")).
			Padding(0, 2).
			Width(wideInner)
		dc := buildDirContent(dirs, wideMax)
		panel := wideBox.Render(titleStyle.Render("DIRS") + "\n" + dc)
		fmt.Println()
		fmt.Println(panel)
		fmt.Println()
		printFooter(len(dirs), len(files))
		return
	}

	// Two panels side by side
	leftPanel := boxStyle.Render(titleStyle.Render("DIRS") + "\n" + dirContent)
	rightPanel := boxStyle.Render(titleStyle.Render("FILES") + "\n" + fileContent)

	joined := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, strings.Repeat(" ", gap), rightPanel)

	fmt.Println()
	fmt.Println(joined)
	fmt.Println()
	printFooter(len(dirs), len(files))
}

func buildDirContent(dirs []entry, nameMax int) string {
	var lines []string
	for _, d := range dirs {
		sub := dirSubtitle(d.subDirs, d.subFiles)
		nameLimit := nameMax - len(sub) - 2
		if nameLimit < 10 {
			nameLimit = 10
		}
		name := truncate(d.name, nameLimit)
		var styled string
		switch {
		case d.isSym:
			styled = symNameStyle.Render(name)
		case d.dot:
			styled = dotDirStyle.Render(name)
		default:
			styled = dirNameStyle.Render(name)
		}
		lines = append(lines, styled+"  "+subStyle.Render(sub))
	}
	return strings.Join(lines, "\n")
}

func buildFileContent(files []entry, nameMax int) string {
	var lines []string
	for _, f := range files {
		sz := humanSize(f.size)
		nameLimit := nameMax - len(sz) - 2
		if nameLimit < 10 {
			nameLimit = 10
		}
		name := truncate(f.name, nameLimit)
		var styled string
		switch {
		case f.isSym:
			styled = symNameStyle.Render(name)
		case f.dot:
			styled = dotFileStyle.Render(name)
		default:
			styled = fileNameStyle.Render(name)
		}
		lines = append(lines, styled+"  "+subStyle.Render(sz))
	}
	return strings.Join(lines, "\n")
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
	if max < 4 {
		max = 4
	}
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func dirSubtitle(subDirs, subFiles int) string {
	if subDirs == 0 && subFiles == 0 {
		return "empty"
	}
	var parts []string
	if subDirs > 0 {
		s := fmt.Sprintf("%d dir", subDirs)
		if subDirs > 1 {
			s += "s"
		}
		parts = append(parts, s)
	}
	if subFiles > 0 {
		s := fmt.Sprintf("%d file", subFiles)
		if subFiles > 1 {
			s += "s"
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ", ")
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
