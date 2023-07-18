package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gocql/gocql"
)

type CQLDriver struct {
	session *gocql.Session
	index   uint64
}

func ConnectCQL(databaseURL string) Driver {
	Step(StepConnect, databaseURL)

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		panic(err)
	}

	keyspace := strings.TrimPrefix(parsedURL.Path, "/")

	if keyspace == "" {
		Step(StepError, "database url missing required keyspace")
		os.Exit(1)
	}

	cluster := gocql.NewCluster(parsedURL.Host)

	if parsedURL.User != nil {
		password, _ := parsedURL.User.Password()
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: parsedURL.User.Username(),
			Password: password,
		}
	}

	sslOptionsList := parseCQLSSLOptions(parsedURL.Query())

	verifyKeyspace(newCQLSession(cluster, sslOptionsList), keyspace)
	cluster.Keyspace = keyspace

	driver := CQLDriver{session: newCQLSession(cluster, sslOptionsList), index: 0}
	driver.verifySledgerTable()

	return &driver
}

func newCQLSession(cluster *gocql.ClusterConfig, sslOptionsList []*gocql.SslOptions) *gocql.Session {
	var errs []string

	for _, sslOptions := range sslOptionsList {
		cluster.SslOpts = sslOptions

		if session, err := cluster.CreateSession(); err != nil {
			if sslOptions == nil {
				errs = append(errs, fmt.Sprintf("Failed to connect without TLS: %v", err))
			} else {
				errs = append(errs, fmt.Sprintf("Failed to connect with TLS: %v", err))
			}
		} else {
			return session
		}
	}

	for _, err := range errs {
		Step(StepError, "%v", err)
	}

	os.Exit(1)
	return nil // Needed by Go even though it should be unreachable.
}

// Based on PostgreSQL SSL settings.
func parseCQLSSLOptions(query url.Values) []*gocql.SslOptions {
	sslOptions := gocql.SslOptions{
		CertPath:               query.Get("sslcert"),
		KeyPath:                query.Get("sslkey"),
		CaPath:                 query.Get("sslrootcert"),
		EnableHostVerification: false,
	}

	switch sslMode := query.Get("sslmode"); sslMode {
	case "disable": // Do not enable TLS.
		return []*gocql.SslOptions{nil}
	case "allow": // First try non-TLS, then try TLS.
		return []*gocql.SslOptions{nil, &sslOptions}
	case "", "prefer": // First try TLS, then try non-TLS. Default.
		return []*gocql.SslOptions{&sslOptions, nil}
	case "require": // Only TLS. Equivalent to verify-ca if sslrootcert is set.
		return []*gocql.SslOptions{&sslOptions}
	case "verify-ca": // Only TLS. Verify the server cert is issued by a sslrootcert or, if not set, authority.
		Step(StepError, `"verify-ca" is not yet implemented -- use "require" or "verify-full"`)
		os.Exit(1)
		return nil // Needed by Go even though it should be unreachable.
	case "verify-full":
		sslOptions.EnableHostVerification = true
		return []*gocql.SslOptions{&sslOptions}
	default:
		Step(StepError, "Unrecognized sslmode: %q", sslMode)
		os.Exit(1)
		return nil // Needed by Go even though it should be unreachable.
	}
}

func escapeCQLID(id string) string {
	if strings.Contains(id, `"`) {
		Step(StepError, "invalid cql id: %v", id)
		os.Exit(1)
	}

	return `"` + id + `"`
}

func verifyKeyspace(session *gocql.Session, keyspace string) {
	// We default to simple replication with factor of one because it has
	// the best compatibility. Anything above and you cannot reach quorum on
	// a single node cluster. Developers should adjust keyspace settings
	// using ALTER KEYSPACE changes.
	Step(StepSetup, "Creating sledger keyspace if it does not exist...")
	if err := session.Query("CREATE KEYSPACE IF NOT EXISTS " + escapeCQLID(keyspace) + " WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}").Exec(); err != nil {
		panic(err)
	}
}

func (driver *CQLDriver) verifySledgerTable() {
	// TODO : consider index from text -> varint
	Step(StepSetup, "Creating sledger table if it does not exist...")
	if err := driver.session.Query(`CREATE TABLE IF NOT EXISTS "sledger" ("index" text PRIMARY KEY, "change" text)`).Exec(); err != nil {
		panic(err)
	}
}

func (driver *CQLDriver) Apply(change *Change) {
	var storedChangeRaw string
	if err := driver.session.Query(`SELECT "change" FROM "sledger" WHERE "index" = ?`, fmt.Sprint(driver.index)).Scan(&storedChangeRaw); err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			driver.doForward(change)
			driver.index++
			return
		}

		panic(err)
	}

	var storedChange Change
	if err := json.Unmarshal([]byte(storedChangeRaw), &storedChange); err != nil {
		panic(err)
	}

	forward, _ := change.ToCQL()
	if storedForward, _ := storedChange.ToCQL(); forward != storedForward {
		Step(StepError, "Database does not match migration.\n Database: %v\nMigration: %v", storedForward, forward)
		os.Exit(1)
	}

	Step(StepSkip, forward)
	driver.index++
}

func (driver *CQLDriver) doForward(change *Change) {
	forward, _ := change.ToCQL()
	Step(StepForward, forward)

	if err := driver.session.Query(forward).Exec(); err != nil {
		panic(err)
	}

	index := fmt.Sprint(driver.index)
	batch := driver.session.NewBatch(gocql.LoggedBatch)
	batch.Entries = []gocql.BatchEntry{
		{Stmt: `INSERT INTO "sledger" ("index", "change") VALUES (?, ?)`, Args: []any{index, change.ToJSON()}},
		{Stmt: `INSERT INTO "sledger" ("index", "change") VALUES (?, ?)`, Args: []any{"latest", index}},
	}

	if err := driver.session.ExecuteBatch(batch); err != nil {
		panic(err)
	}
}

func (driver *CQLDriver) Rollback() {
	var latestIndexRaw string
	if err := driver.session.Query(`SELECT "change" FROM "sledger" WHERE "index" = ?`, "latest").Scan(&latestIndexRaw); err != nil {
		panic(err)
	}

	latestIndex, err := strconv.ParseUint(latestIndexRaw, 10, 64)
	if err != nil {
		panic(err)
	}

	for index := latestIndex; index >= driver.index; index-- {
		var storedChangeRaw string
		if err := driver.session.Query(`SELECT "change" FROM "sledger" WHERE "index" = ?`, fmt.Sprint(index)).Scan(&storedChangeRaw); err != nil {
			panic(err)
		}

		var storedChange Change
		if err := json.Unmarshal([]byte(storedChangeRaw), &storedChange); err != nil {
			panic(err)
		}

		_, backward := storedChange.ToCQL()
		Step(StepRollback, "%v", backward)

		if err := driver.session.Query(backward).Exec(); err != nil {
			panic(err)
		}

		batch := driver.session.NewBatch(gocql.LoggedBatch)
		batch.Entries = []gocql.BatchEntry{
			{Stmt: `DELETE FROM "sledger" WHERE "index" = ?`, Args: []any{fmt.Sprint(index)}},
			{Stmt: `INSERT INTO "sledger" ("index", "change") VALUES (?, ?)`, Args: []any{"latest", fmt.Sprint(index - 1)}},
		}

		if err := driver.session.ExecuteBatch(batch); err != nil {
			panic(err)
		}
	}
}

func (driver *CQLDriver) Close() {
	// gocql drops errors returned on connection close so there's nothing we
	// can do here but follow along.
	driver.session.Close()
}
