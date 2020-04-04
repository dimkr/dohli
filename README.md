```
     _       _     _ _ 
  __| | ___ | |__ | (_)
 / _` |/ _ \| '_ \| | |
| (_| | (_) | | | | | |
 \__,_|\___/|_| |_|_|_|
```

[![Build Status](https://travis-ci.org/dimkr/dohli.svg?branch=master)](https://travis-ci.org/dimkr/dohli) [![Report Card](https://goreportcard.com/badge/github.com/dimkr/dohli)](https://goreportcard.com/report/github.com/dimkr/dohli) [![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy)

## Overview

dohli (pronounced do-kh-li) is a simple and easy to deploy DNS-over-HTTPS (DoH) server that blocks ads and malicious sites.

One of the central points of critique against DoH, is the centralization of data: the big companies that power the internet-scale DoH servers used by default by applications like web browsers, are granted the privilege of unique access to browsing data that spans multiple sites or devices, risking the privacy and security of users.

User-owned DoH servers are a way to solve this problem. In addition, they provide an additional layer of privacy, by masking the DoH client address. Also, DoH servers like dohli, that use a random DNS server for each query, give each company a partial view of the user's browsing habits.

## Implementation

dohli is written in [Go](https://golang.org/).

It performs the actual resolving work using traditional DNS over UDP.

It uses [Redis](https://redis.io/) to cache DNS responses, and as a job queue.

A worker container gets notified each time a new domain name is resolved, then checks whether or not this domain should be blocked, against [Steven Black's unified domain blacklist](https://github.com/StevenBlack/hosts) and [URLHaus](https://urlhaus.abuse.ch).

If yes, blocking is performed by inserting a cache entry that has no expiration time. Therefore, dohli needs some time for "training" and the client's DNS cache must expire, before ads are blocked.

## CI/CD

Every day, dohli's [CI/CD pipeline](https://travis-ci.org/github/dimkr/dohli/builds) deploys the `master` branch to `https://dohli.herokuapp.com`, with an updated domain blacklist.

## Usage

First, [deploy to Heroku](https://heroku.com/deploy).

Then, append `/dns-query` to the web URL and configure your DoH client to use this as the DoH server.

For example, DoH clients that use the dohli instance deployed by CI/CD should use `https://dohli.herokuapp.com/dns-query`.

![Android](https://github.com/dimkr/dohli/raw/master/static/android.png) ![Firefox](https://github.com/dimkr/dohli/raw/master/static/firefox.png)

## Deployment from CLI

```
heroku create -s container --addons heroku-redis
heroku redis:maxmemory $ADDON_NAME --policy allkeys-lru
git push heroku master
heroku ps:scale web=1 worker=1
```

## Legal Information

dohli is free and unencumbered software released under the terms of the MIT license; see COPYING for the license text.

The ASCII art logo at the top was made using [FIGlet](http://www.figlet.org/).
