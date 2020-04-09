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

ARG GITHUB_API_TOKEN

ENV RULES_CONTENT_DIR=/rules-content \
    RULES_REPO=https://github.com/RedHatInsights/ccx-rules-ocp/ \
    GIT_ASKPASS=/git-askpass.sh

# clone rules content repository and build the aggregator
RUN umask 0022 && \
    mkdir -p $RULES_CONTENT_DIR && \
    echo "echo $GITHUB_API_TOKEN" > $GIT_ASKPASS && \
    chmod +x /git-askpass.sh && \
    git -C $RULES_CONTENT_DIR clone $RULES_REPO $RULES_CONTENT_DIR && \
    cd insights-results-aggregator && \
    make build

FROM registry.access.redhat.com/ubi8-minimal

COPY --from=builder /go/insights-results-aggregator/insights-results-aggregator .
COPY --from=builder /go/insights-results-aggregator/openapi.json /openapi/openapi.json
# copy just the rule content instead of the whole repository
COPY --from=builder /rules-content/content/ /rules-content
# copy tutorial/fake rule hit on all reports
COPY rules/tutorial/content/ /rules-content/external/rules

RUN chmod a+x /insights-results-aggregator

USER 1001

CMD ["/insights-results-aggregator"]
