# Overview

A high performance and lightweight gateway to automatically reverse proxy TLS/TCP requests.
It uses the [SNI](https://en.wikipedia.org/wiki/Server_Name_Indication) to route the tcp connection.
It will automatically use the ACME client to obtain, cache, and renew TLS certificates.

Gate will reserve a subdomain `gate-tunnel` on each domain for server-client communication.

## Quick start

Create `gate.json` file like below (doc for config: [conf.go](lib/conf/conf.go)):

```json
{
    "domains": [
        {
            "domain": "test.com",
            "provider": "cloudflare",
            "token": "abc",
            "routes": [
                {
                    "token": "xxx"
                },
                {
                    "token": "yyy"
                }
            ]
        }
    ]
}
```

Run `go run ./cmd/server` to start the gate service.

Run `go run ./cmd/client -h test.com -t xxx -d :3000`
to proxy TLS connections from `test.com` to `:3000`.

Run `go run ./cmd/client -h test.com -s sub -t yyy -d :3001`
to proxy TLS connections from subdomain `sub.test.com` to `:3001`.

Run two test http servers `go run ./cmd/hello -p :3000` and `go run ./cmd/hello -p :3001`

The clients don't have to be in the same network as the server, they will use
TLS to securely communicate with each other.

The diagram below shows how the above setup works:

```mermaid
flowchart TB
    browser1["browser visits test.com"]
    browser2["browser visits sub.test.com"]
    server["go run ./cmd/server"]
    x-client["go run ./cmd/client -h test.com -t xxx -d :3000"]
    y-client["go run ./cmd/client -h test.com -s sub -t yyy -d :3001"]
    x["go run ./cmd/hello -p :3000"]
    y["go run ./cmd/hello -p :3001"]

    browser1 & browser2 -- tls/tcp --> server

    server -- "tls tunnel (gate-tunnel.test.com)" --> x-client & y-client

    x-client -- tcp --> x 
    y-client -- tcp --> y
```

## Client lib

If you use golang as the backend you don't have to use the sidecar client binary, you can use the client lib directly:

```go
package main

import (
    "log"
    "net/http"

    "github.com/ysmood/gate/lib/client"
)

func main() {
    l, err := client.New("test.com", "x", "xxx")
    if err != nil {
        log.Fatal(err)
    }

    http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("hello!"))
    }))
}
```

## TODO

### [mTLS](https://en.wikipedia.org/wiki/Mutual_authentication)

Such as use the same domain for both staging/production env. The staging service tells gate it only accepts the requests coming from cert1, other certs will be routed to production service.

```mermaid
flowchart TB
    cert1["cert1 visits test.com"]
    cert2["cert2 visits test.com"]
    gate
    stg["staging service on :3000"]
    prd["production service on :3001"]

    cert1 & cert2 -- tls --> gate
    gate -- tcp from cert1 --> stg
    gate -- tcp from cert2 --> prd
```
