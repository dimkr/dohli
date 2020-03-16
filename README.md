```
     _       _     _ _ 
  __| | ___ | |__ | (_)
 / _` |/ _ \| '_ \| | |
| (_| | (_) | | | | | |
 \__,_|\___/|_| |_|_|_|
```

## Overview

dohli (pronounced do-kh-li) is a simple and easy to deploy DNS-over-HTTPS (DoH) server that blocks ads and malicious sites.

One of the central points of critique against DoH, is the centralization of data: the big companies that power the internet-scale DoH servers used by default by applications like web browsers, are granted the privilege of unique access to browsing data that spans multiple sites or devices, risking the privacy and security of users.

User-owned DoH servers are a way to solve this problem. In addition, they provide an additional layer of privacy, by masking the DoH client address.

## Implementation

dohli is written in [Go](https://golang.org/) and uses [Steven Black's unified domain blacklist](https://github.com/StevenBlack/hosts) and [URLHaus](https://urlhaus.abuse.ch).

## Usage

```
heroku create -s container --addons heroku-redis
heroku redis:maxmemory $ADDON_NAME --policy allkeys-lru
git push heroku master
```

Then, append `/dns-query` to the web URL and configure your DoH client to use this as the DoH server.

## Legal Information

dohli is free and unencumbered software released under the terms of the MIT license; see COPYING for the license text.

The ASCII art logo at the top was made using [FIGlet](http://www.figlet.org/).
