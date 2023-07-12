#!/bin/sh -e

step() {
	printf "\33[1m==> \33[35m$1\33[0m\n"
}

clean() {
	docker rm --force sledger-test-pg
	docker volume rm --force sledger-test-pg
	docker network rm --force sledger-test
}

step "Change directory to the root."
cd "$(dirname "$0")/.."

step "Build the program to a special test tag."
docker build --tag sledger:test .

step "Clean up previous, failed runs if any."
clean

step "Set up a network for the containers to communicate."
docker network create sledger-test

step "Generate a password to use for the PostgreSQL database."
password="$(pwgen --secure 64)"
database="postgresql://postgres:$password@sledger-test-pg?sslmode=disable"

step "Start up PostgreSQL for testing."
docker run \
	--rm \
	--name sledger-test-pg \
	--network sledger-test \
	--detach \
	--env POSTGRES_PASSWORD="$password" \
	--mount type=volume,source=sledger-test-pg,destination=/var/lib/postgresql/data \
	postgres:15-alpine

step "Wait until PostgreSQL is ready."
until docker exec sledger-test-pg pg_isready; do true; done

step "Run sledger using our example ledger."
docker run \
	--rm \
	--network sledger-test \
	--mount type=bind,source="$PWD/example",destination=/migrations,readonly \
	sledger:test \
	--database "$database"

step "Verify the database dump matches expectations."
docker exec sledger-test-pg pg_dump "$database" |
	sed 's/[0-9][0-9]*-[0-9][0-9]-[0-9][0-9] [0-9][0-9]:[0-9][0-9]:[0-9][0-9]\.[0-9][0-9]*/(timestamp)/g' |
	diff example/pgdump.sql -

step "Clean up after ourselves."
clean
