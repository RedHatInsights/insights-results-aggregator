#!/usr/bin/env bash

# TODO: rewrite with docker-compose?

docker network create kafka-net

docker kill zookeeper
docker run --name zookeeper --rm -d --network kafka-net zookeeper:3.4
if [ $? -ne 0 ]; then
	echo unable to start zookeeper
	exit 1
fi

docker kill kafka
docker run -p 9093:9092 --rm -d --name kafka --network kafka-net --env ZOOKEEPER_IP=zookeeper ches/kafka
if [ $? -ne 0 ]; then
	echo unable to start kafka
	exit 1
fi
