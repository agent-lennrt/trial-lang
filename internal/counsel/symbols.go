package counsel

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/lennrt/trial-lang/internal/gregor"
)

type symbolRange struct {
	Start position `json:"start"`
	End   position `json:"end"`
}
type documentSymbol struct {
	Name           string           `json:"name"`
	Kind           int              `json:"kind"`
	Range          symbolRange      `json:"range"`
	SelectionRange symbolRange      `json:"selectionRange"`
	Children       []documentSymbol `json:"children,omitempty"`
}

// documentSymbols reads only the current buffer. Invalid syntax already has a
// diagnostic; it must not create a misleading partial outline.
func documentSymbols(source string) []documentSymbol {
	symbols := make([]documentSymbol, 0)
	program, err := gregor.Parse(source)
	if err != nil {
		return symbols
	}
	lines := strings.Split(source, "\n")
	end := position{Line: len(lines) - 1, Character: utf16Length(strings.TrimSuffix(lines[len(lines)-1], "\r"))}
	sourcePosition := func(line, column int) position {
		return position{Line: line - 1, Character: utf16Length(lines[line-1][:column-1])}
	}
	declaration := func(name string, kind, line int, span gregor.DeclarationSpan) documentSymbol {
		start := sourcePosition(line, span.Column)
		selectionEnd := sourcePosition(span.EndLine, span.EndColumn)
		return documentSymbol{Name: name, Kind: kind, Range: symbolRange{Start: start, End: end}, SelectionRange: symbolRange{Start: start, End: selectionEnd}}
	}
	for _, constant := range program.Constants {
		symbols = append(symbols, declaration(constant.Name, 14, constant.Line, constant.Span))
	}
	for _, exhibit := range program.Exhibits {
		symbols = append(symbols, declaration(exhibit.Name, 23, exhibit.Line, exhibit.Span))
	}
	for _, article := range program.Articles {
		symbols = append(symbols, declaration(fmt.Sprintf("ARTICLE %d", article.Number), 3, article.Line, article.Span))
	}
	for _, office := range program.Offices {
		symbol := declaration(office.Name, 12, office.Line, office.Span)
		for _, section := range office.Sections {
			symbol.Children = append(symbol.Children, declaration(fmt.Sprintf("SECTION %d", section.Number), 3, section.Line, section.Span))
		}
		symbols = append(symbols, symbol)
	}
	finalizeSymbolRanges(symbols)
	return symbols
}

// finalizeSymbolRanges orders declarations and bounds siblings and nested sections.
func finalizeSymbolRanges(symbols []documentSymbol) {
	sort.SliceStable(symbols, func(i, j int) bool { return positionBefore(symbols[i].Range.Start, symbols[j].Range.Start) })
	for i := range symbols {
		if i+1 < len(symbols) {
			symbols[i].Range.End = symbols[i+1].Range.Start
		}
		for j := range symbols[i].Children {
			child := &symbols[i].Children[j]
			child.Range.End = symbols[i].Range.End
			if j+1 < len(symbols[i].Children) {
				child.Range.End = symbols[i].Children[j+1].Range.Start
			}
		}
	}
}

func utf16Length(text string) int {
	length := 0
	for _, r := range text {
		length += utf16.RuneLen(r)
	}
	return length
}

func positionBefore(a, b position) bool {
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	return a.Character < b.Character
}
