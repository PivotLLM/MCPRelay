/******************************************************************************
 * Copyright (c) 2025 Tenebris Technologies Inc.                              *
 * See LICENSE for details.                                                   *
 ******************************************************************************/

// Package data provides thread-safe access to critical data
package data

import (
	"fmt"
	"io"
	"log"
	"sync"
)

// Logger is an alias for log.Logger
// This is helpful if we decided to use a different logger later
type Logger = *log.Logger

// Data is this package's object
// Critical data is not exported and must be accessed through methods
type Data struct {
	server  string       // server (protocol://host:port)
	sseURL  string       // sseURL (server + path)
	postURL string       // postURL (server + path)
	logger  Logger       // logger
	mutex   sync.RWMutex // Read/Write mutex
}

// New creates a new Data object
func New(logger Logger) *Data {
	data := &Data{logger: logger}

	// Protect against nil logger
	if data.logger == nil {
		data.logger = log.New(io.Discard, "", 0)
	}
	return data
}

func (d *Data) SetServer(server string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.server = server
	d.logger.Printf("Server set to %s", server)
}

func (d *Data) SetPostPath(path string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.postURL = fmt.Sprintf("%s%s", d.server, path)
	d.logger.Printf("Post URL set to %s", d.postURL)
}

func (d *Data) SetPostURL(url string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.postURL = url
	d.logger.Printf("Post URL set to %s", d.postURL)
}

func (d *Data) SetSSEPath(path string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.sseURL = fmt.Sprintf("%s%s", d.server, path)
	d.logger.Printf("SSE URL set to %s", d.sseURL)
}

func (d *Data) SetSSEURL(url string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.sseURL = url
	d.logger.Printf("SSE URL set to %s", d.sseURL)
}

func (d *Data) GetServer() string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.server
}

func (d *Data) GetSSEURL() string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.sseURL
}

func (d *Data) GetPostURL() string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.postURL
}
