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

package main

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dimkr/dohli/pkg/cache"
	"github.com/dimkr/dohli/pkg/dns"
	"github.com/dimkr/dohli/pkg/queue"
	"golang.org/x/net/dns/dnsmessage"
)

const (
	defaultUpstreamServers = "1.1.1.1,8.8.8.8,9.9.9.9"

	minDNSCacheDuration = time.Hour
	maxDNSCacheDuration = time.Hour * 6

	// in seconds
	responseTTL = 60 * 30
)

var upstreamServers []string

var c *cache.Cache
var q *queue.Queue

func getAddresses(response []byte, domain string) []string {
	var p dnsmessage.Parser
	var addresses []string

	if _, err := p.Start(response); err != nil {
		log.Printf("Parsing failed: %v", err)
		return addresses
	}

	if err := p.SkipAllQuestions(); err != nil {
		return addresses
	}

	for {
		var addr []byte

		h, err := p.AnswerHeader()
		if err == dnsmessage.ErrSectionDone {
			break
		}
		if err != nil {
			log.Printf("DNS answer parsing failed: %v", err)
			break
		}

		if (h.Type != dnsmessage.TypeA && h.Type != dnsmessage.TypeAAAA) ||
			h.Class != dnsmessage.ClassINET {
			if err := p.SkipAnswer(); err != nil {
				break
			}
			continue
		}

		if !strings.EqualFold(h.Name.String(), domain) {
			if err := p.SkipAnswer(); err != nil {
				break
			}
			continue
		}

		switch h.Type {
		case dnsmessage.TypeA:
			r, err := p.AResource()
			if err != nil {
				log.Printf("A record parsing failed: %v", err)
				break
			}
			addr = r.A[:]

		case dnsmessage.TypeAAAA:
			r, err := p.AAAAResource()
			if err != nil {
				log.Printf("AAAA record parsing failed: %v", err)
				break
			}
			addr = r.AAAA[:]

		default:
			continue
		}

		addresses = append(addresses, net.IP(addr).String())
	}

	return addresses
}

func resolve(question dnsmessage.Question, request []byte) []byte {
	domain := strings.TrimSuffix(question.Name.String(), ".")

	response := c.Get(domain, question.Type)
	if response != nil {
		return response
	}

	var upstream string
	if len(upstreamServers) > 1 {
		upstream = upstreamServers[rand.Intn(len(upstreamServers))]
	} else {
		upstream = upstreamServers[0]
	}

	addr, err := net.ResolveUDPAddr("udp", upstream+":53")
	if err != nil {
		return nil
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(time.Second * 10))

	if _, err := conn.Write(request); err != nil {
		return nil
	}

	buf := make([]byte, 512)

	len, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return nil
	}

	go func() {
		ttl := dns.GetShortestTTL(buf[:len])
		if ttl < minDNSCacheDuration {
			ttl = minDNSCacheDuration
		} else if ttl > maxDNSCacheDuration {
			ttl = maxDNSCacheDuration
		}
		c.Set(domain, question.Type, buf[:len], ttl)

		// we want the worker to replace the cache entry we just inserted
		if j, err := json.Marshal(queue.DomainAccessMessage{
			Domain:      domain,
			RequestType: question.Type,
			Addresses:   getAddresses(buf[:len], question.Name.String()),
		}); err == nil {
			q.Push(string(j))
		}
	}()

	// we don't want DNS responses to have high TTL, because that would prevent
	// us from blocking them in the future, or have low TTL, which increases
	// the number of requests we serve

	if response, err := dns.ReplaceTTLInResponse(buf[:len], responseTTL); err == nil {
		return response
	}

	return buf[:len]
}

func handleDNSQuery(w http.ResponseWriter, r *http.Request) {
	var body []byte
	var err error

	switch r.Method {
	case "POST":
		body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

	case "GET":
		dns, ok := r.URL.Query()["dns"]
		if !ok {
			http.Redirect(w, r, "/", 301)
			return
		}

		if len(dns[0]) == 0 {
			http.Error(w, "Bad request", 400)
			return
		}

		body, err = base64.RawURLEncoding.DecodeString(dns[0])
		if err != nil {
			http.Error(w, "Bad request", 400)
			return
		}

	default:
		http.Error(w, "Bad request", 400)
		return
	}

	if len(body) == 0 {
		http.Redirect(w, r, "/", 301)
		return
	}

	var p dnsmessage.Parser

	if _, err := p.Start(body); err != nil {
		http.Error(w, "Bad request", 400)
		return
	}

	question, err := p.Question()
	if err != nil {
		http.Error(w, "Bad request", 400)
		return
	}

	// Chrome resolves junk domains without a dot
	domain := question.Name.String()
	if strings.Index(domain, ".") == len(domain)-1 {
		http.Error(w, "Bad request", 400)
		return
	}

	resolvingChan := make(chan []byte)

	go func() {
		resolvingChan <- resolve(question, body)
	}()

	select {
	case buf := <-resolvingChan:
		if buf == nil {
			http.Error(w, "Resolving failed", 500)
			return
		}

		w.Header().Set("Content-Type", "application/dns-message")
		if _, err := w.Write(buf); err != nil {
			return
		}

	case <-r.Context().Done():
		return
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	servers := os.Getenv("UPSTREAM_SERVERS")
	if servers == "" {
		servers = defaultUpstreamServers
	}

	upstreamServers = strings.Split(servers, ",")
	if len(upstreamServers) > 1 {
		rand.Seed(time.Now().Unix())
	}

	var err error
	if c, err = cache.OpenCache(); err != nil {
		panic(err)
	}

	if q, err = queue.OpenQueue(); err != nil {
		panic(err)
	}

	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("/static"))))
	http.HandleFunc("/dns-query", handleDNSQuery)
	http.ListenAndServe(":"+port, nil)
}
