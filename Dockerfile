FROM golang:1.20

COPY . .

RUN go build -o /app

CMD ["/app"]
