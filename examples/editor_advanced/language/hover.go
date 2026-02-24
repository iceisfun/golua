package language

import (
	"fmt"
	"strings"
	"unicode"
)

// HoverResult contains markdown content for display on hover.
type HoverResult struct {
	Contents string `json:"contents"`
}

// Hover returns hover information for the word at the given cursor position.
func Hover(symbols *SymbolTable, line, col int, sourceText string) *HoverResult {
	word, isBefore := wordAtPosition(sourceText, line, col)
	if word == "" {
		return nil
	}

	// Check for dotted access: "string.format", "math.pi", etc.
	dotted := ""
	if isBefore != "" {
		dotted = isBefore + "." + word
	}

	if dotted != "" {
		if e, ok := StdlibLookup(dotted); ok {
			return &HoverResult{
				Contents: formatStdlib(e),
			}
		}
	}

	// Check stdlib globals directly.
	if e, ok := StdlibLookup(word); ok {
		// Only use stdlib for globals (not table members without dot prefix).
		if e.Parent == "" {
			return &HoverResult{
				Contents: formatStdlib(e),
			}
		}
	}

	// Look up in scope-visible symbols.
	if symbols != nil {
		for _, sym := range symbols.VisibleAt(line, col) {
			if sym.Name == word && sym.Pos.Line > 0 {
				return &HoverResult{
					Contents: formatSymbol(sym),
				}
			}
		}
	}

	return nil
}

func formatStdlib(e StdlibEntry) string {
	var b strings.Builder
	b.WriteString("```lua\n")
	b.WriteString(e.Signature)
	b.WriteString("\n```\n")
	b.WriteString(e.Doc)
	return b.String()
}

func formatSymbol(sym *Symbol) string {
	var b strings.Builder
	b.WriteString("```lua\n")
	switch sym.Kind {
	case KindFunction:
		fmt.Fprintf(&b, "function %s(%s)", sym.Name, sym.Detail)
	case KindParam:
		fmt.Fprintf(&b, "(parameter) %s", sym.Name)
	case KindForVar:
		fmt.Fprintf(&b, "(for variable) %s", sym.Name)
	case KindLocal:
		fmt.Fprintf(&b, "local %s", sym.Name)
	case KindGlobal:
		fmt.Fprintf(&b, "(global) %s", sym.Name)
	}
	b.WriteString("\n```")
	return b.String()
}

// wordAtPosition extracts the identifier at the given 1-based line/col.
// Returns the word and the identifier before a dot (if any).
func wordAtPosition(source string, line, col int) (word string, tableBefore string) {
	lines := strings.Split(source, "\n")
	if line < 1 || line > len(lines) {
		return "", ""
	}
	lineText := lines[line-1]
	colIdx := col - 1 // 0-based
	if colIdx < 0 || colIdx > len(lineText) {
		return "", ""
	}

	// Find word boundaries.
	start := colIdx
	for start > 0 && isIdentRune(rune(lineText[start-1])) {
		start--
	}
	end := colIdx
	for end < len(lineText) && isIdentRune(rune(lineText[end])) {
		end++
	}

	if start == end {
		return "", ""
	}
	word = lineText[start:end]

	// Check if there's a dot before the word: "table.word"
	if start > 0 && lineText[start-1] == '.' {
		dotIdx := start - 1
		tStart := dotIdx
		for tStart > 0 && isIdentRune(rune(lineText[tStart-1])) {
			tStart--
		}
		if tStart < dotIdx {
			tableBefore = lineText[tStart:dotIdx]
		}
	}

	return word, tableBefore
}

func isIdentRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
