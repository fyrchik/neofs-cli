FROM golang:1-alpine as builder

ARG BUILD=now
ARG VERSION=dev
ARG REPO=github.com/nspcc-dev/neofs-cli

ENV GOGC off
ENV CGO_ENABLED 0
ENV LDFLAGS "-w -s -X ${REPO}/Version=${VERSION} -X ${REPO}/Build=${BUILD}"

WORKDIR /src

COPY . /src

RUN go build -v -mod=vendor -trimpath -ldflags "${LDFLAGS}" -o /go/bin/neofs-cli ./

# Executable image
FROM alpine:3.10 AS neofs-cli

WORKDIR /

RUN set -x \
  && apk add --no-cache bash \
  && echo "#!/bin/bash" >> ~/.bashrc \
  && echo "neofs-cli --help" >> ~/.bashrc \
  && chmod +rx ~/.bashrc

COPY --from=builder /go/bin/neofs-cli                  /bin/neofs-cli
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["bash"]
