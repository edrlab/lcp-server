// Copyright 2024 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

// The LCP Server generates LCP licenses.
package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/go-chi/chi/v5"

	"github.com/edrlab/lcp-server/pkg/conf"
	"github.com/edrlab/lcp-server/pkg/stor"
)

// Server context
type Server struct {
	*conf.Config
	stor.Store
	Cert   *tls.Certificate
	Router *chi.Mux
}

func main() {

	s := Server{}

	// Initialize the configuration from a config file or/and environment variables
	c, err := conf.Init(os.Getenv("LCPSERVER_CONFIG"))
	if err != nil {
		log.Println("Configuration failed: " + err.Error())
		os.Exit(1)
	}
	s.Config = c

	s.initialize()

	// Set the log level and format
	if s.Config.LogLevel != "" {
		level, err := log.ParseLevel(s.Config.LogLevel)
		if err != nil {
			log.Println("Invalid log level specified, defaulting to debug")
			level = log.DebugLevel
		}
		log.SetLevel(level)
		log.SetFormatter(&log.TextFormatter{})
	}

	// Graceful shutdown
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(c.Port),
		Handler: s.Router,
	}

	// System signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Println("Server starting on port " + strconv.Itoa(c.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutdown requested, initiating graceful shutdown...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Error during shutdown: %v", err)
	}
	log.Println("Server halted.")
}

// Initialize sets the database, X509 certificate and routes
func (s *Server) initialize() {
	var err error

	// Init database
	s.Store, err = stor.Init(s.Config.Dsn)
	if err != nil {
		log.Println("Database setup failed: " + err.Error())
		os.Exit(1)
	}

	// Init X509 certificate
	var certFile, privKeyFile string
	if certFile = s.Config.Certificate.Cert; certFile == "" {
		log.Println("Provider certificate missing")
		os.Exit(1)

	}
	if privKeyFile = s.Config.Certificate.PrivateKey; privKeyFile == "" {
		log.Println("Private key missing")
		os.Exit(1)
	}
	cert, err := tls.LoadX509KeyPair(certFile, privKeyFile)
	if err != nil {
		log.Println("Loading X509 key pair failed: " + err.Error())
		os.Exit(1)

	}
	s.Cert = &cert

	// Init routes
	s.Router = s.setRoutes()
}
