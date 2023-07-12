package main

import "os"

const SLEDGER_VERSION = "a45a9821-8e0d-4126-8d99-0543e7f1f8f7"

func (db *Database) Setup() {
	db.verifySledgerVersion()
	db.verifySledgerTable()
}

func (db *Database) verifySledgerVersion() {
	db.createSledgerVersion()

	version := db.getSledgerVersion()

	if version == "" {
		db.insertSledgerVersion()
		version = SLEDGER_VERSION
	}

	if version != SLEDGER_VERSION {
		Step(StepError, "Unsupported sledger version. Please use the correct version of sledger.")
		os.Exit(1)
	}
}

func (db *Database) createSledgerVersion() {
	Step(StepSetup, "Ensuring sledger_version table exists...")
	rows, err := db.conn.Query(`create table if not exists "sledger_version" ("sledger_version" text)`)
	if err != nil {
		panic(err)
	}
	rows.Close()
}

func (db *Database) getSledgerVersion() string {
	Step(StepSetup, "Getting sledger_version...")
	rows, err := db.conn.Query(`select "sledger_version" from "sledger_version" limit 1`)
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

func (db *Database) insertSledgerVersion() {
	Step(StepSetup, "Setting sledger_version...")
	rows, err := db.conn.Query("insert into sledger_version values ($1)", SLEDGER_VERSION)
	if err != nil {
		panic(err)
	}
	rows.Close()
}

func (db *Database) verifySledgerTable() {
	Step(StepSetup, "Creating sledger table if it does not exist...")
	rows, err := db.conn.Query(`create table if not exists "sledger" ("index" bigint not null, "forward" text not null, "backward" text not null, "timestamp" timestamp not null default now())`)
	if err != nil {
		panic(err)
	}
	rows.Close()
}
