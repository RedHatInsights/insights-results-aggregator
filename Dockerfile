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

FROM registry.redhat.io/rhel8/go-toolset:1.16 AS builder

COPY . .

USER 0

# build the aggregator
RUN umask 0022 && \
    make build && \
    chmod a+x insights-results-aggregator

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

COPY --from=builder /opt/app-root/src/insights-results-aggregator .
COPY --from=builder /opt/app-root/src/openapi.json /openapi/openapi.json

USER 1001

CMD ["/insights-results-aggregator"]
