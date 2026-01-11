module cdn-agent

go 1.24.0

toolchain go1.24.11

require (
	cdn-common v0.0.0
	github.com/florianl/go-nfqueue v1.3.2
	github.com/google/gopacket v1.1.19
)

require (
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/go-acme/lego/v4 v4.16.1 // indirect
	github.com/go-jose/go-jose/v4 v4.0.1 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/josharian/native v1.0.0 // indirect
	github.com/mdlayher/netlink v1.6.0 // indirect
	github.com/mdlayher/socket v0.1.1 // indirect
	github.com/miekg/dns v1.1.58 // indirect
	golang.org/x/crypto v0.19.0 // indirect
	golang.org/x/mod v0.14.0 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.17.0 // indirect
)

replace cdn-common => ../common
