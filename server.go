package main
/*
 * This is freesofware under 2-clause BSD license, See LICENSE file
 */

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"path/filepath"
	"time"

	"github.com/PuerkitoBio/ghost/handlers"
)

// Start serving the blog.
func run() {
	var (
		faviconPath  = filepath.Join(PublicDir, "favicon.ico")
		faviconCache = 2 * 24 * time.Hour
	)

	h := handlers.FaviconHandler(
		handlers.PanicHandler(
			handlers.LogHandler(
				handlers.GZIPHandler(
					http.FileServer(http.Dir(PublicDir)),
					nil),
				handlers.NewLogOptions(nil, handlers.Ldefault)),
			nil),
		faviconPath,
		faviconCache)

	// Assign the combined handler to the server.
	http.Handle("/", h)

	// Start it up.
	INFO("Listening on port %d", Options.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", Options.Port), nil); err != nil {
		FATAL(err.Error())
	}
}
