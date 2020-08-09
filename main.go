package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// TODO: add --version flag and version command.
// Variables set by go build.
var (
	// buildRevision is the build revision (docker commit string or git rev-parse HEAD).
	buildRevision = ""
	// buildTime is the build unix time (in seconds since 1970-01-01 00:00:00 UTC).
	buildTime = ""
)

var (
	slash      = []byte("/")
	newLine    = []byte("\n")
	parenStart = []byte("(")
	parenEnd   = []byte(")")

	bracketStart = []byte("[")
	bracketEnd   = []byte("]")
	verticalBar  = []byte("|")

	whitespace    = []byte(" ")
	asteriskEntry = []byte("* ")

	refPrefix  = []byte("> Reference: ")
	httpPrefix = []byte("http")

	markdownSuffix = []byte(".md")

	codeSnippet = []byte("```")
)

var (
	srcDir    = "./_testfiles"
	destDir   = "./_testoutput.wiki"
	wikiRepo  = "/kataras/iris/wiki"
	verbose   = false
	keepLinks = false
)

var (
	// Note: if async use atomic package for these:
	totalFilesParsedCount int // uint32
	totalFilesCopiedCount int
)

var (
	errNotResponsible = errors.New("not responsible")
	errSkipLine       = errors.New("skip line")
)

// Examples:
// $ gitbook-to-wiki -v ./iris-book ./iris-wiki-test.wiki /kataras/iris-wiki-test/wiki
// $ gitbook-to-wiki -v --keep-links --src=./_testfiles --dest=./_testoutput
func main() {
	flag.StringVar(&srcDir, "src", srcDir, "--src=./my_gitbook (source input)")
	flag.StringVar(&destDir, "dest", destDir, "--dest=./my_repo.wiki (destination output)")
	flag.StringVar(&wikiRepo, "remote", wikiRepo, "--remote=/me/my_repo/wiki (GitHub wiki page base)")
	flag.BoolVar(&verbose, "v", verbose, "-v (to enable verbose messages)")
	flag.BoolVar(&keepLinks, "keep-links", keepLinks, "--keep-links (to keep the files and links as they are)")
	flag.Parse()

	for i, arg := range flag.Args() {
		switch i {
		case 0:
			srcDir = arg
		case 1:
			destDir = arg
		case 2:
			wikiRepo = arg
		default:
			os.Stderr.WriteString("unknown argument " + arg)
			os.Exit(-1)
		}
	}

	os.MkdirAll(destDir, 0666)

	start := time.Now()
	err := filepath.Walk(srcDir, walkFn)
	if err != nil {
		log.Fatal(err)
	}

	finishDur := time.Since(start)
	logf("Total files parsed: %d", totalFilesParsedCount)
	logf("Total files copied: %d", totalFilesCopiedCount)
	logf("Time taken to complete: %s", finishDur)
}

func logf(format string, args ...interface{}) {
	if !verbose {
		return
	}

	log.Printf(format, args...)
}

func walkFn(inPath string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if info.IsDir() || !info.Mode().IsRegular() {
		if info.Name() == ".git" {
			logf("Skip <.git> directory")
			return filepath.SkipDir
		}

		return nil
	}

	f, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// relative path work after opening the file.
	rel, err := filepath.Rel(srcDir, inPath)
	if err != nil {
		return err
	}

	rel = filepath.ToSlash(rel)

	outPath := filepath.Join(destDir, resolvePath(rel))
	os.MkdirAll(filepath.Dir(outPath), 0666)

	outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer outFile.Close()

	if verbose {
		inPath = filepath.ToSlash(inPath)
		outPath = filepath.ToSlash(outPath)
	}

	switch filepath.Ext(outPath) {
	case ".md":
		if verbose {
			if path.Base(inPath) != path.Base(outPath) {
				logf("Parse <%s> as <%s>", inPath, outPath)
			} else {
				logf("Parse <%s>", inPath)
			}
		}

		err = parse(rel, f, outFile)
		if err == nil {
			totalFilesParsedCount++
		}
		return err
	default:

		logf("Copy <%s> to <%s>", inPath, outPath)
		_, err = io.Copy(outFile, f)
		if err == nil {
			totalFilesCopiedCount++
		}
		return err
	}
}

func parse(filename string, src io.Reader, dest io.Writer) error {
	var (
		out = bufio.NewWriter(dest)

		// nr = &nameResolver{
		// 	wikiRepo: "/kataras/iris/wiki",
		// }

		p = newParser(src)

		lineReplacers = []simpleLineReplacer{
			tocEntry(filename, p),
			unescapeParens,
			unescapePageRefs,
			unescapeLinks(wikiRepo),
		}
	)

	for {
		line, err := p.readLine()
		if err != nil {
			break
		}

		if bytes.HasPrefix(line, codeSnippet) { // start snippet, read and write without changes until
			out.Write(line)
			out.Write(newLine)
			for {
				line, err = p.readLine()
				if err != nil {
					break
				}

				out.Write(line)
				out.Write(newLine)

				if bytes.Equal(line, codeSnippet) { // end of the snippet.
					p.prevLine = p.prevLine[0:0]
					break
				}
			}

			continue
		}

		for _, rpl := range lineReplacers {
			line, err = rpl(line)
			if err != nil {
				// if err == errSkip {
				// 	continue readLoop
				// }
				if err == errNotResponsible {
					continue
				}

				if err == errSkipLine {
					break
				}

				return err
			}
		}

		// if last error was skip, no need to write a new line.
		if err == errSkipLine {
			continue
		}

		if len(line) > 0 {
			// If current and previous starts with >
			// then we must separate them with a second new line,
			// it's markdown thing, otherwise they act as one > .
			if line[0] == '>' && len(p.prevLine) > 0 && p.prevLine[0] == '>' {
				out.Write(newLine)
			}
			out.Write(line)
		}

		out.Write(newLine)

		// Could be inaccurate via a second call of p.readLine
		// inside of a line replacer, currently that happens to
		// SUMMARY.md file only so we are  safe.
		p.prevLine = line
	}

	return out.Flush()
}

// resolvePath returns output (to be saved) fullpath.
func resolvePath(name string) string {
	name = filepath.ToSlash(name)
	if keepLinks {
		return name
	}

	name = strings.ReplaceAll(name, "../", "") // remove all ../, we don't handle it atm.

	dir := path.Dir(name)
	base := path.Base(name)
	if dir == "." {
		dir = ""
	}

	switch base {
	case "README.md":
		return path.Join(dir, "Home.md")
	case "SUMMARY.md":
		return path.Join(dir, "_Sidebar.md")
	default:
		// ../.gitbook/assets/image.png
		// _assets/image.png
		if strings.HasPrefix(name, ".gitbook/assets") {
			return strings.ReplaceAll(name, ".gitbook/assets", "_assets")
		}

		// responses/json.md
		// responses/responses-json.md
		//
		// responses/sub/other.md
		// responses/sub/responses-sub-other.md
		if dir != "" {
			base = "-" + base
		}

		newBase := strings.ReplaceAll(dir, "/", "-") + base
		return path.Join(dir, newBase)
	}
}

// resolveLink returns the wiki section name of "name"
// or if it is asset, returns the full wiki link of the asset.
func resolveLink(name string, wikiRepo string) string {
	name = resolvePath(name)
	if keepLinks {
		return name
	}

	if strings.HasPrefix(name, "_assets") {
		return path.Join(wikiRepo, name)
	}

	name = strings.TrimSuffix(path.Base(name), string(markdownSuffix))
	return name
}

type parser struct {
	rd       *bufio.Reader
	prevLine []byte // outside set, in order to give access for line placers.
}

func newParser(r io.Reader) *parser {
	return &parser{
		rd: bufio.NewReader(r),
	}
}

func (p *parser) readLine() ([]byte, error) {
	var linePrefix []byte
	for {
		line, isPrefixed, err := p.rd.ReadLine()
		if err != nil {
			return nil, err
		}

		if isPrefixed {
			linePrefix = append(linePrefix, line...)
			continue
		}

		if len(linePrefix) > 0 { // not prefixed and has a prior line prefix, so it's the end of the big line.
			line = append(linePrefix, line...)
			linePrefix = linePrefix[0:0]
		}

		return line, nil
	}
}

func (p *parser) skipNextEmptyLine() bool {
	nextLine, err := p.rd.Peek(2)
	if err != nil {
		if err == io.EOF {
			return false
		}
		// Note: can fire EOF too.
		return false
	}

	isNewLine := len(nextLine) > 1 && nextLine[1] == newLine[0]
	if isNewLine {
		_, _ = p.readLine()
	}

	return isNewLine
}

type simpleLineReplacer func(line []byte) (result []byte, err error)

var unescapeParensRegex = regexp.MustCompile(`\\\((.*?)\\\)`)

func unescapeParens(line []byte) ([]byte, error) {
	return wrapRegex(unescapeParensRegex, line, parenStart, parenEnd), nil
}

var unescapePageRefRegex = regexp.MustCompile(`{% page-ref page="(.*?)" %}`)

func unescapePageRefs(src []byte) ([]byte, error) {
	result := make([]byte, len(src))
	copy(result, src)

	for _, submatches := range unescapePageRefRegex.FindAllSubmatch(src, -1) {
		// {% page-ref page="../view/view.md" %}
		// [View](../view/view.md)

		link := submatches[1]

		if baseIdx := bytes.LastIndex(link, slash); baseIdx != -1 && len(link)-1 > baseIdx {
			link = link[baseIdx+1:]
		}

		title := bytes.TrimSuffix(link, markdownSuffix)
		title = bytes.Title(title)

		start := refPrefix
		start = append(start, append(append(bracketStart, title...), bracketEnd...)...)
		start = append(start, parenStart...)
		end := parenEnd

		result = bytes.Replace(result, submatches[0], append(start, append(submatches[1], end...)...), 1)
	}

	return result, nil
}

var unescapeLinksRegex = regexp.MustCompile(`\[(.*?)]\(([^()]+)\)`)

func unescapeLinks(wikiRepo string) simpleLineReplacer {
	return func(src []byte) ([]byte, error) {
		if keepLinks {
			return src, nil
		}

		result := make([]byte, len(src))
		copy(result, src)

		// group 0: "[...](...)"
		// group 1: "title"
		// group 2: ("...")
		for _, submatches := range unescapeLinksRegex.FindAllSubmatch(src, -1) {
			name := submatches[2]
			if bytes.HasPrefix(name, httpPrefix) {
				continue
			}

			link := []byte(resolveLink(string(name), wikiRepo))
			if !bytes.HasSuffix(name, markdownSuffix) {
				// if it's not a page, it's a link to an asset:
				result = bytes.Replace(result, name, link, 1)
				// not 100% precise, it could replace a link outside of []() in the same line but we rly don't care about it atm, it's a good thing.
				continue
			}

			// It's a section link:
			// [JSON](responses/json.md) to
			// [[JSON|responses-json]]
			title := submatches[1]

			if len(title) == 0 {
				return nil, fmt.Errorf("Title is missing from: %s", submatches[0])
			}

			result = bytes.Replace(result, submatches[0], bytes.Join([][]byte{
				bracketStart,
				bracketStart,
				title,
				verticalBar,
				link,
				bracketEnd,
				bracketEnd,
			}, nil), 1)
		}

		return result, nil
	}
}

func tocEntry(filename string, p *parser) simpleLineReplacer {
	return func(src []byte) ([]byte, error) {
		if path.Base(filename) != "SUMMARY.md" {
			return src, errNotResponsible
		}

		src = bytes.TrimSpace(src)
		if len(src) == 0 {
			return nil, nil
		}

		if len(src) < 4 {
			return src, nil
		}

		defer p.skipNextEmptyLine() // skip next empty line, the _Sidebar.md should NOT have any line separators.

		if src[0] == '#' {
			if src[1] != '#' { // is not followed by #, so it's a space or text.
				// 1st level header, probably a "Table of Contents" thing,
				// remove it by skipping (no new line).
				return nil, errSkipLine
			}

			// 2nd level header
			// From:
			// ## Compression
			//
			// * [Index](link.md)
			// -------To---------
			// * Compression
			//		* [Index](link)
			if src[2] == ' ' {
				// next is empty char, e.g. "## "
				return append(asteriskEntry, src[3:]...), nil
			}
			return append(asteriskEntry, src[2:]...), nil
		}

		if src[0] == '*' {
			tab := bytes.Repeat(whitespace, 2)

			// if the prev line was:
			// "* " or "  * " then add two spaces, otherwise
			// it is a root * which could be translated from a "## header".
			// Example:
			// * [[What is Iris|Home]]
			// * 📌Getting started
			//   * [[Installation|getting-started-installation]]
			//   * [[Quick start|getting-started-quick-start]]
			if bytes.HasPrefix(p.prevLine, asteriskEntry) || bytes.HasPrefix(p.prevLine, bytes.Join([][]byte{
				tab,
				asteriskEntry,
			}, nil)) {
				return append(tab, src...), nil
			}
		}

		return src, nil
	}
}

func wrapRegex(regex *regexp.Regexp, src, start, end []byte) []byte {
	result := make([]byte, len(src))
	copy(result, src)
	for _, submatches := range regex.FindAllSubmatch(src, -1) {
		// \(text)\
		// start + text + end
		result = bytes.Replace(result, submatches[0], append(start, append(submatches[1], end...)...), 1)
	}

	return result
}

func wrap(src []byte, start, end []byte) []byte {
	return append(start, append(src, end...)...)
}

// Of course need cleanup but it works like a charm for my needs. However,
// if users ask to make it faster or perform code cleanup or add more features, as always,
// I am ready to fulfill their wishes.
