FROM golang:1.23

WORKDIR /cni
COPY . /cni

RUN CGO_ENABLED=0 go build -o output/plugin github.com/moycat-ucr/cs208-cni

FROM debian:10-slim

RUN apt update && \
    apt install -y iptables && \
    apt clean && \
    rm -rf /var/lib/apt/lists/* /var/log/dpkg.log /var/log/apt/*

COPY --from=0 /cni/output/plugin /usr/bin/plugin
CMD ["/usr/bin/plugin"]
