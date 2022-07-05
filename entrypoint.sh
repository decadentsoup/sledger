#!/bin/bash

export PGPASSWORD=$DATABASE_MIGRATION_PASSWORD

function wait_for_db() {
   # before doing anything, wait for our sidecar to be up
   time_spent=0
   while [ "$time_spent" -lt "$DATABASE_PING_TIMEOUT" ] && ! pg_isready -h $DATABASE_HOST -p $DATABASE_PORT
   do
      time_spent=$((time_spent+1))
      >&2 echo "$(date) - waiting for database to start"
      sleep 1
   done   
}

function run_sledger() {
   # run sledger and capture exit code to exit with later
   /sledger --database "postgres://$DATABASE_MIGRATION_USERNAME:$DATABASE_MIGRATION_PASSWORD@$DATABASE_HOST:$DATABASE_PORT/$DATABASE_NAME?sslmode=disable"
   result=$?
   
   # after sledger runs, kill the proxy
   if [ $(pgrep cloud_sql_proxy) ] && [ ${result} -eq 0 ]
   then
      pkill cloud_sql_proxy
   fi

   exit $result
}

if [[ $# -gt 0 ]]; then
   exec $@
else
   wait_for_db
   run_sledger
fi

