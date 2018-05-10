FROM alpine:3.6

ADD tmp/_output/bin/envoy-operator /usr/local/bin/envoy-operator

RUN adduser -D envoy-operator
USER envoy-operator
