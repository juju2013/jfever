package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/russross/blackfriday"
)

var (
	ErrEmptyPost          = fmt.Errorf("empty post file")
	ErrInvalidFrontMatter = fmt.Errorf("invalid front matter")
	ErrMissingFrontMatter = fmt.Errorf("missing front matter")
	bfExtensions          = 0
	bfRender              blackfriday.Renderer

	// Lookup table to find the format based on the length of the date in the front matter
	pubDtFmt = map[int]string{
		10: "2006-01-02",
		13: "2006-01-02 15h",
		14: "2006-01-02 15h",
		15: "2006-01-02 15:04",
		16: "2006-01-02 15:04",
		25: time.RFC3339,
	}
)

// The TemplateData structure contains all the relevant information passed to the
// template to generate the static HTML file.
type TemplateData map[string]string

// Post data contains all the relevant information about a post (ie. meta data)
// and also a TemplateData
type PostData struct {
	PubTime time.Time
	ModTime time.Time
	Recent  []*PostData
	Prev    *PostData
	Next    *PostData
	D       TemplateData
	Content template.HTML
}

// Initialize a custom HTML render
func initBF() {
	bfExtensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	bfExtensions |= blackfriday.EXTENSION_TABLES
	bfExtensions |= blackfriday.EXTENSION_FENCED_CODE
	bfExtensions |= blackfriday.EXTENSION_AUTOLINK
	bfExtensions |= blackfriday.EXTENSION_STRIKETHROUGH
	bfExtensions |= blackfriday.EXTENSION_SPACE_HEADERS

	htmlFlags := 0
	htmlFlags |= blackfriday.HTML_USE_XHTML
	htmlFlags |= blackfriday.HTML_USE_SMARTYPANTS
	htmlFlags |= blackfriday.HTML_SMARTYPANTS_FRACTIONS
	bfRender = blackfriday.HtmlRenderer(htmlFlags, "", "")
}

// All Posts readed, ready to be generated, make site Index (inter-link) here
// `all` is an ordered array of all posts to generate
// return the index post
func siteIndex(all []*PostData) (index int) {
	index = 0
	l := len(all)
	for i := l - 1; i >= 0; i-- {
		if i > 0 {
			all[i].Prev = all[i-1]
		}
		if i < l-1 {
			all[i].Next = all[i+1]
		}
		if _, ex := all[i].D["IndexPage"]; ex {
			index = i
		}
	}
	return
}

// Replace special characters to form a valid slug (post path)
var rxSlug = regexp.MustCompile(`[^a-zA-Z\-_0-9]`)

// Return a valid slug from the file name of the post.
func getSlug(fnm string) string {
	return rxSlug.ReplaceAllString(strings.Replace(fnm, filepath.Ext(fnm), "", 1), "-")
}

// Read the front matter from the post. If there is no front matter, this is
// not a valid post.
func readFrontMatter(s *bufio.Scanner) (TemplateData, error) {
	// make a clone of SiteData
	m := make(TemplateData)
	for k, v := range SiteMeta.meta {
		m[k] = v
	}

	// defaut template name if not specified
	m["Template"] = "default"

	// scan the front matter
	infm := false
	for s.Scan() {
		l := strings.Trim(s.Text(), " ")
		if l == "---" { // The front matter is delimited by 3 dashes
			if infm {
				// This signals the end of the front matter
				return m, nil
			} else {
				// This is the start of the front matter
				infm = true
			}
		} else if infm {
			sections := strings.SplitN(l, ":", 2)
			if len(sections) != 2 {
				// Invalid front matter line
				return nil, ErrInvalidFrontMatter
			}
			m[sections[0]] = strings.Trim(sections[1], " ")
		} else if l != "" {
			// No front matter, quit
			return nil, ErrMissingFrontMatter
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return nil, ErrEmptyPost
}

// Generate an Out file from Src file
func (cms *CMS) genContent(mdf string) (*PostData, error) {
	f, err := os.Open(filepath.Join(cms.GetSrcDir(), mdf))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	td, err := readFrontMatter(s)
	if err != nil {
		return nil, err
	}

	slug := getSlug(mdf)
	fi, _ := f.Stat()
	pubdt := fi.ModTime()
	if dt, ok := td["Date"]; ok && len(dt) > 0 {
		pubdt, err = time.Parse(pubDtFmt[len(dt)], dt)
		if err != nil {
			return nil, err
		}
	}

	lp := PostData{
		D: td,
	}

	td["Slug"] = slug
	td["PubTime"] = pubdt.Format("2006-01-02")
	lp.PubTime = pubdt
	lp.ModTime = fi.ModTime()
	td["ModTime"] = lp.ModTime.Format("15:04")

	// Read rest of file
	buf := bytes.NewBuffer(nil)
	for s.Scan() {
		buf.WriteString(s.Text() + "\n")
	}
	if err = s.Err(); err != nil {
		return nil, err
	}
	res := blackfriday.Markdown(buf.Bytes(), bfRender, bfExtensions)
	lp.Content = template.HTML(res)
	return &lp, nil
}
