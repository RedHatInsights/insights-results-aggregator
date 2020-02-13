#!/usr/bin/env bash

DIR="$( cd "$( dirname "$0" )" && pwd )"
IMG_NAME="aggregator-postgres"

# export MOCK_DATA=true
# export PGDATA=/var/lib/postgresql/data/nonstandardpath/

[ "$(docker ps -a | grep $IMG_NAME)" ] && docker stop $IMG_NAME

docker build -f $DIR/Dockerfile.postgres -t $IMG_NAME .

docker run --name $IMG_NAME -p 5432:5432 --rm -d $IMG_NAME
# docker run --name $IMG_NAME -p 5432:5432 -e MOCK_DATA --rm -d $IMG_NAME
