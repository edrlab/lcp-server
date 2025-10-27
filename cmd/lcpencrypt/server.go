// Copyright 2025 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

// lcpencrypt server mode

package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/fsnotify/fsnotify"
)

func activateServer(c Config) {
	// system signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// process files already present in the input directory
	processExistingFiles(c)

	go func() {
		// semaphore, limits processing to 4 concurrent files
		sem := make(chan struct{}, 4)
		watchFileChanges(ctx, c, &wg, sem)
	}()

	<-stop
	log.Println("Shutdown requested, initiating graceful shutdown...")
	cancel()  // signal the watcher to stop
	wg.Wait() // wait for ongoing processing to finish
	log.Println("Server halted.")
}

// processExistingFiles processes files already present in the input directory
// TODO: Move provider URI and input path to a map in config.
func processExistingFiles(c Config) {
	files, err := os.ReadDir(c.InputPath)
	if err != nil {
		log.Printf("Error reading directory: %v", err)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if file.Name() == ".DS_Store" {
			log.Printf("Ignoring .DS_Store file")
			continue
		}
		log.Printf("File found: %s", file.Name())
		err = processFile(c, file.Name())
		if err != nil {
			log.Errorf("Error processing file %s: %v", file.Name(), err)
		}
	}
}

// watchFileChanges monitors changes in the input directory
// TODO: Move provider URI and input path to a map in config.

func watchFileChanges(ctx context.Context, c Config, wg *sync.WaitGroup, sem chan struct{}) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error creating watcher: %v", err)
	}
	defer watcher.Close()

	err = watcher.Add(c.InputPath)
	if err != nil {
		log.Fatalf("Error adding directory: %v", err)
	}

	log.Printf("Monitoring directory: %s", c.InputPath)
	for {
		select {
		case <-ctx.Done():
			log.Println("Watcher stop requested.")
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				log.Printf("File modified or created: %s", event.Name)
				sem <- struct{}{} // block if 4 processes are already running
				wg.Add(1)
				go func(filePath string) {
					defer wg.Done()
					defer func() { <-sem }() // free up a slot in the semaphore
					fileName := filepath.Base(filePath)
					err = processFile(c, fileName)
					if err != nil {
						log.Errorf("Error processing file %s: %v", fileName, err)
					}
				}(event.Name)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Error watching: %v", err)
		}
	}
}
