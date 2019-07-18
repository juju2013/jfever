package main

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/howeyc/fsnotify"
)

const (
	// Sometimes many events can be triggered in succession for the same file
	// (i.e. Create followed by Modify, etc.). No need to rush to generate
	// the HTML, just wait for it to calm down before processing.
	watchEventDelay = 1 * time.Second
)

var (
	tempo    = make(chan time.Time, 100)
	generate = make(chan bool, 1)
)

// Create and start a watcher, watching both the posts and the templates directories.
func startWatcher() *fsnotify.Watcher {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		FATAL(err.Error())
	}
	go bufGenSite()
	go watch(w)
	// Watch the posts directory
	if err = w.Watch(PostsDir); err != nil {
		w.Close()
		FATAL(err.Error())
	}
	// Watch the templates directory
	if err = w.Watch(TemplatesDir); err != nil {
		w.Close()
		FATAL(err.Error())
	}
	return w
}

// Receive watcher events for the directories. All events require re-generating
// the whole site (because the template may display the n most recent posts, the
// next and previous post, etc.). It could be fine-tuned based on what data we give
// to the templates, but for now, lazy approach.
func watch(w *fsnotify.Watcher) {
	for {
		select {
		case ev := <-w.Event:
			// Regenerate the files after the delay, reset the delay if an event is triggered
			// in the meantime
			ext := filepath.Ext(ev.Name)
			// Care only about changes to markdown files in the Posts directory, or to
			// Amber or Native Go template files in the Templates directory.
			if strings.HasPrefix(ev.Name, PostsDir) && ext == ".md" {
				tempo <- time.Now()
			} else if strings.HasPrefix(ev.Name, TemplatesDir) && (ext == ".amber" || ext == ".html") {
				tempo <- time.Now()
			}

		case err := <-w.Error:
			ERROR("%v", err)
		}
	}
}

// Generate site with a buffered watch/expire channel
func bufGenSite() {
	hitLast := time.Now()
	for {
		select {
		case hitNow := <-tempo:
			if hitNow.Sub(hitLast) > watchEventDelay {
				go chExpire(generate)
			}
			hitLast = hitNow
		case <-generate:
			if err := generateSite(); err != nil {
  			ERROR("%v", err)
			} else {
				INFO("site generated")
			}
		}
	}
}

func chExpire(ch chan<- bool) {
	time.Sleep(watchEventDelay)
	ch <- true
}
