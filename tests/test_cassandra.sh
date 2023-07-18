#!/bin/sh -e
. "$(dirname "$0")/common.sh"

clean() {
        docker rm --force sledger-test-cassandra
        docker volume rm --force sledger-test-cassandra
        docker network rm --force sledger-test-cassandra
}

step "Clean up previous, failed runs if any."
clean

step "Set up a network for the containers to communicate."
docker network create sledger-test-cassandra

step "Start up Cassandra for testing."
docker run \
        --rm \
        --name sledger-test-cassandra \
        --network sledger-test-cassandra \
        --detach \
        --mount type=volume,source=sledger-test-cassandra,destination=/var/lib/cassandra \
        cassandra:4.1

step "Wait until Cassandra is ready."
until docker exec sledger-test-cassandra cqlsh -e "DESCRIBE CLUSTER"; do true; done

step "Run sledger using our example ledger."
docker run \
	--rm \
	--network sledger-test-cassandra \
	--mount type=bind,source="$PWD/examples/cql",destination=/migrations,readonly \
	sledger:test \
	--database "cassandra://sledger-test-cassandra/test"

step "Verify the database dump matches expectations."
docker exec sledger-test-cassandra cqlsh -e "DESCRIBE SCHEMA; SELECT * FROM test.sledger; SELECT * FROM test.account; SELECT * FROM test.post" |
        diff examples/cql/dump_cassandra.cql -

step "Clean up after ourselves."
clean
