FROM golang:1.13 AS builder

COPY . insights-results-aggregator

RUN cd insights-results-aggregator && \
    make build

FROM registry.access.redhat.com/ubi8-minimal

COPY --from=builder /go/insights-results-aggregator/insights-results-aggregator .

RUN chmod a+x /insights-results-aggregator

USER 1001

CMD ["/insights-results-aggregator"]
