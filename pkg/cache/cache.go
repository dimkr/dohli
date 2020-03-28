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

package cache

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/net/dns/dnsmessage"
	"gopkg.in/redis.v5"
)

type Cache struct {
	redisClient *redis.Client
}

func OpenCache() (*Cache, error) {
	opts, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		return nil, err
	}

	return &Cache{redisClient: redis.NewClient(opts)}, nil
}

func getCacheKey(domain string, requestType dnsmessage.Type) string {
	return fmt.Sprintf("%s:%d", domain, int(requestType))
}

func (c *Cache) Get(domain string, requestType dnsmessage.Type) []byte {
	response, err := c.redisClient.Get(getCacheKey(domain, requestType)).Result()
	if err == nil {
		rawResponse, err := hex.DecodeString(response)
		if err == nil {
			return rawResponse
		} else {
			log.Printf("Failed to decode a cached DNS response")
		}
	}

	return nil
}

func (c *Cache) Set(domain string, requestType dnsmessage.Type, response []byte, TTL time.Duration) {
	if _, err := c.redisClient.Set(getCacheKey(domain, requestType), hex.EncodeToString(response), TTL).Result(); err != nil {
		log.Printf("Failed to cache a DNS response: %v", err)
	}
}
