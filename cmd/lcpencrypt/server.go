// Copyright 2025 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

// lcpencrypt server mode

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

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
		log.Fatalf("Error adding watched directory %s: %v", c.InputPath, err)
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
			// Only listen to Create events to avoid duplicate processing on Write
			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Printf("File created: %s", event.Name)
				sem <- struct{}{} // block if 4 processes are already running
				wg.Add(1)
				go func(filePath string) {
					defer wg.Done()
					defer func() { <-sem }() // free up a slot in the semaphore

					// Wait for the file to be ready (not empty and stable)
					if err := waitForFileReady(filePath); err != nil {
						log.Errorf("Skipping file %s: %v", filePath, err)
						return
					}

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

// waitForFileReady waits until the file is not empty, its size is stable, and it is readable.
func waitForFileReady(filePath string) error {
	const (
		checkInterval = 1000 * time.Millisecond
		maxRetries    = 60 // Wait up to 60 seconds
	)

	var lastSize int64 = -1

	for i := 0; i < maxRetries; i++ {
		time.Sleep(checkInterval)

		info, err := os.Stat(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		// Check if file is not empty
		if info.Size() > 0 {
			// Check if size is stable
			if info.Size() == lastSize {
				// Try to open the file to ensure it is accessible for reading
				f, err := os.Open(filePath)
				if err == nil {
					f.Close()
					return nil
				}
			}
			lastSize = info.Size()
		}
	}
	return fmt.Errorf("timeout waiting for file to be ready")
}
