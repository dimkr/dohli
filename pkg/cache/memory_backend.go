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
	"context"

	"github.com/coocood/freecache"
)

// MemoryBackend is an in-memory, freecache-based caching backend.
type MemoryBackend struct {
	CacheBackend
	cache *freecache.Cache
}

func (mb *MemoryBackend) Connect() error {
	mb.cache = freecache.NewCache(0)
	return nil
}

func (mb *MemoryBackend) WithContext(_ context.Context) CacheBackend {
	return mb
}

func (mb *MemoryBackend) Get(key string) []byte {
	if value, _ := mb.cache.Get([]byte(key)); value != nil {
		return value
	}

	return nil
}

func (mb *MemoryBackend) Set(key string, value []byte, expiry int) {
	mb.cache.Set([]byte(key), value, expiry)
}
