# Compile stage
FROM golang:1.18.3 AS build-env
# add serts
RUN apt-get update && apt-get install -y \
    ca-certificates
# Build Delve
RUN go install github.com/go-delve/delve/cmd/dlv@latest

ADD . /dockerdev
WORKDIR /dockerdev

# Compile the application with the optimizations turned off
# This is important for the debugger to correctly work with the binary
RUN go build -gcflags "all=-N -l" -o /publisher

# Final stage
FROM debian:buster

EXPOSE 44444

WORKDIR /
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /go/bin/dlv /
COPY --from=build-env /publisher /
COPY sitemap.xml /

CMD ["/dlv", "--listen=:44444", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "/publisher"]