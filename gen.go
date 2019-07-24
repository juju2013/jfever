package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
  "regexp"
	"sort"
	"strings"
	"time"

	"github.com/juju2013/amber"
)

// Content Management Struct
type CMS struct {
  path        string          // part of path relatif to SRC and OUT
  outfiles    []os.FileInfo   // (extra) files in Out
  subdirs     []*CMS            // subdirectories
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
  rootCMS     *CMS

	funcs = template.FuncMap{
		"fmttime": func(t time.Time, f string) string {
			return t.Format(f)
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
	postTpls, err = amber.CompileDir(TemplatesDir, amber.DefaultDirOptions, amber.DefaultOptions)
	if err != nil {
		return
	}
	postTplNm = "post"
	if _, exists = postTpls[postTplNm]; !exists {
		return fmt.Errorf("error parsing templates: %v", err)
	}
	return nil
}

// scan a directory tree and generate outputs
func genPath(dir string) {
  rootCMS = WalkGenPath(".")
}

// Walk traverse a directory tree and generate outputs
func WalkGenPath(dir string) *CMS {
  DEBUG("Processing %v", dir)
  
  cms := CMS{path: dir}
  cms.PopulateOut()
  subdirs := []os.FileInfo{}
  
  files, err := ioutil.ReadDir(cms.GetSrcDir())
  if err != nil {
    WARN(err.Error())
    return nil
  }
  // walk all files first
  for _, fi := range files {
    if fi.IsDir() {
      subdirs = append(subdirs, fi)
    } else {
      fname := fi.Name()
      DEBUG("\t... file %v\n", fname)
      if strings.HasPrefix(fi.Name(), ".") {
        continue
      }
      if matched, _ := regexp.MatchString(".*\\.md", fname); matched {
        cms.generate(fname)
      } else {
        cms.copy(fname)
      }
    }
  }
  
  // walk subdir then
  for _, d := range subdirs {
    if subcms := WalkGenPath(filepath.Join(cms.path, d.Name())); subcms != nil {
      cms.subdirs = append(cms.subdirs, subcms)
    }
  }
 
  cms.CleanOut()
  return &cms
}

// Generate an Out file from Src file
func (cms *CMS) generate(src string) {
  
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
    DEBUG("Going to delete %v", f.Name())
  }
  cms.path = ""
  cms.outfiles = nil
}
 
// Populate pubContent from a PostsDir's sub dir
func (cms *CMS) PopulateOut() {
  outDir := cms.GetOutDir()
  DEBUG("populating %v", outDir)

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

// Generate the whole site.
func generateSite() error {
	// First compile the template(s)
	if err := compileTemplates(); err != nil {
		return err
	}
  genPath(PostsDir)
  return nil
	// Now read the posts
	fis, err := ioutil.ReadDir(PostsDir)
	if err != nil {
		return err
	}
	// Get all posts.
	all, recent := getPosts(fis)
	// Delete current public directory files
	if err := clearPublicDir(); err != nil {
		return err
	}
	// Generate the static files
	index := siteIndex(all)
	for i, p := range all {
		if err := generateFile(p, i == index); err != nil {
			fmt.Printf("DEBUG: template %v genration failed (%v)\n", p.D["Slug"], err)
		}
	}
	// Generate the RSS feed
	return generateRss(recent)
}

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

// Generate the static HTML file for the post identified by the index.
func generateFile(td *PostData, idx bool) error {
	var w io.Writer

	// check if template exists
	tplName := td.D["Template"]
	var tpl *template.Template
	var ex bool

	if tpl, ex = postTpls[tplName]; !ex {
		return fmt.Errorf("Template not found: %s", tplName)
	}
	slug := td.D["Slug"]
	fw, err := os.Create(filepath.Join(PublicDir, slug))

	if err != nil {
		return fmt.Errorf("error creating static file %s: %s", slug, err)
	}
	defer fw.Close()

	// If this is the newest file, also save as index.html
	w = fw
	if idx {
		idxw, err := os.Create(filepath.Join(PublicDir, "index.html"))
		if err != nil {
			return fmt.Errorf("error creating static file index.html: %s", err)
		}
		defer idxw.Close()
		w = io.MultiWriter(fw, idxw)
	}
	return tpl.ExecuteTemplate(w, tplName+".amber", td)
}

