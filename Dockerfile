FROM golang:alpine AS builder

ADD cmd/ /src/cmd
ADD pkg/ /src/pkg
ADD go.mod /src/go.mod
ADD go.sum /src/go.sum
WORKDIR /src
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /web ./cmd/web
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /worker ./cmd/worker

RUN apk add gcc musl-dev python3 py3-pip python3-dev libxml2-dev libxslt-dev git
RUN git clone --depth 1 https://github.com/StevenBlack/hosts /hosts
WORKDIR /hosts
RUN pip3 install -r requirements.txt
RUN python3 updateHostsFile.py --auto -s -m -e "fakenews gambling porn social"

FROM alpine
COPY --from=builder /hosts/hosts /hosts.block
COPY --from=builder /web /web
COPY --from=builder /worker /worker
