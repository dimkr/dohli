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
	"log"
	"time"

	"gopkg.in/redis.v5"
)

// URLEnvironmentVariable is the name of the environment variable containing the
// Redis URL.
const URLEnvironmentVariable = "REDIS_URL"

// RedisBackend is a Redis-based caching backend.
type RedisBackend struct {
	CacheBackend
	client *redis.Client
}

func (rb *RedisBackend) Connect() error {
	opts, err := redis.ParseURL(URLEnvironmentVariable)
	if err != nil {
		return err
	}

	rb.client = redis.NewClient(opts)
	return nil
}

func (rb *RedisBackend) Get(key string) []byte {
	response, err := rb.client.Get(key).Result()
	if err != nil {
		return nil
	}

	rawResponse, err := hex.DecodeString(response)
	if err != nil {
		return nil
	}

	return rawResponse
}

func (rb *RedisBackend) Set(key string, value []byte, ttl time.Duration) {
	if _, err := rb.client.Set(key, hex.EncodeToString(value), ttl).Result(); err != nil {
		log.Printf("Failed to cache a DNS response: %v", err)
	}
}
