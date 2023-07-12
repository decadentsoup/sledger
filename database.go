package main

import (
	"database/sql"
	"os"

	_ "github.com/lib/pq"
)

type Database struct {
	conn      *sql.DB
	rows      *sql.Rows
	index     int
	rollbacks []*DatabaseSledgerEntry
}

type DatabaseSledgerEntry struct {
	index    int
	forward  string
	backward string
}

func Connect(databaseURL string) *Database {
	Step(StepConnect, databaseURL)

	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		panic(err)
	}

	return &Database{conn: conn, rows: nil, index: 0, rollbacks: []*DatabaseSledgerEntry{}}
}

func (db *Database) Close() {
	if err := db.conn.Close(); err != nil {
		panic(err)
	}
}

func (db *Database) next() bool {
	if db.rows == nil {
		rows, err := db.conn.Query(`select "index", "forward", "backward" from "sledger" order by "index"`)
		if err != nil {
			panic(err)
		}

		db.rows = rows
	}

	return db.rows.Next()
}

func (db *Database) scan() *DatabaseSledgerEntry {
	var entry DatabaseSledgerEntry
	if err := db.rows.Scan(&entry.index, &entry.forward, &entry.backward); err != nil {
		panic(err)
	}

	return &entry
}

func (db *Database) Change(expectedForward string, expectedBackward string) {
	if !db.next() {
		db.doForward(expectedForward, expectedBackward)
		db.index++
		return
	}

	actualChange := db.scan()

	if actualChange.index != db.index {
		Step(StepError, "Expected index %v, got index %v.", db.index, actualChange.index)
		os.Exit(1)
	}

	if actualChange.forward != expectedForward {
		Step(StepError, "Database does not match migration.\n Database: %v\nMigration: %v", actualChange.forward, expectedForward)
		os.Exit(1)
	}

	Step(StepSkip, expectedForward)
	db.index++
}

func (db *Database) doForward(forward string, backward string) {
	Step(StepForward, forward)

	tx, err := db.conn.Begin()
	if err != nil {
		panic(err)
	}
	defer tx.Rollback()

	rows, err := db.conn.Query(forward)
	if err != nil {
		panic(err)
	}
	rows.Close()

	rows, err = db.conn.Query(`insert into "sledger" ("index", "forward", "backward") values ($1, $2, $3)`, db.index, forward, backward)

	if err != nil {
		panic(err)
	}
	rows.Close()

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}

func (db *Database) RunRollbacks() {
	for db.next() {
		db.rollbacks = append([]*DatabaseSledgerEntry{db.scan()}, db.rollbacks...)
	}

	for _, rollback := range db.rollbacks {
		if rollback.backward == "" {
			Step(StepError, "Missing rollback command, cannot rollback.")
			os.Exit(1)
		}

		Step(StepRollback, rollback.backward)

		tx, err := db.conn.Begin()
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
