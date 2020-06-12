# this file is part of dohli.
#
# Copyright (c) 2020 Dima Krasner
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

ARG GO_VERSION=1.13
FROM golang:$GO_VERSION-alpine AS builder

ADD cmd/ /src/cmd
ADD pkg/ /src/pkg
ADD go.mod /src/go.mod
ADD go.sum /src/go.sum
WORKDIR /src
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /stub ./cmd/stub
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /web ./cmd/web
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /worker ./cmd/worker

FROM debian:buster-slim AS hosts
RUN apt-get -qq update && apt-get install -y --no-install-recommends python3 python3-pip python3-wheel git wget
RUN git clone --depth 1 https://github.com/StevenBlack/hosts /hosts
WORKDIR /hosts
RUN pip3 install -r requirements.txt
RUN for i in data/*/update.json; do [ -z "`cat $i | grep license | grep -e CC -e MIT`" ] && rm -vf $i; done
RUN python3 updateHostsFile.py --auto -s -m -e "fakenews gambling porn social"

RUN git clone --depth 1 https://github.com/EnergizedProtection/block /block
WORKDIR /block
RUN wget -O- `cat README.md | grep SOURCE | grep -e "CC BY-SA" -e MIT -e BSD -e Permissive -e Apache  | cut -f 4 -d \] | cut -f 2 -d \( | cut -f 1 -d \)` > hosts
RUN cat hosts /hosts/hosts | grep -v -e ^\# -e '^$' | sort | uniq > /tmp/hosts.block

FROM alpine
ADD static/ /static
COPY --from=hosts /tmp/hosts.block /hosts.block
COPY --from=builder /stub /stub
COPY --from=builder /web /web
COPY --from=builder /worker /worker
