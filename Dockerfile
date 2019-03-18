FROM golang:1.11 as builder
#ENV HTTP_PROXY=http://172.19.37.21:80
#ENV HTTPS_PROXY=http://172.19.37.21:80
RUN go get -d -v gopkg.in/goracle.v2
RUN mkdir /opt/oracle
WORKDIR /opt/oracle 
ADD ./instantclient_11_2 ./
RUN mv libclntsh.so.11.1 libclntsh.so
WORKDIR /lib64
ADD ./lib64 ./
RUN sh -c "echo /opt/oracle > /etc/ld.so.conf.d/oracle-instantclient.conf"
RUN ldconfig
ENV LD_LIBRARY_PATH=/opt/oracle:/lib64:$LD_LIBRARY_PATH

WORKDIR /go/src/github.com/patomp3/wfacore
RUN go get -d -v github.com/gorilla/mux
RUN go get -d -v github.com/spf13/viper
COPY .  .
#COPY reconnect.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o wfacore .

FROM weeradej/alpine-oracle-instantclient
ENV TNS_LANG=THAI_THAILAND.TH8TISASCII
RUN adduser -S -D -H -h /app appuser
USER appuser

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/patomp3/wfacore .
CMD ["./wfacore"]
#CMD ["./icc-reconnect", "production"]