package main

import "flag"

func main() {
	ledgerRoot := flag.String("ledger", "migrations", "path within git repository to sledger file")
	databaseURL := flag.String("database", "postgresql://localhost", "URL of database to update")
	flag.Parse()

	driver := Connect(*databaseURL)
	defer driver.Close()
	Migrate(driver, *ledgerRoot)
	driver.Rollback()
}
