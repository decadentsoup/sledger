package main

import (
	"os"
	"path/filepath"
	"strings"
)

type Migration struct {
	ID      string
	Changes []Change
}

type Change struct {
	SQL *SQLChange
}

type SQLChange struct {
	Forward  string
	Backward string
}

func RunMigrations(db *Database, root string) {
	Step(StepLedger, root)
	entries, err := os.ReadDir(root)
	if err != nil {
		panic(err)
	}

	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(root, name)

		switch strings.ToLower(filepath.Ext(name)) {
		case ".yaml", ".yml":
			var migration Migration
			Step(StepRead, name)
			ReadYAML(path, &migration)
			migration.Run(db, name)
		}
	}
}

func (migration *Migration) Run(db *Database, name string) {
	migration.verifyID(name)

	for _, change := range migration.Changes {
		change.Run(db)
	}
}

func (migration *Migration) verifyID(name string) {
	baseName := name[:len(name)-len(filepath.Ext(name))]

	if migration.ID != baseName {
		Step(StepError, "%v has mismatching ID %v", baseName, migration.ID)
		os.Exit(1)
	}
}

func (change *Change) Run(db *Database) {
	db.Change(change.SQL.Forward, change.SQL.Backward)
}
