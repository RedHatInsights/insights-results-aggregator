/*
Copyright © 2020, 2021, 2022 Red Hat, Inc.

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

package types

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/RedHatInsights/insights-operator-utils/types"

	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type (
	// ValidationError validation error, for example when string is longer then expected
	ValidationError = types.ValidationError
	// ItemNotFoundError shows that item with id ItemID wasn't found in the storage
	ItemNotFoundError = types.ItemNotFoundError
)

// ErrOldReport is an error returned if a more recent already
// exists on the storage while attempting to write a report for a cluster.
var ErrOldReport = types.ErrOldReport

// ErrEmptyReport is an error returned if an empty report or a report with
// only empty attributes is found in the parsed message
var ErrEmptyReport = errors.New("empty report found in deserialized message")

// TableNotFoundError table not found error
type TableNotFoundError struct {
	tableName string
}

// Error returns error string
func (err *TableNotFoundError) Error() string {
	return fmt.Sprintf("no such table: %v", err.tableName)
}

// TableAlreadyExistsError represents table already exists error
type TableAlreadyExistsError struct {
	tableName string
}

// Error returns error string
func (err *TableAlreadyExistsError) Error() string {
	return fmt.Sprintf("table %v already exists", err.tableName)
}

// ForeignKeyError something violates foreign key error
type ForeignKeyError struct {
	TableName      string
	ForeignKeyName string

	// Details can reveal you information about specific item violating fk
	Details string
}

// Error returns error string
func (err *ForeignKeyError) Error() string {
	return fmt.Sprintf(
		`operation violates foreign key "%v" on table "%v"`, err.ForeignKeyName, err.TableName,
	)
}

// ConvertDBError converts sql errors to those defined in this package
func ConvertDBError(err error, itemID interface{}) error {
	if err == nil {
		return nil
	}

	if err == sql.ErrNoRows {
		if itemIDArray, ok := itemID.([]interface{}); ok {
			var strArray []string
			for _, item := range itemIDArray {
				strArray = append(strArray, fmt.Sprint(item))
			}

			itemID = strings.Join(strArray, "/")
		}

		return &ItemNotFoundError{ItemID: itemID}
	}

	err = convertPostgresError(err)

	return err
}
func regexGetFirstMatchOrLogError(regexStr, str string) string {
	return regexGetNthMatchOrLogError(regexStr, 1, str)
}

func regexGetNthMatchOrLogError(regexStr string, nMatch uint, str string) string {
	match, err := regexGetNthMatch(regexStr, nMatch, str)
	if err != nil {
		log.Error().
			Str("regex", regexStr).
			Str("str", str).
			Msg("unable to get first match from string")
		return ""
	}

	return match
}

func regexGetNthMatch(regexStr string, nMatch uint, str string) (string, error) {
	regex := regexp.MustCompile(regexStr)
	if !regex.MatchString(str) {
		return "", errors.New("regex doesn't match string")
	}

	matches := regex.FindStringSubmatch(str)
	if len(matches) < int(nMatch+1) {
		return "", errors.New("regexGetNthMatch unable to find match")
	}

	return matches[nMatch], nil
}

func convertPostgresError(err error) error {
	pqError, ok := err.(*pq.Error)
	if !ok {
		return err
	}

	// see https://www.postgresql.org/docs/current/errcodes-appendix.html to get the magic happening below
	switch pqError.Code {
	case pgDuplicateTableErrorCode: // duplicate_table
		return &TableAlreadyExistsError{
			tableName: regexGetFirstMatchOrLogError(`relation "(.+)" already exists`, pqError.Message),
		}
	case pgUndefinedTableErrorCode: // undefined_table
		return &TableNotFoundError{
			tableName: regexGetNthMatchOrLogError(`(table|relation) "(.+)" does not exist`, 2, pqError.Message),
		}
	case pgForeignKeyViolationErrorCode: // foreign_key_violation
		// for some reason field Table is filled not in all errors
		return &ForeignKeyError{
			TableName:      pqError.Table,
			ForeignKeyName: pqError.Constraint,
			Details:        pqError.Detail,
		}
	}

	return err
}
