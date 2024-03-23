FROM golang:1.22.1-bookworm AS build
WORKDIR /build
COPY main.go ./
RUN go mod init getjs && \
    go get . && \
    CGO_ENABLED=0 GOOS=linux go build -o getjs

FROM alpine
WORKDIR /app
COPY --from=build /etc/ssl/certs/ca-certificates.crt \    
                  /etc/ssl/certs/ca-certificates.crt
COPY --from=build /build/getjs ./

ENTRYPOINT ["/app/getjs"]