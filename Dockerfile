FROM golang:1.19

WORKDIR /go/go-app
COPY ./ ./
RUN go build -o main .
CMD ["./main"]
