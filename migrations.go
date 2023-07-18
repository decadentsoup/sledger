package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Migration struct {
	ID      string   `json:"id"`
	Changes []Change `json:"changes"`
}

type Change struct {
	CQL *RawChange `json:"cql,omitempty"`
	SQL *RawChange `json:"sql,omitempty"`
}

type RawChange struct {
	Forward  string `json:"forward"`
	Backward string `json:"backward,omitempty"`
}

func Migrate(driver Driver, root string) {
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
			migration.Run(driver, name)
		}
	}
}

func (migration *Migration) Run(driver Driver, name string) {
	migration.verifyID(name)

	for _, change := range migration.Changes {
		change.Run(driver)
	}
}

func (migration *Migration) verifyID(name string) {
	baseName := name[:len(name)-len(filepath.Ext(name))]

	if migration.ID != baseName {
		Step(StepError, "%v has mismatching ID %v", baseName, migration.ID)
		os.Exit(1)
	}
}

func (change *Change) Run(driver Driver) {
	driver.Apply(change)
}

func (change *Change) ToJSON() string {
	result, err := json.Marshal(change)
	if err != nil {
		panic(err)
	}

	return string(result)
}

func (change *Change) ToCQL() (string, string) {
	if change.CQL != nil {
		return change.CQL.Forward, change.CQL.Backward
	}

	Step(StepError, "%v not supported for cql", change)
	os.Exit(1)
	return "", "" // Needed by Go even though it should be unreachable.
}

func (change *Change) ToSQL() (string, string) {
	if change.SQL != nil {
		return change.SQL.Forward, change.SQL.Backward
	}

	Step(StepError, "%v not supported for sql", change)
	os.Exit(1)
	return "", "" // Needed by Go even though it should be unreachable.
}
