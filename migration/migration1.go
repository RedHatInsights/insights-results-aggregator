package migration

import (
	"database/sql"
)

var mig1 = Migration{
	StepUp: func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			CREATE TABLE report (
				org_id          INTEGER NOT NULL,
				cluster         VARCHAR NOT NULL UNIQUE,
				report          VARCHAR NOT NULL,
				reported_at     DATETIME,
				last_checked_at DATETIME,
				PRIMARY KEY(org_id, cluster)
			)`)
		return err
	},
	StepDown: func(tx *sql.Tx) error {
		_, err := tx.Exec(`DROP TABLE report`)
		return err
	},
}
