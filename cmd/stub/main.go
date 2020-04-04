// this file is part of dohli.
//
// Copyright (c) 2020 Dima Krasner
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// stub is a local DNS server that resolves using DoH.
package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/dimkr/dohli/pkg/cache"
	"github.com/dimkr/dohli/pkg/dns"
	"golang.org/x/net/dns/dnsmessage"
)

const (
	packetSize       = 512
	dnsHeaderSize    = 12
	fallbackTTL      = 60
	resolvingTimeout = 5 * time.Second
)

var server string
var client = http.Client{Timeout: resolvingTimeout}

func resolve(request []byte) []byte {
	response, err := client.Post(server, "application/dns-message", bytes.NewBuffer(request))
	if err != nil {
		log.Println("Resolving failed: ", err)
		return nil
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Println("Resolving failed: ", response.Status)
		return nil
	}

	if response.ContentLength != -1 && response.ContentLength < dnsHeaderSize {
		return nil
	}

	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println("Resolving failed: ", err)
		return nil
	}

	if len(buf) < dnsHeaderSize {
		return nil
	}

	return buf
}

func main() {
	flag.StringVar(&server, "server", "https://dohli.herokuapp.com/dns-query", "DoH server to use")
	flag.Parse()

	l, err := net.ListenPacket("udp4", ":53")
	if err != nil {
		panic(err)
	}

	cache, err := cache.OpenCache(&cache.MemoryBackend{})
	if err != nil {
		panic(err)
	}

	for {
		buf := make([]byte, packetSize)

		len, addr, err := l.ReadFrom(buf)
		if err != nil {
			continue
		}

		go func() {
			var p dnsmessage.Parser

			if _, err := p.Start(buf[:len]); err != nil {
				return
			}

			question, err := p.Question()
			if err != nil {
				return
			}

			domain := question.Name.String()

			if cached := cache.Get(domain, question.Type); cached != nil {
				// we join the first two bytes of the request (the ID) with the
				// cached response, which has a different ID
				l.WriteTo(append(buf[:2], cached[2:]...), addr)
				return
			}

			if response := resolve(buf[:len]); response != nil {
				l.WriteTo(response, addr)

				ttl := dns.GetShortestTTL(response)
				if ttl == 0 {
					ttl = fallbackTTL
				}

				cache.Set(domain, question.Type, response, ttl)
			}
		}()
	}
}
