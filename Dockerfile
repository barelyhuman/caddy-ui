
FROM golang:1.23.0 AS builder 
WORKDIR /app
COPY . . 
RUN go mod download && go mod verify 
RUN go build -v -o dcc-app .


FROM ubuntu:latest 
WORKDIR /app
COPY --from=builder /app/dcc-app /app/dcc-app

EXPOSE 8081

CMD ["/app/dcc-app"]