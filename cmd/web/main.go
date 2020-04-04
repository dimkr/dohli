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

// web is a caching DoH server.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
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
	"golang.org/x/sync/semaphore"
)

const (
	defaultUpstreamServers = "1.1.1.1,8.8.8.8,9.9.9.9"

	minDNSCacheDuration = int(time.Hour / time.Second)
	maxDNSCacheDuration = int(time.Hour/time.Second) * 6

	maxResolvingOperations = 512

	// in seconds
	responseTTL = 60 * 30

	staticAssertRequestTimeout = 5 * time.Second
	resolvingRequestTimeout    = 3 * time.Second

	readTimeout  = 5 * time.Second
	writeTimeout = 5 * time.Second
	idleTimeout  = 10 * time.Second

	resolvingTimeout = 3 * time.Second
)

var upstreamServers []string

var sem *semaphore.Weighted
var c *cache.Cache
var q *queue.Queue

func resolveWithUpstream(question dnsmessage.Question, request []byte) []byte {
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

	conn.SetDeadline(time.Now().Add(resolvingTimeout))

	if _, err := conn.Write(request); err != nil {
		return nil
	}

	buf := make([]byte, 512)

	len, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return nil
	}

	return buf[:len]
}

func resolve(ctx context.Context, question dnsmessage.Question, request []byte) []byte {
	if err := sem.Acquire(ctx, 1); err != nil {
		return nil
	}
	defer sem.Release(1)

	domain := strings.TrimSuffix(question.Name.String(), ".")

	// Chrome resolves junk domains without a dot
	if strings.Index(domain, ".") == -1 {
		response, err := dns.BuildNXDomainResponse(domain, question.Type)
		if err == nil {
			return response
		}
		return nil
	}

	if cachedResponse := c.Get(domain, question.Type); cachedResponse != nil {
		return cachedResponse
	}

	responseChan := make(chan []byte)

	go func() {
		responseChan <- resolveWithUpstream(question, request)
	}()

	var response []byte

	select {
	case response = <-responseChan:
		break

	case <-ctx.Done():
		return nil
	}

	go func() {
		ttl := dns.GetShortestTTL(response)
		if ttl < minDNSCacheDuration {
			ttl = minDNSCacheDuration
		} else if ttl > maxDNSCacheDuration {
			ttl = maxDNSCacheDuration
		}
		c.Set(domain, question.Type, response, ttl)

		// we want the worker to replace the cache entry we just inserted
		if j, err := json.Marshal(queue.DomainAccessMessage{
			Domain:      domain,
			RequestType: question.Type,
		}); err == nil {
			q.Push(string(j))
		}
	}()

	// we don't want DNS responses to have high TTL, because that would prevent
	// us from blocking them in the future, or have low TTL, which increases
	// the number of requests we serve
	if response, err := dns.ReplaceTTLInResponse(response, responseTTL); err == nil {
		return response
	}

	return response
}

func handleDNSQuery(w http.ResponseWriter, r *http.Request) {
	var body []byte
	var err error

	switch r.Method {
	case http.MethodPost:
		body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	case http.MethodGet:
		dns, ok := r.URL.Query()["dns"]
		if !ok {
			http.Redirect(w, r, "/", http.StatusMovedPermanently)
			return
		}

		if len(dns[0]) == 0 {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		body, err = base64.RawURLEncoding.DecodeString(dns[0])
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

	default:
		http.Error(w, "Bad method", http.StatusMethodNotAllowed)
		return
	}

	if len(body) == 0 {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
		return
	}

	var p dnsmessage.Parser

	if _, err := p.Start(body); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	question, err := p.Question()
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	resolvingChan := make(chan []byte)

	go func() {
		resolvingChan <- resolve(r.Context(), question, body)
	}()

	select {
	case buf := <-resolvingChan:
		if buf == nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
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
	if c, err = cache.OpenCache(&cache.RedisBackend{}); err != nil {
		panic(err)
	}

	if q, err = queue.OpenQueue(); err != nil {
		panic(err)
	}

	sem = semaphore.NewWeighted(maxResolvingOperations)

	mux := http.ServeMux{}
	mux.Handle("/", http.TimeoutHandler(http.StripPrefix("/", http.FileServer(http.Dir("/static"))), staticAssertRequestTimeout, "Timeout"))
	mux.Handle("/dns-query", http.TimeoutHandler(http.HandlerFunc(handleDNSQuery), resolvingRequestTimeout, "Timeout"))

	server := http.Server{
		Addr:         ":" + port,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
		Handler:      &mux,
	}

	server.ListenAndServe()
}
