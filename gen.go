package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"git.universelle.science/juju/amber"
	"github.com/russross/blackfriday"
)

// Site structure : FOLDER
type FOLDER struct {
	Path     string        // part of path relatif to SRC and OUT
	Name     string        // navigation name
	outfiles []os.FileInfo // (extra) files in Out
	srcfiles []os.FileInfo // Source files

	// Navigation links
	Subdirs []*FOLDER // subdirectories
	Pages   PAGES     // pages in this folder
	index   *PAGE     // index page
}

// Site structure : Page
type PAGE struct {
  Root    *FOLDER // shortcut to root folder
	Folder  *FOLDER // contening folder
	SrcName string  // source .md file name
	DstName string  // destination (slug) name

	PubTime time.Time
	ModTime time.Time
	Prev    *PAGE
	Next    *PAGE
	Up      *PAGE

	Meta    TemplateData
	Content template.HTML
	buf     *bytes.Buffer
}
type PAGES []*PAGE

// Site data
type Site struct {
	RootFOLDER *FOLDER // root FOLDER
	Navigation string  // navigation HTML
}

// return the full Out path
func (folder *FOLDER) GetOutDir() string {
	return filepath.Join(PublicDir, folder.Path)
}

// return the full Src path
func (folder *FOLDER) GetSrcDir() string {
	return filepath.Join(PostsDir, folder.Path)
}

var (
	//postTpl   *template.Template // The one and only compiled post template
	postTpls  map[string]*template.Template // [templateName]=*compiledTemplate
	postTplNm = "post.amber"                // The amber post template file name (native Go are compiled using ParseGlob)
	site      = Site{}

	funcs = template.FuncMap{
		"fmttime": func(t time.Time, f string) string {
			return t.Format(f)
		},
		"navmenu": func() string {
			return NavigationMenu()
		},
	}
)

func init() {
	// Add the custom functions to Amber in the init(), since this is global
	// (package) state in my Amber fork.
	amber.AddFuncs(funcs)
}

// Sort pages in same folder
func (p PAGES) Less(i, j int) bool { return p[i].PubTime.Before(p[j].PubTime) }
func (p PAGES) Len() int           { return len(p) }
func (p PAGES) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Compile the tempalte directory
func compileTemplates() (err error) {
	var exists bool
	tmptpl, err := amber.CompileDir(TemplatesDir, amber.DefaultDirOptions, amber.DefaultOptions)
	if err != nil {
		return
	}
	postTpls = tmptpl
	postTplNm = "post"
	if _, exists = postTpls[postTplNm]; !exists {
		return fmt.Errorf("error parsing templates: %v", err)
	}
	DEBUG("Directory compiled: %v", TemplatesDir)
	return nil
}

// scan a directory tree and generate outputs
func genPath(dir string) {
	site.RootFOLDER = FOLDERTree(".")
	site.BuildNavigation()
	site.RootFOLDER.BuildTree()
}

// Build a FOLDER tree from SRC directory tree
func FOLDERTree(dir string) *FOLDER {

	folder := FOLDER{Path: dir, Name: filepath.Base(dir)}

	files, err := ioutil.ReadDir(folder.GetSrcDir())
	if err != nil {
		WARN(err.Error())
		return nil
	}
	// walk all files first
	for _, fi := range files {
		if fi.IsDir() {
			if subfolder := FOLDERTree(filepath.Join(folder.Path, fi.Name())); subfolder != nil {
				folder.Subdirs = append(folder.Subdirs, subfolder)
			}
		} else {
			folder.srcfiles = append(folder.srcfiles, fi)
		}
	}

	return &folder
}

// XXX :  DEPRECIATED, TO BE DELETED
// from template
func NavigationMenu() string {
	site.BuildNavigation()
	return site.Navigation
}

// XXX :  DEPRECIATED, TO BE DELETED
// Build a simple navigation tree with ul/li
func (site *Site) BuildNavigation() {
	var f func(*FOLDER) string
	f = func(folder *FOLDER) string {
		html := ""
		for _, sfolder := range folder.Subdirs {
			html += " <li>" + sfolder.Name + "</li> "
			html += f(sfolder)
		}
		if len(html) > 0 {
			html = "  <ul>" + html + "</ul>  "
		}
		return html
	}
	site.Navigation = f(site.RootFOLDER)
}

// Build the site from FOLDER
func (folder *FOLDER) BuildTree() {

	// build all pages for current directory
	folder.PopulateOut()
	for _, fi := range folder.srcfiles {
		fname := fi.Name()
		if strings.HasPrefix(fi.Name(), ".") {
			// ignore hidden files
			continue
		}
		if matched, _ := regexp.MatchString(".*\\.md", fname); matched {
			folder.newPage(fname)
		} else {
			folder.copy(fname)
		}
	}

	// read metadata for all pages of current folder
	sort.Sort(PAGES(folder.Pages))

	// build sub-directories
	for _, fi := range folder.Subdirs {
		fi.BuildTree()
	}

	// buil all page for current folder
	for _, pa := range folder.Pages {
		folder.generateFile(pa, pa == folder.index)
	}

	// clean up
	folder.CleanOut()
}

// Copy as is a Src file to an Out file
func (folder *FOLDER) copy(src string) {
	fsrc := filepath.Join(folder.GetSrcDir(), src)
	fdst := filepath.Join(folder.GetOutDir(), src)

	inf, err := os.Open(fsrc)
	if err != nil {
		ERROR(err.Error())
		return
	}
	defer inf.Close()

	ouf, err := os.Create(fdst)
	if err != nil {
		ERROR(err.Error())
		return
	}
	defer ouf.Close()

	_, err = io.Copy(ouf, inf)
	if err != nil {
		ERROR(err.Error())
		return
	}
	folder.legit(src)
}

// Mark a file in Out as legit from Src, by deleting it from outfiles
func (folder *FOLDER) legit(src string) {
	nf := []os.FileInfo{}
	for _, f := range folder.outfiles {
		if f.Name() != src {
			nf = append(nf, f)
		}
	}
	folder.outfiles = nf
}

// Cleanup: delete any extra files in Pub not present in Post
func (folder *FOLDER) CleanOut() {
	for _, f := range folder.outfiles {
		os.Remove(filepath.Join(folder.GetOutDir(), f.Name()))
	}
	folder.Path = ""
	folder.outfiles = nil
}

// Populate pubContent from a PostsDir's sub dir
func (folder *FOLDER) PopulateOut() {
	outDir := folder.GetOutDir()

	os.MkdirAll(outDir, 0755)
	files, err := ioutil.ReadDir(outDir)
	if err != nil {
		WARN(err.Error())
		return
	}
	for _, f := range files {
		if !f.IsDir() {
			folder.outfiles = append(folder.outfiles, f)
		}
	}
}

// Clear the public directory, ignoring special files, subdirectories, and hidden (dot) files.
func clearPublicDir() error {
	// do nothing for now
	return nil
	// Clear the public directory, except subdirs and special files (favicon.ico & co.)
	fis, err := ioutil.ReadDir(PublicDir)
	if err != nil {
		return fmt.Errorf("error getting public directory files: %s", err)
	}
	for _, fi := range fis {
		if !fi.IsDir() && !strings.HasPrefix(fi.Name(), ".") {
		}
	}
	return nil
}

// Generate the whole site.
func generateSite() error {
	// First compile the template(s)
	if err := compileTemplates(); err != nil {
		DEBUG("template error")
		return err
	}
	genPath(PostsDir)
	return nil
}

// create newpage, fill with metadata, but don't render template yet
func (folder *FOLDER) newPage(mdf string) {
	var p PAGE = PAGE{
    Root:     site.RootFOLDER,
		Folder:   folder,
		SrcName:  mdf,
	}

	fpath := filepath.Join(folder.GetSrcDir(), mdf)
	f, err := os.Open(fpath)
	if err != nil {
		ERROR("Cannot open %v(%v)", fpath, err)
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	p.Meta, err = readFrontMatter(s)
	if err != nil {
		WARN("Cannot read meta from %v(%v)", fpath, err)
		return
	}

	p.DstName = getSlug(mdf)
	fi, _ := f.Stat()
	p.PubTime = fi.ModTime()
	p.ModTime = fi.ModTime()
	if dt, ok := p.Meta["Date"]; ok && len(dt) > 0 {
		pubdt, err := time.Parse(pubDtFmt[len(dt)], dt)
		if err == nil {
			p.PubTime = pubdt
		}
	}

	p.Meta["Slug"] = p.DstName
	p.Meta["PubTime"] = p.PubTime.Format("2006-01-02")
	p.Meta["ModTime"] = p.ModTime.Format("15:04")
	if _, ok := p.Meta["Index"]; ok {
		folder.index = &p
	}

	// Read rest of file
	p.buf = bytes.NewBuffer(nil)
	for s.Scan() {
		p.buf.WriteString(s.Text() + "\n")
	}
	folder.Pages = append(folder.Pages, &p)
}

// Generate the static HTML file for the post identified by the index.
func (folder *FOLDER) generateFile(p *PAGE, idx bool) {
	var w io.Writer

	// check if template exists
	tplName, ok := p.Meta["Template"]
	if !ok {
		tplName = "default"
	}
	var tpl *template.Template
	var ex bool

	if tpl, ex = postTpls[tplName]; !ex {
		ERROR("Template not found: %s", tplName)
		return
	}

	slug := p.Meta["Slug"]
	fw, err := os.Create(filepath.Join(folder.GetOutDir(), slug))
	if err != nil {
		ERROR("error creating output %s: %s", slug, err)
		return
	}
	defer fw.Close()

	// If this is the newest file, also save as index.html
	w = fw
	if idx {
		idxw, err := os.Create(filepath.Join(folder.GetOutDir(), "index.html"))
		if err != nil {
			ERROR("error creating static file index.html: %s", err)
			return
		}
		defer idxw.Close()
		w = io.MultiWriter(fw, idxw)
	}

	// format from mardown
	res := blackfriday.Markdown(p.buf.Bytes(), bfRender, bfExtensions)
	p.Content = template.HTML(res)

	tpl.ExecuteTemplate(w, tplName+".amber", p)
	folder.legit(slug)
}
