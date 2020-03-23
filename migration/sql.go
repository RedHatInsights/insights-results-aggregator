// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package migration

import "database/sql"

func withTransaction(db *sql.DB, txFunc func(*sql.Tx) error) (errOut error) {
	var tx *sql.Tx
	tx, errOut = db.Begin()
	if errOut != nil {
		return
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			// panic again
			panic(p)
		} else if errOut != nil {
			_ = tx.Rollback()
		} else {
			errOut = tx.Commit()
		}
	}()

	errOut = txFunc(tx)

	return
}
