#!/usr/bin/env bash
# Copyright 2020 Red Hat, Inc
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


DIR="$( cd "$( dirname "$0" )" && pwd )"
IMG_NAME="aggregator-postgres"

# export MOCK_DATA=true
# export PGDATA=/var/lib/postgresql/data/nonstandardpath/

# shellcheck disable=SC2143
[ "$(docker ps -a | grep $IMG_NAME)" ] && docker stop $IMG_NAME

docker build -f "$DIR"/Dockerfile.postgres -t $IMG_NAME .

docker run --name $IMG_NAME -p 5432:5432 --rm -d $IMG_NAME
# docker run --name $IMG_NAME -p 5432:5432 -e MOCK_DATA --rm -d $IMG_NAME
