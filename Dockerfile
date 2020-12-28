ARG work_dir=/go/src/github.com/Bnei-Baruch/gxydb-api
ARG build_number=dev
ARG db_url="postgres://user:password@host.docker.internal/galaxy?sslmode=disable"
ARG test_gateway_url="ws://host.docker.internal:8188/"
ARG test_gateway_admin_url="http://host.docker.internal:7088/admin"
ARG test_mqtt_broker="host.docker.internal:1883"

FROM golang:1.14-alpine3.11 as build

LABEL maintainer="edoshor@gmail.com"

ARG work_dir
ARG build_number
ARG db_url
ARG test_gateway_url
ARG test_gateway_admin_url
ARG test_mqtt_broker

ENV GOOS=linux \
	CGO_ENABLED=0 \
	DB_URL=${db_url} \
	TEST_GATEWAY_URL=${test_gateway_url} \
	TEST_GATEWAY_ADMIN_URL=${test_gateway_admin_url} \
	SECRET=12345678901234567890123456789012 \
	MONITOR_GATEWAY_TOKENS=false \
	COLLECT_PERIODIC_STATS=false \
	GATEWAY_ROOMS_SECRET=adminpwd \
	MQTT_BROKER_URL=${test_mqtt_broker}

RUN apk update && \
    apk add --no-cache \
    git

WORKDIR ${work_dir}
COPY . .

RUN go test -v $(go list ./... | grep -v /models) \
    && go build -ldflags "-w -X github.com/Bnei-Baruch/gxydb-api/version.PreRelease=${build_number}"


FROM alpine:3.11
ARG work_dir
WORKDIR /app
COPY ./misc/wait-for /wait-for
COPY --from=build ${work_dir}/gxydb-api .

EXPOSE 8080
CMD ["./gxydb-api", "server"]
