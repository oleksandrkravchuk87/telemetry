FROM golang:1.24-alpine AS builder

RUN apk add --no-cache protobuf-dev make git
WORKDIR /app
COPY go.mod go.sum ./

RUN go mod download

COPY proto/ ./proto/
COPY *.go ./

RUN GOOS=linux go build  -o sensor_node .
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/sensor_node .

LABEL resource.memory="64m"
LABEL resource.cpu="0.5"

ENV RATE=1.0
ENV SENSOR_NAME=docker-sensor
ENV SINK_ADDR=localhost:9090

ENTRYPOINT ["./sensor_node"]
CMD ["-rate", "${RATE}", "-sensor-name", "${SENSOR_NAME}", "-sink-addr", "${SINK_ADDR}"]

