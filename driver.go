package main

type Driver interface {
	// Apply the given change to the server. The driver is expected to store
	// the forward/backward information in the database ledger.
	Apply(forward string, backward string)
	// Run any rollbacks that are necessary based on what commands exist in
	// the database ledger but were not passed in with Apply().
	Rollback()
	// Called to close the connection to the database and clean up.
	Close()
}
