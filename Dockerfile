FROM golang:1.20

COPY . .

RUN go build -o dismention

CMD ["dismention"]
