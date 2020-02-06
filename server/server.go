/*
Copyright © 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
)

// Server represents any REST API HTTP server
type Server interface {
	Start() error
}

// Impl in an implementation of Server interface
type Impl struct {
	Config Configuration
}

// New constructs new implementation of Server interface
func New(configuration Configuration) Server {
	return Impl{Config: configuration}
}

func logRequestHandler(writer http.ResponseWriter, request *http.Request, nextHandler http.Handler) {
	log.Println("Request URI: " + request.RequestURI)
	log.Println("Request method: " + request.Method)
	nextHandler.ServeHTTP(writer, request)
}

// LogRequest - middleware for loging requests
func (s Impl) LogRequest(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			logRequestHandler(writer, request, nextHandler)
		})
}

func (server Impl) mainEndpoint(writer http.ResponseWriter, request *http.Request) {
	// TODO: just a stub!
	io.WriteString(writer, "Hello world!\n")
}

// Initialize perform the server initialization
func (server Impl) Initialize() error {
	address := server.Config.Address

	log.Println("Initializing HTTP server at", address)
	router := mux.NewRouter().StrictSlash(true)
	router.Use(server.LogRequest)

	// common REST API endpoints
	router.HandleFunc(server.Config.APIPrefix, server.mainEndpoint).Methods("GET")

	log.Println("Starting HTTP server at", address)
	err := http.ListenAndServe(address, router)

	if err != nil {
		log.Fatal("Unable to initialize HTTP server", err)
	}
	return nil
}

// Start starts server
func (server Impl) Start() error {
	server.Initialize()
	return nil
}
