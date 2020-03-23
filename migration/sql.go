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
