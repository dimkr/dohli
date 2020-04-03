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

package dns

import (
	"errors"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// GetShortestTTL returns the lowest TTL among the answer records of a DNS
// response, or 0.
func GetShortestTTL(response []byte) time.Duration {
	var p dnsmessage.Parser

	if _, err := p.Start(response); err != nil {
		return 0
	}

	if err := p.SkipAllQuestions(); err != nil {
		return 0
	}

	ns := time.Second.Nanoseconds()
	var shortestTTL int64
	first := true

	for {
		a, err := p.Answer()
		if errors.Is(err, dnsmessage.ErrSectionDone) {
			break
		}

		if err != nil {
			break
		}

		ttl := int64(a.Header.TTL) * ns
		if first || ttl < shortestTTL {
			shortestTTL = ttl
		}
		first = false
	}

	return time.Duration(shortestTTL)
}

// ReplaceTTLInResponse sets the TTL of all answer records in a DNS response.
func ReplaceTTLInResponse(response []byte, TTL uint32) ([]byte, error) {
	var p dnsmessage.Parser

	header, err := p.Start(response)
	if err != nil {
		return nil, err
	}

	questions, err := p.AllQuestions()
	if err != nil {
		return nil, err
	}

	var additionals []dnsmessage.Resource

	additionals, err = p.AllAdditionals()
	if err != nil && !errors.Is(err, dnsmessage.ErrNotStarted) {
		return nil, err
	}
	var answers []dnsmessage.Resource

	for {
		answer, err := p.Answer()
		if errors.Is(err, dnsmessage.ErrSectionDone) {
			break
		}
		if err != nil {
			return nil, err
		}

		answer.Header.TTL = TTL

		answers = append(answers, answer)
	}

	msg := dnsmessage.Message{
		Header:      header,
		Questions:   questions,
		Answers:     answers,
		Additionals: additionals,
	}

	return msg.Pack()
}
