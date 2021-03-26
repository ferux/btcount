FROM golang:1.16.2-alpine3.13 as builder

RUN apk update && apk add --no-cache make git
COPY . /app
WORKDIR /app

RUN go mod tidy
RUN make release

FROM alpine:3.13

EXPOSE 8080
COPY --from=builder /app/bin/btcount /app/btcount

ENTRYPOINT ["/app/btcount"]
