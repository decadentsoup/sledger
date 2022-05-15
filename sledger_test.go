package main

import (
	"testing"
)

var (
	MockOSGetenvCounter    = 0
	MockOSGetenvResult     []string
	MockOSGetenvParameters []map[string]string
)

// Test the case where there are no variable substitutions in the string
func TestReplaceVariablesInStringNoReplacements(t *testing.T) {
	sqlStatement := "REVOKE ALL ON SCHEMA public FROM PUBLIC"
	result := ReplaceVariablesInString(sqlStatement)
	if sqlStatement != result {
		t.Fatalf(`ReplaceVariablesInString("%v") = %v, want match for %v`, sqlStatement, result, sqlStatement)
	}
}

// Test the case where there is one variable substitution in the string
func TestReplaceVariablesInStringOneReplacement(t *testing.T) {
	setupMocks()

	sqlStatement := "CREATE ROLE some_user WITH LOGIN ENCRYPTED PASSWORD '${PASSWORD}'"
	want := "CREATE ROLE some_user WITH LOGIN ENCRYPTED PASSWORD 'abc123'"

	OSGetenv = mockOSGetenv
	MockOSGetenvResult = []string{"abc123"}

	result := ReplaceVariablesInString(sqlStatement)
	if want != result {
		t.Fatalf(`ReplaceVariablesInString("%v") = %v, want match for %v`, sqlStatement, result, sqlStatement)
	}
	if MockOSGetenvParameters[0]["key"] != "PASSWORD" {
		t.Fatalf(`Expected parameter to OSGetEnv: %v, instead received %v`, "PASSWORD", MockOSGetenvParameters[0]["key"])
	}
	if MockOSGetenvCounter != 1 {
		t.Fatalf(`Expected OSGetEnv function to be called 1 time`)
	}
}

// Test the case where there are multiple variable substitutions in the string
func TestReplaceVariablesInStringTwoReplacements(t *testing.T) {
	setupMocks()

	sqlStatement := "CREATE ROLE ${USER} WITH LOGIN ENCRYPTED PASSWORD '${PASSWORD}'"
	want := "CREATE ROLE some_user WITH LOGIN ENCRYPTED PASSWORD 'abc123'"

	MockOSGetenvResult = []string{"some_user", "abc123"}

	result := ReplaceVariablesInString(sqlStatement)
	if want != result {
		t.Fatalf(`ReplaceVariablesInString("%v") = %v, want match for %v`, sqlStatement, result, sqlStatement)
	}
	if MockOSGetenvParameters[1]["key"] != "PASSWORD" {
		t.Fatalf(`Expected parameter to OSGetEnv: %v, instead received %v`, "PASSWORD", MockOSGetenvParameters[0]["key"])
	}
	if MockOSGetenvParameters[0]["key"] != "USER" {
		t.Fatalf(`Expected parameter to OSGetEnv: %v, instead received %v`, "USER", MockOSGetenvParameters[0]["key"])
	}
	if MockOSGetenvCounter != 2 {
		t.Fatalf(`Expected OSGetEnv function to be called 2 times`)
	}
}

// Sets up and/or resets mocks
func setupMocks() {
	OSGetenv = mockOSGetenv
	MockOSGetenvCounter = 0
	MockOSGetenvResult = []string{}
	MockOSGetenvParameters = []map[string]string{}
}

// Mock function for os.Getenv
func mockOSGetenv(key string) string {
	MockOSGetenvCounter += 1

	MockOSGetenvParameters = append(MockOSGetenvParameters, map[string]string{"key": key})

	return MockOSGetenvResult[MockOSGetenvCounter-1]
}
