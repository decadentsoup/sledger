package main

import (
	"os"
	"strings"
)

type Driver interface {
	// Apply the given change to the server. The driver is expected to store
	// the forward/backward information in the database ledger.
	Apply(change *Change)
	// Run any rollbacks that are necessary based on what commands exist in
	// the database ledger but were not passed in with Apply().
	Rollback()
	// Called to close the connection to the database and clean up.
	Close()
}

func Connect(databaseURL string) Driver {
	protocol, _, _ := strings.Cut(databaseURL, ":")

	switch strings.ToLower(protocol) {
	case "postgresql":
		return ConnectSQL(databaseURL)
	case "cassandra":
		return ConnectCQL(databaseURL)
	default:
		Step(StepError, "Unknown database protocol: %q.", protocol)
		os.Exit(1)
		return nil // Needed by Go even though it should be unreachable.
	}
}
