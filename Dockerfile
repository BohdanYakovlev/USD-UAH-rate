FROM golang:alpine AS builder

WORKDIR /build

ADD go.mod .
ADD go.sum .

COPY . .

RUN go mod tidy

RUN go build -o api-service api-service/main.go

FROM alpine

WORKDIR /build

COPY --from=builder /build/api-service /build/api-service

COPY db /build/api-service/db

CMD ["/build/api-service/main"]