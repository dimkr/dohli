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

// Package cache implements DNS response cache.
package cache

import (
	"fmt"

	"golang.org/x/net/dns/dnsmessage"
)

// Cache is a DNS response cache.
type Cache struct {
	backend CacheBackend
}

// OpenCache opens the cache.
func OpenCache(backend CacheBackend) (*Cache, error) {
	if err := backend.Connect(); err != nil {
		return nil, err
	}

	return &Cache{backend: backend}, nil
}

func getCacheKey(domain string, requestType dnsmessage.Type) string {
	return fmt.Sprintf("%s:%d", domain, int(requestType))
}

// Get returns a cached DNS response, or nil.
func (c *Cache) Get(domain string, requestType dnsmessage.Type) []byte {
	if response := c.backend.Get(getCacheKey(domain, requestType)); response != nil {
		return response
	}

	return nil
}

// Set adds a DNS response in the cache, or replaces a cache entry, while
// optionally settings the cache entry's expiry time (specified in seconds, 0
// means no expiry).
func (c *Cache) Set(domain string, requestType dnsmessage.Type, response []byte, expiry int) {
	c.backend.Set(getCacheKey(domain, requestType), response, expiry)
}
