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

package hosts

import (
	"bufio"
	"os"
	"strings"

	"github.com/dimkr/dohli/pkg/queue"
)

type HostsBlacklist struct {
	blockedDomains map[string]bool
}

func (hb *HostsBlacklist) Connect() error {
	hosts, err := os.Open("/hosts.block")
	if err != nil {
		return err
	}
	defer hosts.Close()

	hb.blockedDomains = make(map[string]bool)

	scanner := bufio.NewScanner(hosts)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "0.0.0.0 ") {
			hb.blockedDomains[line[len("0.0.0.0 "):]] = true
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (hb *HostsBlacklist) IsAsync() bool {
	return false
}

func (hb *HostsBlacklist) IsBad(msg *queue.DomainAccessMessage) bool {
	_, ok := hb.blockedDomains[msg.Domain]
	return ok
}
