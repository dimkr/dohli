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

// worker monitors for domain access events and blocks domains using the cache.
package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dimkr/dohli/pkg/cache"
	"github.com/dimkr/dohli/pkg/dns"
	"github.com/dimkr/dohli/pkg/hosts"
	"github.com/dimkr/dohli/pkg/queue"
	"github.com/dimkr/dohli/pkg/urlhaus"
	"golang.org/x/net/dns/dnsmessage"
)

const (
	// no expiration
	blockedDomainTTL = 0
)

type blocker interface {
	Connect() error
	IsBad(*queue.DomainAccessMessage) bool
	IsAsync() bool
}

var c *cache.Cache
var q *queue.Queue
var blockers []blocker = []blocker{&hosts.HostsBlacklist{}, &urlhaus.Client{}}

func doBlockDomain(domain string, requestType dnsmessage.Type) error {
	response, err := dns.BuildNXDomainResponse(domain, requestType)
	if err == nil {
		c.Set(domain, requestType, response, blockedDomainTTL)
	}
	return err
}

func blockDomain(msg *queue.DomainAccessMessage) {
	log.Print("Blocking " + msg.Domain)

	if err := doBlockDomain(msg.Domain, msg.RequestType); err != nil {
		log.Printf("Failed to block %s: %v", msg.Domain, err)
	}

	var otherType dnsmessage.Type

	switch msg.RequestType {
	case dnsmessage.TypeA:
		otherType = dnsmessage.TypeAAAA

	case dnsmessage.TypeAAAA:
		otherType = dnsmessage.TypeA

	default:
		return
	}

	if err := doBlockDomain(msg.Domain, otherType); err != nil {
		log.Printf("Failed to block %s: %v", msg.Domain, err)
	}
}

func blockDomainIfNeeded(msg *queue.DomainAccessMessage) {
	for _, b := range blockers {
		if !b.IsAsync() && b.IsBad(msg) {
			blockDomain(msg)
			return
		}
	}

	verdicts := make(chan bool)
	n := 0

	for _, b := range blockers {
		if !b.IsAsync() {
			continue
		}

		n++

		go func(b blocker) {
			verdicts <- b.IsBad(msg)
		}(b)
	}

	for i := 0; i < n; i++ {
		select {
		case shouldBlock := <-verdicts:
			if shouldBlock {
				blockDomain(msg)
				return
			}
		}
	}
}

func handleDomainAccess(j string) {
	var msg queue.DomainAccessMessage
	if err := json.Unmarshal([]byte(j), &msg); err != nil {
		log.Printf("Failed to parse %s: %v", j, err)
		return
	}

	blockDomainIfNeeded(&msg)
}

func handleMessages() {
	for {
		j, err := q.Pop()
		if err != nil {
			log.Printf("Failed to receive a message: %v", err)
			break
		}

		handleDomainAccess(j)
	}
}

func main() {
	var err error

	if c, err = cache.OpenCache(&cache.RedisBackend{}); err != nil {
		panic(err)
	}

	if q, err = queue.OpenQueue(); err != nil {
		panic(err)
	}

	for _, b := range blockers {
		if err = b.Connect(); err != nil {
			panic(err)
		}
	}

	go handleMessages()

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
}
