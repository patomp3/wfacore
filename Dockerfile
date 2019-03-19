FROM golang:1.11 as builder
#ENV HTTP_PROXY=http://172.19.37.21:80
#ENV HTTPS_PROXY=http://172.19.37.21:80

WORKDIR /go/src/github.com/patomp3/wfacore
RUN go get -d -v github.com/gorilla/mux
RUN go get -d -v github.com/spf13/viper
RUN go get -d -v gopkg.in/goracle.v2
RUN go get -d -v github.com/patomp3/smslogs
RUN go get -d -v github.com/patomp3/smsservices
RUN go get -d -v github.com/streadway/amqp

COPY .  .
#COPY reconnect.go .
#RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o icc-reconnect .
RUN go build -a -o wfacore .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/patomp3/wfacore .
CMD ["./wfacore"]
#CMD ["./icc-reconnect", "production"]