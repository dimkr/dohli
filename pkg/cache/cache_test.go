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
	"fmt"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

func ExampleCache_Get() {
	cache, err := OpenCache(&DummyBackend{})
	if err != nil {
		panic(err)
	}

	cache.Set("wikipedia.org", dnsmessage.TypeA, []byte{1, 2, 3, 4}, time.Hour)

	fmt.Print(cache.Get("wikipedia.org", dnsmessage.TypeA))
	// Output: [1 2 3 4]
}

func TestCacheGetNoKeys(t *testing.T) {
	cache, _ := OpenCache(&DummyBackend{})

	if cache.Get("wikipedia.org", dnsmessage.TypeA) != nil {
		t.Error()
	}
}

func TestCacheGet(t *testing.T) {
	cache, _ := OpenCache(&DummyBackend{})

	val := []byte{1, 2, 3, 4}
	cache.Set("wikipedia.org", dnsmessage.TypeA, val, time.Hour)

	cached := cache.Get("wikipedia.org", dnsmessage.TypeA)
	if cached == nil || !reflect.DeepEqual(cached, val) {
		t.Error()
	}
}

func TestCacheGetDifferentType(t *testing.T) {
	cache, _ := OpenCache(&DummyBackend{})

	cache.Set("wikipedia.org", dnsmessage.TypeA, []byte{1, 2, 3, 4}, time.Hour)

	if cache.Get("wikipedia.org", dnsmessage.TypeAAAA) != nil {
		t.Error()
	}
}

func TestCacheGetReplace(t *testing.T) {
	cache, _ := OpenCache(&DummyBackend{})

	val := []byte{1, 2, 3, 4}
	cache.Set("wikipedia.org", dnsmessage.TypeA, val, time.Hour)

	cached := cache.Get("wikipedia.org", dnsmessage.TypeA)
	if cached == nil || !reflect.DeepEqual(cached, val) {
		t.Error()
	}

	val2 := []byte{5, 6, 7, 8}

	cached = cache.Get("wikipedia.org", dnsmessage.TypeA)
	if cached == nil || reflect.DeepEqual(cached, val2) {
		t.Error()
	}

	cache.Set("wikipedia.org", dnsmessage.TypeA, val2, time.Hour)

	cached = cache.Get("wikipedia.org", dnsmessage.TypeA)
	if cached == nil || !reflect.DeepEqual(cached, val2) {
		t.Error()
	}
}

func TestCacheGetMissingKey(t *testing.T) {
	cache, _ := OpenCache(&DummyBackend{})

	cache.Set("wikipedia.org", dnsmessage.TypeA, []byte{1, 2, 3, 4}, time.Hour)

	cached := cache.Get("wikipedia.or", dnsmessage.TypeA)
	if cached != nil {
		t.Error()
	}
}
