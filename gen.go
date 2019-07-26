package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	_ "net/url"
	"os"
	"path/filepath"
  "regexp"
	_ "sort"
	"strings"
	"time"

	"github.com/juju2013/amber"
)

// Site data
type Site struct {
  RootCMS     *CMS            // root CMS
  Navigation  string          // navigation HTML
}

// Content Management Struct
type CMS struct {
  path        string          // part of path relatif to SRC and OUT
  name        string          // navigation name
  outfiles    []os.FileInfo   // (extra) files in Out
  srcfiles    []os.FileInfo   // Source files
  subdirs     []*CMS          // subdirectories
}

// return the full Out path
func (cms *CMS) GetOutDir() string {
  return filepath.Join(PublicDir, cms.path)
}

// return the full Src path
func (cms *CMS) GetSrcDir() string {
  return filepath.Join(PostsDir, cms.path)
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

// This type is a slice of *LongPost that implements the sort.Interface, to sort in PubTime order.
type sortablePosts []*PostData

func (s sortablePosts) Len() int           { return len(s) }
func (s sortablePosts) Less(i, j int) bool { return s[i].PubTime.Before(s[j].PubTime) }
func (s sortablePosts) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

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
  site.RootCMS = CMSTree(".")
  site.BuildNavigation()
  site.RootCMS.BuildTree()
}

// Build a CMS tree from SRC directory tree
func CMSTree(dir string) *CMS {
  
  cms := CMS{path: dir, name: filepath.Base(dir)}
  
  files, err := ioutil.ReadDir(cms.GetSrcDir())
  if err != nil {
    WARN(err.Error())
    return nil
  }
  // walk all files first
  for _, fi := range files {
    if fi.IsDir() {
      if subcms := CMSTree(filepath.Join(cms.path, fi.Name())); subcms != nil {
        cms.subdirs = append(cms.subdirs, subcms)
      }
    } else {
      cms.srcfiles = append(cms.srcfiles, fi)
    }
  }
  
  return &cms
}

// from template
func NavigationMenu() string {
  site.BuildNavigation()
  return site.Navigation
}

// Build a simple navigation tree with ul/li
func (site *Site) BuildNavigation() {
  var f func(*CMS) string
  f = func(cms *CMS) string {
    html:=""
    for _, scms := range cms.subdirs {
      html+=" <li>"+scms.name+"</li> "
      html+=f(scms)
    }
    if len(html)>0 {
      html="  <ul>"+html+"</ul>  "
    }
    return html
  }
  site.Navigation = f(site.RootCMS)
}

// Build the site from CMS
func (cms *CMS) BuildTree() {
  
  // build all pages for current directory
  cms.PopulateOut()
  for _, fi := range cms.srcfiles {
    fname := fi.Name()
    if strings.HasPrefix(fi.Name(), ".") {
      continue
    }
    if matched, _ := regexp.MatchString(".*\\.md", fname); matched {
      cms.generate(fname)
    } else {
      cms.copy(fname)
    }
  }
  
  // build sub-directories
  for _, fi := range cms.subdirs {
    fi.BuildTree()
  }

  // clean up 
  cms.CleanOut()
}

// Copy as is a Src file to an Out file
func (cms *CMS) copy(src string) {
  fsrc := filepath.Join(cms.GetSrcDir(), src)
  fdst := filepath.Join(cms.GetOutDir(), src)
  
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
  cms.legit(src)
}

// Mark a file in Out as legit from Src, by deleting it from outfiles
func (cms *CMS) legit(src string) {
  nf := []os.FileInfo{}
  for _, f := range cms.outfiles {
    if f.Name() != src {
      nf = append(nf, f)
    }
  }
  cms.outfiles = nf
}

// Cleanup: delete any extra files in Pub not present in Post
func (cms *CMS) CleanOut() {
  for _, f := range cms.outfiles {
    os.Remove(filepath.Join(cms.GetOutDir(), f.Name()))
  }
  cms.path = ""
  cms.outfiles = nil
}
 
// Populate pubContent from a PostsDir's sub dir
func (cms *CMS) PopulateOut() {
  outDir := cms.GetOutDir()

  os.MkdirAll(outDir, 0755)
  files, err := ioutil.ReadDir(outDir)
  if err != nil {
    WARN(err.Error())
    return
  }
  for _, f := range files {
    if ! f.IsDir() {
      cms.outfiles = append(cms.outfiles, f)
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

/*
func getPosts(fis []os.FileInfo) (all, recent []*PostData) {
	all = make([]*PostData, 0, len(fis))
	for _, fi := range fis {
    DEBUG("Generating %v...", fi.Name())
		lp, err := newPost(fi)
		if err == nil {
			all = append(all, lp)
		} else {
			WARN("post ignored: %s; error: %s\n", fi.Name(), err)
		}
	}

	// Then sort in reverse order (newer first)
	sort.Sort(sort.Reverse(sortablePosts(all)))
	cnt := Options.RecentPostsCount
	if l := len(all); l < cnt {
		cnt = l
	}
	// Slice to get only recent posts
	recent = all[:cnt]
	return
}
*/
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

/*
// Creates the rss feed from the recent posts.
func generateRss(td []*PostData) error {
	r := NewRss(Options.SiteName, Options.TagLine, Options.BaseURL)
	base, err := url.Parse(Options.BaseURL)
	if err != nil {
		return fmt.Errorf("error parsing base URL: %s", err)
	}
	for _, p := range td {
		u, err := base.Parse((p.D["Slug"]))
		if err != nil {
			return fmt.Errorf("error parsing post URL: %s", err)
		}
		r.Channels[0].AppendItem(NewRssItem(
			p.D["Title"],
			u.String(),
			p.D["Description"],
			p.D["Author"],
			"",
			p.PubTime))
	}
	return r.WriteToFile(filepath.Join(PublicDir, "rss"))
}
*/

func (cms *CMS) generate(mdf string) {
  if data, err := cms.genContent(mdf); err == nil {
    cms.generateFile(data, false)
  }
}


// Generate the static HTML file for the post identified by the index.
func (cms *CMS) generateFile(td *PostData, idx bool) {
	var w io.Writer

	// check if template exists
	tplName, ok := td.D["Template"]
  if ! ok {
    tplName="default"
  }
	var tpl *template.Template
	var ex bool

	if tpl, ex = postTpls[tplName]; !ex {
		ERROR("Template not found: %s", tplName)
    return
	}

	slug := td.D["Slug"]
	fw, err := os.Create(filepath.Join(cms.GetOutDir(), slug))
	if err != nil {
		ERROR("error creating static file %s: %s", slug, err)
    return
	}
	defer fw.Close()

	// If this is the newest file, also save as index.html
	w = fw
	if idx {
		idxw, err := os.Create(filepath.Join(cms.GetOutDir(), "index.html"))
		if err != nil {
			ERROR("error creating static file index.html: %s", err)
      return
		}
		defer idxw.Close()
		w = io.MultiWriter(fw, idxw)
	}
	tpl.ExecuteTemplate(w, tplName+".amber", td)
  cms.legit(slug)
}
