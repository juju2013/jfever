package main

import (
	"time"

	"github.com/radovskyb/watcher"
)

const (
	// Sometimes many events can be triggered in succession for the same file
	// (i.e. Create followed by Modify, etc.). No need to rush to generate
	// the HTML, just wait for it to calm down before processing.
	watchEventDelay = 1 * time.Second
)

var (
	fwatcher *watcher.Watcher

	tempo    = make(chan time.Time, 100)
	generate = make(chan bool, 1)
)

// start watch and loop till the end of time
func beginWatch(paths ...string) {
	fwatcher = watcher.New()
	fwatcher.SetMaxEvents(1)
	fwatcher.FilterOps(watcher.Rename, watcher.Move, watcher.Create, watcher.Remove, watcher.Write)
	fwatcher.IgnoreHiddenFiles(true)
	fwatcher.Ignore("examples")

	go fwHandler()

	// Start by rebuild - and launch - your program
	rebuild()

	// The root directory for source to watch
	for _, path := range paths {
		if err := fwatcher.AddRecursive(path); err != nil {
			ERROR(err.Error())
		}
	}

	// start watch source change, will loop
	DEBUG("Begin file watch")
	if err := fwatcher.Start(watchEventDelay); err != nil {
		FATAL(err.Error())
	}
}

// Handle watch events such as file change, error or exit
func fwHandler() {
	for {
		select {
		case event := <-fwatcher.Event:
			DEBUG("Change :%v", event) // Print the event's info.
			go rebuild()
		case err := <-fwatcher.Error:
			WARN(err.Error())
		case <-fwatcher.Closed:
			return
		}
	}
}

func rebuild() {
	DEBUG("REBUILD...")
	generateSite()
}
