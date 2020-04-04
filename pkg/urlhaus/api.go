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

package urlhaus

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/dimkr/dohli/pkg/queue"
)

const (
	url     = "https://urlhaus-api.abuse.ch"
	timeout = 5 * time.Second
)

type UrlhausAPI struct {
	client http.Client
}

func (api *UrlhausAPI) Connect() error {
	api.client.Timeout = timeout
	return nil
}

func (api *UrlhausAPI) IsAsync() bool {
	return true
}

type hostResponse struct {
	QueryStatus string            `json:"query_status"`
	Blacklists  map[string]string `json:"blacklists"`
}

func (api *UrlhausAPI) IsBad(msg *queue.DomainAccessMessage) bool {
	response, err := api.client.Post(url+"/v1/host", "application/x-www-form-urlencoded", bytes.NewBuffer([]byte("host="+msg.Domain)))
	if err != nil {
		return false
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return false
	}

	j, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return false
	}

	var parsedResponse hostResponse
	if err := json.Unmarshal(j, &parsedResponse); err != nil {
		log.Printf("Failed to parse %s: %v", j, err)
		return false
	}

	if parsedResponse.QueryStatus != "ok" {
		return false
	}

	for _, status := range parsedResponse.Blacklists {
		if status != "not listed" {
			log.Print(msg.Domain + " is blocked by URLHaus")
			return true
		}
	}

	return false
}
