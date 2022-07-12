package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"gopkg.in/yaml.v2"
)

type (
	sledger struct {
		Sledger []struct {
			Forward  string
			Backward string
		}
	}

	rollback struct {
		index      int
		dbBackward string
	}
)

// This is done for testing purposes
var (
	OSGetenv = os.Getenv
)

const (
	SLEDGER_VERSION = "a45a9821-8e0d-4126-8d99-0543e7f1f8f7"
)

func main() {
	path := flag.String("ledger", "sledger.yaml", "path within git repository to sledger file")
	var database string = *flag.String("database", "postgresql://localhost", "URL of database to update")
	flag.Parse()

	fmt.Println("==> Sledger")

	fmt.Println("==> Parameters")
	fmt.Println("  Ledger:", *path)
	if isFlagPassed("database") {
		fmt.Println("  Database: Data from flag used.")
	} else {
		database = getDbFromEnv()
	}

	fmt.Println("==> Setup")

	fmt.Println("Connecting to database...")

	db, err := sql.Open("postgres", database)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	verifySledgerVersion(db)
	verifySledgerTable(db)
	sledger := loadSledgerYaml(*path)

	fmt.Println("==> Synchorization")
	sync(db, sledger)

	fmt.Println("==> Complete")
}

func verifySledgerVersion(db *sql.DB) {
	createSledgerVersion(db)

	version := getSledgerVersion(db)

	if version == "" {
		insertSledgerVersion(db)
		version = SLEDGER_VERSION
	}

	if version != SLEDGER_VERSION {
		panic("Unsupported sledger version. Please use the correct version of sledger.")
	}
}

func createSledgerVersion(db *sql.DB) {
	fmt.Println("Ensuring sledger_version exists...")
	rows, err := db.Query("create table if not exists sledger_version (sledger_version text)")
	if err != nil {
		panic(err)
	}
	rows.Close()
}

func getSledgerVersion(db *sql.DB) string {
	fmt.Println("Getting sledger_version...")
	rows, err := db.Query("select sledger_version from sledger_version limit 1")
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

func insertSledgerVersion(db *sql.DB) {
	fmt.Println("Inserting sledger_version...")
	rows, err := db.Query("insert into sledger_version values ($1)", SLEDGER_VERSION)
	if err != nil {
		panic(err)
	}
	rows.Close()
}

func verifySledgerTable(db *sql.DB) {
	fmt.Println("Creating sledger table if it does not exist...")
	rows, err := db.Query("create table if not exists sledger (index bigint not null, forward text not null, backward text not null, timestamp timestamp not null default now())")
	if err != nil {
		panic(err)
	}
	rows.Close()
}

func loadSledgerYaml(path string) sledger {
	fmt.Println("Loading sledger...")

	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	sledger := sledger{}

	if err := yaml.Unmarshal(data, &sledger); err != nil {
		panic(err)
	}

	replaceVariables(&sledger)

	return sledger
}

// Replaces all variables in Forward and Backward sql statements of sledger.
func replaceVariables(sledger *sledger) {
	fmt.Println("\t...replacing env vars")

	// iterate over all the elements in the ledger
	for idx, _ := range sledger.Sledger {
		sledger.Sledger[idx].Forward = ReplaceVariablesInString(sledger.Sledger[idx].Forward)
		sledger.Sledger[idx].Backward = ReplaceVariablesInString(sledger.Sledger[idx].Backward)
	}
}

// This function takes a string and replaces all ${VARIABLES} with their corresponding environment variable value.
// This function also does minimal checking (eg: if there is a '${' there must be a closing '}').  If there is no
// environment variable set for a defined ${VARIABLE}, then a panic occurs.
func ReplaceVariablesInString(in string) string {
	index := strings.IndexAny(in, "${")
	for index != -1 {
		closeIndex := strings.IndexAny(in, "}")
		if closeIndex == -1 {
			panic("Found '${' without a terminating '}' in string: " + in)
		}
		variableName := in[index+2 : closeIndex]
		envVarValue := OSGetenv(variableName)
		if envVarValue == "" {
			panic(fmt.Sprintf("Environment variable [%v] is not set!", variableName))
		}

		// calculate the return value
		in = in[:index] + envVarValue + in[closeIndex+1:]

		// while looper iteration to catch more than 1 variable in the string
		index = strings.IndexAny(in, "${")
	}

	return in
}

func sync(db *sql.DB, sledger sledger) {
	rows, err := db.Query("select forward, backward from sledger order by index")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	index := 0

	var rollbacks []rollback

	for rows.Next() {
		var dbForward, dbBackward string

		if err := rows.Scan(&dbForward, &dbBackward); err != nil {
			panic(err)
		}

		if index >= len(sledger.Sledger) {
			rollbacks = append(rollbacks, rollback{index, dbBackward})
		} else {
			yamlForward := sledger.Sledger[index].Forward

			if dbForward != yamlForward {
				fmt.Printf("ERROR     Database does not match YAML\nDatabase: %s\n    YAML: %s\n", dbForward, yamlForward)
				os.Exit(1)
			}

			fmt.Printf("SKIP      %s\n", AbbreviateSqlCommand(yamlForward))
		}

		index++
	}

	for index < len(sledger.Sledger) {
		doForward(db, index, sledger.Sledger[index].Forward, sledger.Sledger[index].Backward)

		index++
	}

	for index = len(rollbacks) - 1; index >= 0; index-- {
		doRollback(db, rollbacks[index].index, rollbacks[index].dbBackward)
	}
}

// This function abbreviates an sql command down to it's first instruction (which is usually meaningful).  This is
// primarily done to prevent leaking credentials to stdout.
func AbbreviateSqlCommand(cmd string) string {
	idx := strings.IndexAny(cmd, " ")
	if idx > 0 {
		return cmd[:idx]
	} else {
		return cmd
	}
}

func doRollback(db *sql.DB, index int, dbBackward string) {
	if dbBackward == "" {
		fmt.Println("ERROR     Missing rollback command, cannot rollback.")
		os.Exit(1)
	} else {
		fmt.Printf("ROLLBACK  %s\n", AbbreviateSqlCommand(dbBackward))

		tx, err := db.Begin()
		if err != nil {
			panic(err)
		}
		defer tx.Rollback()

		rows, err := tx.Query(dbBackward)
		if err != nil {
			panic(err)
		}
		rows.Close()

		rows, err = tx.Query("delete from sledger where index = $1", index)
		if err != nil {
			panic(err)
		}
		rows.Close()

		if err := tx.Commit(); err != nil {
			panic(err)
		}
	}
}

func doForward(db *sql.DB, index int, yamlForward string, yamlBackward string) {
	fmt.Printf("FORWARD   %s\n", AbbreviateSqlCommand(yamlForward))

	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	defer tx.Rollback()

	rows, err := db.Query(yamlForward)
	if err != nil {
		panic(err)
	}
	rows.Close()

	rows, err = db.Query("insert into sledger (index, forward, backward) values ($1, $2, $3)", index, yamlForward, yamlBackward)
	if err != nil {
		panic(err)
	}
	rows.Close()

	if err := tx.Commit(); err != nil {
		panic(err)
	}
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func getDbFromEnv() string {
	host := OSGetenv("DATABASE_HOST")
	if len(host) < 1 {
		fmt.Println("Host not specified. Default to localhost.")
		host = "localhost"
	}

	port := OSGetenv("DATABASE_PORT")
	if len(port) < 1 {
		fmt.Println("Port not specified. Default to 5432.")
		port = "5432"
	}

	dbName := OSGetenv("DATABASE_NAME")

	dbUser := OSGetenv("DATABASE_MIGRATION_USERNAME")
	if len(dbUser) < 1 {
		fmt.Println("User not specified. Please provide one.")
	}

	dbUserPass := OSGetenv("DATABASE_MIGRATION_PASSWORD")

	fmt.Println("  Database:")
	fmt.Println("  - Host: ", host)
	fmt.Println("  - Port: ", port)
	fmt.Println("  - Name: ", dbName)
	fmt.Println("  - User: ", dbUser)

	dbEnv := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbUserPass, host, port, dbName)
	return dbEnv
}
