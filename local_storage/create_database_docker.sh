#!/usr/bin/env bash

SUPERUSER=postgres
SU_PASSWORD=postgres

DATABASE=aggregator

USER=tester
USER_PASSWORD=tester


SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )/sql/"

# create user and database
psql "postgresql://${SUPERUSER}:${SU_PASSWORD}@${DB_SERVER}" -c "CREATE DATABASE ${DATABASE};"
psql "postgresql://${SUPERUSER}:${SU_PASSWORD}@${DB_SERVER}" -c "CREATE USER ${USER} PASSWORD '${USER_PASSWORD}';"

#create schema
cat "${SCRIPT_DIR}/schema_postgres.sql" | psql  "postgresql://${SUPERUSER}:${SU_PASSWORD}@${DB_SERVER}/${DATABASE}"

# grant priviliges to user
psql "postgresql://${SUPERUSER}:${SU_PASSWORD}@${DB_SERVER}/${DATABASE}" -c "GRANT  SELECT, INSERT, UPDATE,  DELETE
    ON  ALL TABLES IN SCHEMA public
    TO  ${USER};
	GRANT  SELECT,  UPDATE
    ON  ALL SEQUENCES IN SCHEMA public
    TO  ${USER};
    GRANT USAGE ON SCHEMA public TO ${USER};"

