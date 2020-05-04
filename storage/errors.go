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
	"errors"
	"fmt"
	"regexp"

	"github.com/rs/zerolog/log"
)

// ErrOldReport is an error returned if a more recent already
// exists on the storage while attempting to write a report for a cluster.
var ErrOldReport = errors.New("More recent report already exists in storage")

// ItemNotFoundError shows that item with id ItemID wasn't found in the storage
type ItemNotFoundError struct {
	ItemID interface{}
}

// Error returns error string
func (e *ItemNotFoundError) Error() string {
	return fmt.Sprintf("Item with ID %+v was not found in the storage", e.ItemID)
}

// TableNotFoundError table now found error
type TableNotFoundError struct {
	tableName string
}

// Error returns error string
func (err *TableNotFoundError) Error() string {
	return fmt.Sprintf("no such table: %v", err.tableName)
}

// ForeignKeyError something violates foreign key error
// tableName and foreignKeyName can be empty for DBs not supporting it (SQLite)
type ForeignKeyError struct {
	tableName      string
	foreignKeyName string
}

// Error returns error string
func (err *ForeignKeyError) Error() string {
	return fmt.Sprintf(
		`operation violates foreign key "%v" on table "%v"`, err.foreignKeyName, err.tableName,
	)
}

// convertDBError converts dbs error to the
func convertDBError(err error) error {
	if err == nil {
		return nil
	}

	err = convertNoTableError(err)
	err = convertForeignKeyError(err)

	return err
}

func convertNoTableError(err error) error {
	errString := err.Error()

	sqliteRegex := regexp.MustCompile(`no such table: (.+)`)

	if sqliteRegex.MatchString(errString) {
		matches := sqliteRegex.FindStringSubmatch(errString)
		if len(matches) < 2 {
			log.Error().
				Str("errString", errString).
				Msg("convertDBError unable to find table name")

			return &TableNotFoundError{tableName: ""}
		}

		return &TableNotFoundError{tableName: matches[1]}
	}

	postgresRegex := regexp.MustCompile(`pq: relation "(.+)" does not exist`)
	if postgresRegex.MatchString(errString) {
		matches := postgresRegex.FindStringSubmatch(errString)
		if len(matches) < 2 {
			logRegexError(errString, "convertDBError unable to find table name")

			return &TableNotFoundError{tableName: ""}
		}

		return &TableNotFoundError{tableName: matches[1]}
	}

	return err
}

func convertForeignKeyError(err error) error {
	errString := err.Error()

	if errString == "FOREIGN KEY constraint failed" {
		return &ForeignKeyError{}
	}

	postgresRegex := regexp.MustCompile(
		`pq: .+? on table "(.+?)" violates foreign key constraint "(.+?)"`,
	)
	if postgresRegex.MatchString(errString) {
		matches := postgresRegex.FindStringSubmatch(errString)
		if len(matches) < 3 {
			logRegexError(errString, "convertDBError unable to find table and constraint name")

			return &ForeignKeyError{}
		}

		return &ForeignKeyError{
			tableName:      matches[1],
			foreignKeyName: matches[2],
		}
	}

	return err
}

func logRegexError(errString, msg string) {
	log.Error().
		Str("errString", errString).
		Msg(msg)
}
