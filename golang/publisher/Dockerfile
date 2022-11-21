# Compile stage
FROM golang:1.18.3 AS build-env
# add serts
RUN apt-get update && apt-get install -y \
    ca-certificates

ADD . /dockerdev
WORKDIR /dockerdev

RUN go build -v -o /publisher

# Final stage
FROM debian:buster

WORKDIR /
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /publisher /
COPY sitemap.xml /

CMD ["/publisher"]