package main

import (
	"log"
	"os"
)

func INFO(format string, args ...interface{}) {
	log.Printf("[INFO]"+format, args...)
}


func WARN(format string, args ...interface{}) {
	if Options.Debug {
		log.Printf("[WARN]"+format, args...)
	}
}

func DEBUG(format string, args ...interface{}) {
	if Options.Debug {
		log.Printf("[DEBUG]"+format, args...)
	}
}

func ERROR(format string, args ...interface{}) {
	log.Printf("[ERROR]"+format, args...)
}

func FATAL(format string, args ...interface{}) {
	log.Printf("[FATAL]"+format, args...)
	os.Exit(1)
}

func logInit() {
	// do any log init
}
