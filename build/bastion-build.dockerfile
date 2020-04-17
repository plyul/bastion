FROM golang:1.13 as builder
ARG BASTION_VERSION
ENV GOROOT /usr/local/go
ENV GOPATH /go
ENV CGO_ENABLED 0
ENV GOOS linux

ENV BASTION_SERVER_EXE ${GOPATH}/bin/bastion-server
ENV BASTION_PROXY_EXE ${GOPATH}/bin/bastion-proxy

RUN apt-get update && apt-get -y install ca-certificates xz-utils

WORKDIR /opt
RUN curl -sSfL https://github.com/upx/upx/releases/download/v3.96/upx-3.96-amd64_linux.tar.xz | tar -xJ

WORKDIR ${GOPATH}/src/bastion
COPY . .

RUN ${GOROOT}/bin/go build -v -ldflags "-X main.Version=${BASTION_VERSION}" -o ${BASTION_SERVER_EXE} cmd/bastion-server/bastion-server.go
RUN /opt/upx-3.96-amd64_linux/upx --ultra-brute -q ${BASTION_SERVER_EXE}

RUN ${GOROOT}/bin/go build -v -ldflags "-X main.Version=${BASTION_VERSION}" -o ${BASTION_PROXY_EXE} cmd/bastion-proxy/bastion-proxy.go
RUN /opt/upx-3.96-amd64_linux/upx --ultra-brute -q ${BASTION_PROXY_EXE}

FROM scratch as bastion-server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/bastion/web /srv/bastion/web
COPY --from=builder /go/bin/bastion-server /opt/bastion/bastion-server
EXPOSE 1443/tcp
ENTRYPOINT ["/opt/bastion/bastion-server", \
            "--web-templates", "/srv/bastion/web/templates", \
            "--web-static", "/srv/bastion/web/webroot/static", \
            "--tls-cert-file", "/srv/bastion/web/certs/bastion-cert.pem", \
            "--tls-key-file", "/srv/bastion/web/certs/bastion-key.pem"]

FROM scratch as bastion-proxy
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/bastion-proxy /opt/bastion/bastion-proxy
COPY --from=builder /go/src/bastion/web/certs/bastion-cert.pem /srv/bastion/web/certs/bastion-cert.pem
EXPOSE 2200/tcp
ENTRYPOINT ["/opt/bastion/bastion-proxy", \
            "--api-cert", "/srv/bastion/web/certs/bastion-cert.pem"]
