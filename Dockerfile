FROM golang:1.21-alpine

WORKDIR /mnt/homework

COPY . .

RUN go mod tidy && \
    go build -o gateway .

FROM docker

COPY --from=0 /mnt/homework/gateway /usr/local/bin/gateway

EXPOSE 3000

ENV PORT 3000
ENV STORAGE_INSTANCES_PREFIX amazin-object-storage
ENV LOG_DEBUG true

ENTRYPOINT gateway
