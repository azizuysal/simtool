package simulator

import (
	"strings"
	"testing"
)

func TestDetectContentLanguage(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "html doctype",
			content: "<!DOCTYPE html><html><body>hi</body></html>",
			want:    "html",
		},
		{
			name:    "html body tag mid-document",
			content: "   <div class=\"x\">stuff</div>",
			want:    "html",
		},
		{
			name:    "xml declaration with svg tag",
			content: `<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"></svg>`,
			want:    "svg",
		},
		{
			name:    "xml declaration with svg xmlns",
			content: `<?xml version="1.0"?><root xmlns="http://www.w3.org/2000/svg"></root>`,
			want:    "svg",
		},
		{
			name:    "generic xml (non-html, non-svg)",
			content: `<?xml version="1.0"?><config><key>value</key></config>`,
			want:    "xml",
		},
		{
			name:    "json object",
			content: `{"name": "thing", "value": 42}`,
			want:    "json",
		},
		{
			name:    "json array with object",
			content: `[{"a": 1}, {"b": 2}]`,
			want:    "json",
		},
		{
			name:    "unrecognized content returns empty",
			content: "just a line of plain prose with no markers",
			want:    "",
		},
		{
			name:    "empty string",
			content: "",
			want:    "",
		},
		{
			name:    "brace-wrapped text without colon is not json",
			content: "{not really json}",
			want:    "", // no tags, no colon — none of the branches match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectContentLanguage(tt.content)
			if got != tt.want {
				t.Errorf("detectContentLanguage(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestGetLexerForExtension_AliasFallbacks(t *testing.T) {
	// Each extension should resolve to a non-nil lexer via either the
	// direct `lexers.Match` lookup or the alias switch inside
	// getLexerForExtension.
	exts := []string{
		".h", ".hpp", ".hxx", ".m", ".mm",
		".yml", ".tsx", ".jsx", ".plist",
		".htm", ".html", ".podspec",
		".go", ".js", ".py",
	}
	for _, ext := range exts {
		t.Run(ext, func(t *testing.T) {
			if lex := getLexerForExtension(ext); lex == nil {
				t.Errorf("getLexerForExtension(%q) = nil, want a lexer", ext)
			}
		})
	}
}

func TestGetLexerForExtension_Cached(t *testing.T) {
	// Resolving twice must return the same cached instance.
	first := getLexerForExtension(".go")
	if first == nil {
		t.Fatal("first lookup returned nil")
	}
	second := getLexerForExtension(".go")
	if first != second {
		t.Error("getLexerForExtension(.go) returned different instances on second call, cache miss")
	}
}

func TestGetLexerForExtension_Unknown(t *testing.T) {
	// Extension with no match and no alias should return nil.
	if lex := getLexerForExtension(".this-is-not-a-real-extension-xyz"); lex != nil {
		t.Errorf("getLexerForExtension on unknown extension returned %v, want nil", lex)
	}
}

func TestGetSyntaxHighlightedLineWithLang_UsesDetectedLanguage(t *testing.T) {
	// With a blank extension but an explicit detected language, output
	// should still carry ANSI escapes.
	line := `<div class="x">hi</div>`
	out := GetSyntaxHighlightedLineWithLang(line, "", "html")
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("expected ANSI escape in highlighted html output, got %q", out)
	}
}

func TestGetSyntaxHighlightedLineWithLang_UnknownLanguageFallsThroughToExtension(t *testing.T) {
	// Unknown detected language but known extension should fall back to
	// extension-based resolution and still highlight.
	line := `package main`
	out := GetSyntaxHighlightedLineWithLang(line, ".go", "not-a-real-language")
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("expected ANSI escape after extension fallback, got %q", out)
	}
}

func TestGetSyntaxHighlightedLineWithLang_NoLexerReturnsInput(t *testing.T) {
	// No detected language and unknown extension → plain text returned.
	line := "some arbitrary content"
	out := GetSyntaxHighlightedLineWithLang(line, ".this-is-not-real", "")
	if out != line {
		t.Errorf("expected plain passthrough, got %q", out)
	}
}
