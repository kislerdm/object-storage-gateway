FROM golang:1.21-alpine

WORKDIR /mnt/homework

COPY . .

RUN go mod tidy && \
    go build -o homework-object-storage .

FROM docker

COPY --from=0 /mnt/homework/homework-object-storage /usr/local/bin/homework-object-storage

RUN apk add bash curl
