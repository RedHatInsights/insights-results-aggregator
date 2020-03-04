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

FROM golang:1.13 AS builder

COPY . insights-results-aggregator

ARG SOURCE_REPOSITORY_URL

# get CA cert for SSL
RUN wget https://password.corp.redhat.com/RH-IT-Root-CA.crt -P /tmp
ENV GIT_SSL_CAINFO=/tmp/RH-IT-Root-CA.crt
ENV RULES_CONTENT_DIR=/rules_content

# clone rules content repository and build the aggregator
RUN umask 0022 && \
    mkdir -p $RULES_CONTENT_DIR && \
    git -C $RULES_CONTENT_DIR clone $SOURCE_REPOSITORY_URL $RULES_CONTENT_DIR && \
    cd insights-results-aggregator && \
    make build

FROM registry.access.redhat.com/ubi8-minimal

COPY --from=builder /go/insights-results-aggregator/insights-results-aggregator .
COPY --from=builder /go/insights-results-aggregator/openapi.json /openapi/openapi.json
# copy just the content of the rules, not the whole repository
COPY --from=builder /rules_content/content /rules_content

RUN chmod a+x /insights-results-aggregator

USER 1001

CMD ["/insights-results-aggregator"]
