package main

import (
	"bytes"
	"regexp"
	"testing"
)

func TestWrap(t *testing.T) {
	var (
		src      = []byte("should wrapped")
		expected = []byte("(should wrapped)")
	)

	if got := wrap(src, parenStart, parenEnd); !bytes.Equal(expected, got) {
		t.Fatalf("expected: %s but got: %s", expected, got)
	}
}

func TestWrapRegex(t *testing.T) {
	var (
		expr     = regexp.MustCompile(`\\\((.*?)\\\)`)
		src      = []byte(`this \(should be unescaped\) always and \(that as well\).`)
		expected = []byte(`this (should be unescaped) always and (that as well).`)
	)

	if got := wrapRegex(expr, src, parenStart, parenEnd); !bytes.Equal(expected, got) {
		t.Fatalf("expected: %s but got: %s", expected, got)
	}
}

func TestResolvePathLink(t *testing.T) {
	var tests = []struct {
		in      string
		outFile string
		link    string
	}{
		{"relative.md", "relative.md", "relative"},
		{"responses/json.md", "responses/responses-json.md", "responses-json"},
		{"responses/sub/other.md", "responses/sub/responses-sub-other.md", "responses-sub-other"},
		{"../view/view.md", "view/view-view.md", "view-view"},
		{"../dependency-injection/inputs.md", "dependency-injection/dependency-injection-inputs.md", "dependency-injection-inputs"},
		{"../.gitbook/assets/image.png", "_assets/image.png", "/kataras/iris/wiki/_assets/image.png"},
	}

	wikiRepo := "/kataras/iris/wiki"

	for i, tt := range tests {
		if expected, got := tt.outFile, resolvePath(tt.in); expected != got {
			t.Fatalf("[%d] expected path: %s but got: %s", i, expected, got)
		}

		if expected, got := tt.link, resolveLink(tt.in, wikiRepo); expected != got {
			t.Fatalf("[%d] expected link: %s but got: %s", i, expected, got)
		}
	}
}
