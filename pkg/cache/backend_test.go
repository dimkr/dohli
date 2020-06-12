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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/dns/dnsmessage"
)

type MockBackend struct {
	mock.Mock
}

func (mb *MockBackend) Connect() error {
	return mb.Called().Error(0)
}

func (mb *MockBackend) WithContext(_ context.Context) CacheBackend {
	return mb
}

func (mb *MockBackend) Set(key string, value []byte, ttl int) {
	mb.Called(key, value, ttl)
}

func (mb *MockBackend) Get(key string) []byte {
	if val, ok := mb.Called(key).Get(0).([]byte); ok {
		return val
	}

	return nil
}

func TestConnect(t *testing.T) {
	backend := MockBackend{}

	backend.On("Connect").Return(nil).Once()
	_, err := OpenCache(&backend)

	assert.Nil(t, err)
}

func TestConnectFailed(t *testing.T) {
	backend := MockBackend{}

	backend.On("Connect").Return(errors.New("Error")).Once()
	_, err := OpenCache(&backend)

	assert.NotNil(t, err)
}

func TestGet(t *testing.T) {
	backend := MockBackend{}

	backend.On("Connect").Return(nil).Once()
	cache, _ := OpenCache(&backend)

	response := []byte{1, 2, 3, 4}
	backend.On("Get", getCacheKey("wikipedia.org", dnsmessage.TypeA)).Return(response).Once()
	assert.Equal(t, cache.Get(context.Background(), "wikipedia.org", dnsmessage.TypeA), response)
}

func TestGetMiss(t *testing.T) {
	backend := MockBackend{}

	backend.On("Connect").Return(nil).Once()
	cache, _ := OpenCache(&backend)

	backend.On("Get", getCacheKey("wikipedia.org", dnsmessage.TypeA)).Return(nil).Once()
	assert.Nil(t, cache.Get(context.Background(), "wikipedia.org", dnsmessage.TypeA))
}

func TestSet(t *testing.T) {
	backend := MockBackend{}

	backend.On("Connect").Return(nil).Once()
	cache, _ := OpenCache(&backend)

	response := []byte{1, 2, 3, 4}
	backend.On("Set", getCacheKey("wikipedia.org", dnsmessage.TypeA), response, 3600).Return(response).Once()
	cache.Set(context.Background(), "wikipedia.org", dnsmessage.TypeA, response, 3600)
}
