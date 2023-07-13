package main

import (
	"database/sql"
	"os"

	_ "github.com/lib/pq"
)

const SQL_VERSION_0 = "a45a9821-8e0d-4126-8d99-0543e7f1f8f7"

type SQLDriver struct {
	db        *sql.DB
	rows      *sql.Rows
	index     int
	rollbacks []*SQLLedgerRow
}

type SQLLedgerRow struct {
	index    int
	forward  string
	backward string
}

func ConnectSQL(databaseURL string) Driver {
	Step(StepConnect, databaseURL)

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		panic(err)
	}

	driver := SQLDriver{db: db, rows: nil, index: 0, rollbacks: []*SQLLedgerRow{}}
	driver.verifySledgerVersion()
	driver.verifySledgerTable()
	driver.createRowCursor()

	return &driver
}

func (driver *SQLDriver) verifySledgerVersion() {
	driver.createSledgerVersion()

	version := driver.getSledgerVersion()

	if version == "" {
		driver.setSledgerVersion()
		version = SQL_VERSION_0
	}

	if version != SQL_VERSION_0 {
		Step(StepError, "Unsupported sledger version. Please use the correct version of sledger.")
		os.Exit(1)
	}
}

func (driver *SQLDriver) createSledgerVersion() {
	Step(StepSetup, "Ensuring sledger_version table exists...")
	rows, err := driver.db.Query(`create table if not exists "sledger_version" ("sledger_version" text)`)
	if err != nil {
		panic(err)
	}
	rows.Close()
}

func (driver *SQLDriver) getSledgerVersion() string {
	Step(StepSetup, "Getting sledger_version...")
	rows, err := driver.db.Query(`select "sledger_version" from "sledger_version" limit 1`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	if rows.Next() {
		var version string

		if err := rows.Scan(&version); err != nil {
			panic(err)
		}

		return version
	}

	return ""
}

func (driver *SQLDriver) setSledgerVersion() {
	Step(StepSetup, "Setting sledger_version...")
	rows, err := driver.db.Query("insert into sledger_version values ($1)", SQL_VERSION_0)
	if err != nil {
		panic(err)
	}
	rows.Close()
}

func (driver *SQLDriver) verifySledgerTable() {
	Step(StepSetup, "Creating sledger table if it does not exist...")
	rows, err := driver.db.Query(`create table if not exists "sledger" ("index" bigint not null, "forward" text not null, "backward" text not null, "timestamp" timestamp not null default now())`)
	if err != nil {
		panic(err)
	}
	rows.Close()
}

func (driver *SQLDriver) createRowCursor() {
	rows, err := driver.db.Query(`select "index", "forward", "backward" from "sledger" order by "index"`)
	if err != nil {
		panic(err)
	}

	driver.rows = rows
}

func (driver *SQLDriver) Apply(forward string, backward string) {
	if !driver.rows.Next() {
		driver.doForward(forward, backward)
		driver.index++
		return
	}

	actualChange := driver.scan()

	if actualChange.index != driver.index {
		Step(StepError, "Expected index %v, got index %v.", driver.index, actualChange.index)
		os.Exit(1)
	}

	if actualChange.forward != forward {
		Step(StepError, "Database does not match migration.\n Database: %v\nMigration: %v", actualChange.forward, forward)
		os.Exit(1)
	}

	Step(StepSkip, forward)
	driver.index++
}

func (driver *SQLDriver) doForward(forward string, backward string) {
	Step(StepForward, forward)

	tx, err := driver.db.Begin()
	if err != nil {
		panic(err)
	}
	defer tx.Rollback()

	rows, err := driver.db.Query(forward)
	if err != nil {
		panic(err)
	}
	rows.Close()

	rows, err = driver.db.Query(`insert into "sledger" ("index", "forward", "backward") values ($1, $2, $3)`, driver.index, forward, backward)

	if err != nil {
		panic(err)
	}
	rows.Close()

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}

func (driver *SQLDriver) Rollback() {
	for driver.rows.Next() {
		driver.rollbacks = append([]*SQLLedgerRow{driver.scan()}, driver.rollbacks...)
	}

	for _, rollback := range driver.rollbacks {
		if rollback.backward == "" {
			Step(StepError, "Missing rollback command, cannot rollback.")
			os.Exit(1)
		}

		Step(StepRollback, rollback.backward)

		tx, err := driver.db.Begin()
		if err != nil {
			panic(err)
		}
		defer tx.Rollback()

		rows, err := tx.Query(rollback.backward)
		if err != nil {
			panic(err)
		}
		rows.Close()

		rows, err = tx.Query("delete from sledger where \"index\" = $1", rollback.index)
		if err != nil {
			panic(err)
		}
		rows.Close()

		if err := tx.Commit(); err != nil {
			panic(err)
		}
	}
}

func (driver *SQLDriver) scan() *SQLLedgerRow {
	var entry SQLLedgerRow
	if err := driver.rows.Scan(&entry.index, &entry.forward, &entry.backward); err != nil {
		panic(err)
	}

	return &entry
}

func (driver *SQLDriver) Close() {
	if err := driver.db.Close(); err != nil {
		panic(err)
	}
}
