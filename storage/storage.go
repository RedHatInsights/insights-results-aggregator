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

package storage

import (
	"database/sql"
	_ "github.com/lib/pq"           // PostgreSQL database driver
	_ "github.com/mattn/go-sqlite3" // SQLite database driver
	"log"
)

// Storage represents an interface to any relational database based on SQL language
type Storage interface {
	Close() error
}

// Impl is an implementation of Storage interface
type Impl struct {
	connections *sql.DB
	driver      string
}

// New function creates and initializes a new instance of Storage interface
func New(driverName string, dataSourceName string) (Storage, error) {
	log.Printf("Making connection to data storage, driver=%s datasource=%s", driverName, dataSourceName)
	connections, err := sql.Open(driverName, dataSourceName)

	if err != nil {
		log.Fatal("Can not connect to data storage", err)
		return nil, err
	}

	return Impl{connections, driverName}, nil
}

// Close method closes the connection to database. Needs to be called at the end of application lifecycle.
func (storage Impl) Close() error {
	log.Println("Closing connection to data storage")
	if storage.connections != nil {
		err := storage.connections.Close()
		if err != nil {
			log.Fatal("Can not close connection to data storage", err)
			return err
		}
	}
	return nil
}
