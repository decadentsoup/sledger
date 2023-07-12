package main

import "fmt"

const (
	StepConnect    = "CONNECT"
	StepSetup      = "SETUP"
	StepLedger     = "LEDGER"
	StepRead       = "READ"
	StepError      = "ERROR"
	StepSkip       = "SKIP"
	StepForward    = "FORWARD"
	StepRollback   = "ROLLBACK"
	StepComplete   = "COMPLETE"
	StepDisconnect = "DISCONNECT"
	MaxStepLength  = 8
)

func Step(category string, format string, args ...any) {
	fmt.Printf("\033[1m%*v\033[0m %v\n", MaxStepLength, category, fmt.Sprintf(format, args...))
}
