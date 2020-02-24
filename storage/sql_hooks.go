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
	"context"
	"database/sql"
	sql_driver "database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gchaincl/sqlhooks"
	pq "github.com/lib/pq"
	sqlite3 "github.com/mattn/go-sqlite3"
)

type sqlHooks struct {
	SQLQueriesLogger *log.Logger
}

type sqlHooksKey int

const (
	sqlHooksKeyQueryBeginTime sqlHooksKey = iota
)

// LogFormatterString is format string for sql queries logging
// first arg is query string
// second arg is params array
const logFormatterString = "query `%+v` with params `%+v`"

// Before is called before the query was executed allowing yout to log what you asked db to do
func (h *sqlHooks) Before(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	jsonArgs, err := json.Marshal(args)
	if err == nil {
		h.SQLQueriesLogger.Printf(logFormatterString+"\n", query, string(jsonArgs))
	} else {
		h.SQLQueriesLogger.Printf(logFormatterString+"\n", query, args)
	}

	return context.WithValue(ctx, sqlHooksKeyQueryBeginTime, time.Now()), nil
}

// After is called after the query was executed showing only successful ones
// it allows you to see how long your query took
func (h *sqlHooks) After(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	beginTime := ctx.Value(sqlHooksKeyQueryBeginTime).(time.Time)

	jsonArgs, err := json.Marshal(args)
	if err == nil {
		h.SQLQueriesLogger.Printf(
			logFormatterString+" took %s\n",
			query, string(jsonArgs), time.Since(beginTime),
		)
	} else {
		h.SQLQueriesLogger.Printf(
			logFormatterString+" took %s\n",
			query, args, time.Since(beginTime),
		)
	}

	return ctx, nil
}

// InitAndGetSQLDriverWithLogs initializes driver with logging queries and returns driver's name
func InitAndGetSQLDriverWithLogs(driverName string, logger *log.Logger) (string, error) {
	var driver sql_driver.Driver

	switch driverName {
	case "sqlite", "sqlite3":
		driver = &sqlite3.SQLiteDriver{}
	case "postgres":
		driver = &pq.Driver{}
	default:
		return "", fmt.Errorf("driver %v is not supported", driverName)
	}

	// linear search is not gonna be an issue since there's not many drivers
	// and we call New() only ones/twice per process life
	foundHooksDriver := false

	for _, existingDriver := range sql.Drivers() {
		if existingDriver == driverName+"WithHooks" {
			foundHooksDriver = true
			break
		}
	}

	if !foundHooksDriver {
		sql.Register(driverName+"WithHooks", sqlhooks.Wrap(driver, &sqlHooks{
			SQLQueriesLogger: logger,
		}))
	}

	return driverName + "WithHooks", nil
}
