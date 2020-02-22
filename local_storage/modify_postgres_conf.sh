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


# md5 for user `all` is enabled by default if POSTGRES_PASSWORD env var is present.
echo "host all all 0.0.0.0/0 trust" >> "${PGDATA}/pg_hba.conf"
echo "local all all trust" >> "${PGDATA}/pg_hba.conf"

echo "listen_addresses='*'" >> "${PGDATA}/postgresql.conf"

pg_ctl restart
