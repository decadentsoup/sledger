package main

import "flag"

func main() {
	ledgerRoot := flag.String("ledger", "migrations", "path within git repository to sledger file")
	databaseURL := flag.String("database", "postgresql://localhost", "URL of database to update")
	flag.Parse()

	db := Connect(*databaseURL)
	defer db.Close()

	db.Setup()

	RunMigrations(db, *ledgerRoot)

	db.RunRollbacks()
}
