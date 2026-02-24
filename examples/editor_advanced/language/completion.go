package language

import (
	"sort"
	"strings"
	"unicode"
)

// CompletionItemKind matches Monaco's CompletionItemKind enum.
const (
	CompletionKindFunction = 1
	CompletionKindVariable = 6
	CompletionKindConstant = 15
	CompletionKindKeyword  = 17
	CompletionKindModule   = 9
)

// CompletionItem is a single completion suggestion.
type CompletionItem struct {
	Label      string `json:"label"`
	Kind       int    `json:"kind"`
	Detail     string `json:"detail,omitempty"`
	InsertText string `json:"insertText,omitempty"`
	SortText   string `json:"sortText"`
}

var luaKeywords = []string{
	"and", "break", "do", "else", "elseif", "end",
	"false", "for", "function", "goto", "if", "in",
	"local", "nil", "not", "or", "repeat", "return",
	"then", "true", "until", "while",
}

// Complete returns completion items for the given cursor position.
func Complete(symbols *SymbolTable, line, col int, lineText string) []CompletionItem {
	// Determine what's before the cursor on this line.
	colIdx := col - 1 // 0-based index into lineText
	if colIdx < 0 {
		colIdx = 0
	}
	if colIdx > len(lineText) {
		colIdx = len(lineText)
	}
	before := lineText[:colIdx]

	// Check for dot completion: "tablename."
	if dotIdx := strings.LastIndex(before, "."); dotIdx >= 0 {
		tableName := extractIdentBefore(before, dotIdx)
		if tableName != "" {
			prefix := ""
			if dotIdx+1 < len(before) {
				prefix = before[dotIdx+1:]
			}
			return dotCompletion(tableName, prefix)
		}
	}

	// Check for colon completion: "object:"
	if colonIdx := strings.LastIndex(before, ":"); colonIdx >= 0 {
		tableName := extractIdentBefore(before, colonIdx)
		if tableName != "" {
			prefix := ""
			if colonIdx+1 < len(before) {
				prefix = before[colonIdx+1:]
			}
			return colonCompletion(tableName, prefix)
		}
	}

	// Identifier completion.
	prefix := extractPrefix(before)
	return identCompletion(symbols, line, col, prefix)
}

func dotCompletion(tableName, prefix string) []CompletionItem {
	var items []CompletionItem

	// Stdlib members.
	for _, e := range StdlibMembers(tableName) {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(e.Name), strings.ToLower(prefix)) {
			continue
		}
		items = append(items, CompletionItem{
			Label:    e.Name,
			Kind:     stdlibKindToCompletionKind(e.Kind),
			Detail:   e.Signature,
			SortText: "0-" + e.Name,
		})
	}

	return items
}

func colonCompletion(tableName, prefix string) []CompletionItem {
	// For colon completion, show stdlib members that are functions (method-style).
	var items []CompletionItem
	for _, e := range StdlibMembers(tableName) {
		if e.Kind != "function" {
			continue
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(e.Name), strings.ToLower(prefix)) {
			continue
		}
		items = append(items, CompletionItem{
			Label:    e.Name,
			Kind:     CompletionKindFunction,
			Detail:   e.Signature,
			SortText: "0-" + e.Name,
		})
	}
	return items
}

func identCompletion(symbols *SymbolTable, line, col int, prefix string) []CompletionItem {
	var items []CompletionItem
	lowerPrefix := strings.ToLower(prefix)

	// Scope-visible symbols.
	if symbols != nil {
		for _, sym := range symbols.VisibleAt(line, col) {
			name := sym.Name
			if prefix != "" && !strings.HasPrefix(strings.ToLower(name), lowerPrefix) {
				continue
			}
			// Skip stdlib globals here, they'll be added below with docs.
			if sym.Pos.Line == 0 {
				continue
			}

			item := CompletionItem{
				Label:    name,
				SortText: "0-" + name,
			}
			switch sym.Kind {
			case KindFunction:
				item.Kind = CompletionKindFunction
				item.Detail = "function " + name + "(" + sym.Detail + ")"
			case KindParam:
				item.Kind = CompletionKindVariable
				item.Detail = "parameter"
				item.SortText = "0-" + name
			case KindForVar:
				item.Kind = CompletionKindVariable
				item.Detail = "for variable"
				item.SortText = "0-" + name
			case KindLocal:
				item.Kind = CompletionKindVariable
				item.Detail = "local"
				item.SortText = "0-" + name
			case KindGlobal:
				item.Kind = CompletionKindVariable
				item.Detail = "global"
				item.SortText = "1-" + name
			}
			items = append(items, item)
		}
	}

	// Stdlib globals.
	for _, e := range StdlibGlobals() {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(e.Name), lowerPrefix) {
			continue
		}
		items = append(items, CompletionItem{
			Label:    e.Name,
			Kind:     stdlibKindToCompletionKind(e.Kind),
			Detail:   e.Signature,
			SortText: "2-" + e.Name,
		})
	}

	// Lua keywords.
	for _, kw := range luaKeywords {
		if prefix != "" && !strings.HasPrefix(kw, lowerPrefix) {
			continue
		}
		items = append(items, CompletionItem{
			Label:    kw,
			Kind:     CompletionKindKeyword,
			SortText: "3-" + kw,
		})
	}

	// Deduplicate: prefer user symbols over stdlib.
	items = dedup(items)

	sort.Slice(items, func(i, j int) bool {
		return items[i].SortText < items[j].SortText
	})

	return items
}

func dedup(items []CompletionItem) []CompletionItem {
	seen := make(map[string]int) // label → index of first occurrence
	var result []CompletionItem
	for _, item := range items {
		if idx, ok := seen[item.Label]; ok {
			// Keep the one with smaller sort text (higher priority).
			if item.SortText < result[idx].SortText {
				result[idx] = item
			}
			continue
		}
		seen[item.Label] = len(result)
		result = append(result, item)
	}
	return result
}

func extractIdentBefore(s string, dotIdx int) string {
	end := dotIdx
	start := end
	for start > 0 && isIdentChar(rune(s[start-1])) {
		start--
	}
	if start == end {
		return ""
	}
	return s[start:end]
}

func extractPrefix(s string) string {
	end := len(s)
	start := end
	for start > 0 && isIdentChar(rune(s[start-1])) {
		start--
	}
	return s[start:end]
}

func isIdentChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func stdlibKindToCompletionKind(kind string) int {
	switch kind {
	case "function":
		return CompletionKindFunction
	case "table":
		return CompletionKindModule
	case "value":
		return CompletionKindConstant
	default:
		return CompletionKindVariable
	}
}
