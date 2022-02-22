# Build the manager binary
FROM golang:1.15 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer

#RUN go mod download
RUN GOPROXY=https://goproxy.cn go mod download

# Copy the go source
COPY . ./

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o kube-log-helper main.go

FROM python:3.8-buster

# oss distribution: oss-x.x.x
# default distribution: x.x.x
ENV FILEBEAT_VERSION=7.8.0

RUN wget https://artifacts.elastic.co/downloads/beats/filebeat/filebeat-oss-${FILEBEAT_VERSION}-linux-x86_64.tar.gz -P /tmp/ && \
    mkdir -p /etc/filebeat /var/lib/filebeat /var/log/filebeat && \
    tar zxf /tmp/filebeat-oss-${FILEBEAT_VERSION}-linux-x86_64.tar.gz -C /tmp/ && \
    cp -rf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64/filebeat /usr/bin/ && \
    cp -rf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64/fields.yml /etc/filebeat/ && \
    cp -rf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64/kibana /etc/filebeat/ && \
#    cp -rf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64/module /etc/filebeat/ && \
#    cp -rf /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64/modules.d /etc/filebeat/ && \
    rm -rf /var/lib/apt/lists/* /tmp/filebeat-oss-${FILEBEAT_VERSION}-linux-x86_64.tar.gz /tmp/filebeat-${FILEBEAT_VERSION}-linux-x86_64

COPY --from=builder /workspace/kube-log-helper /srv/kube-log-helper
COPY assets/entrypoint assets/filebeat/ assets/healthz /srv/

RUN chmod +x /srv/kube-log-helper /srv/entrypoint /srv/healthz /srv/config.filebeat

HEALTHCHECK CMD /srv/healthz

VOLUME /var/log/filebeat
VOLUME /var/lib/filebeat

WORKDIR /srv/
ENTRYPOINT ["/srv/entrypoint"]
