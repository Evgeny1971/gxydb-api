ARG work_dir=/go/src/github.com/Bnei-Baruch/gxydb-api

FROM golang:1.14-alpine3.11 as build

LABEL maintainer="edoshor@gmail.com"

ARG work_dir

ENV GOOS=linux \
	CGO_ENABLED=0

RUN apk update && \
    apk add --no-cache \
    git

WORKDIR ${work_dir}
COPY . .
RUN go build


FROM alpine:3.11
ARG work_dir
WORKDIR /app
#COPY ./misc/wait-for /wait-for
COPY --from=build ${work_dir}/gxydb-api .

ENV DB_URL="postgres://user:password@db/galaxy?sslmode=disable"
ENV ACC_URL="https://accounts.kbb1.com/auth/realms/main"

EXPOSE 8080
CMD ["./gxydb-api"]
