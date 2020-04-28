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
	"fmt"
	"regexp"

	"github.com/rs/zerolog/log"
)

// ItemNotFoundError shows that item with id ItemID wasn't found in the storage
type ItemNotFoundError struct {
	ItemID interface{}
}

// Error returns error string
func (e *ItemNotFoundError) Error() string {
	return fmt.Sprintf("Item with ID %+v was not found in the storage", e.ItemID)
}

type TableNotFoundError struct {
	tableName string
}

func (err *TableNotFoundError) Error() string {
	return fmt.Sprintf("no such table: %v", err.tableName)
}

// convertDBError converts dbs error to the
func convertDBError(err error) error {
	if err == nil {
		return nil
	}

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
			log.Error().
				Str("errString", errString).
				Msg("convertDBError unable to find table name")

			return &TableNotFoundError{tableName: ""}
		}

		return &TableNotFoundError{tableName: matches[1]}
	}

	return err
}
