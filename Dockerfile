ARG work_dir=/go/src/github.com/Bnei-Baruch/gxydb-api

FROM golang:1.14-alpine3.11 as build

LABEL maintainer="edoshor@gmail.com"

ARG work_dir
ARG db_url="postgres://user:password@host.docker.internal/galaxy?sslmode=disable"

ENV GOOS=linux \
	CGO_ENABLED=0 \
	DB_URL=${db_url}

RUN apk update && \
    apk add --no-cache \
    git

WORKDIR ${work_dir}
COPY . .

RUN go test $(go list ./... | grep -v /models) \
    && go build


FROM alpine:3.11
ARG work_dir
WORKDIR /app
COPY ./misc/wait-for /wait-for
COPY --from=build ${work_dir}/gxydb-api .

EXPOSE 8080
CMD ["./gxydb-api"]
