# cfzone

This is a utility to update a cloudflare zone based on a bind-style [zone file](https://en.wikipedia.org/wiki/Zone_file).


## Limitations

Only `A`, `AAAA`, `CNAME`, `MX`, and `TXT` records are supported.

Cloudflare supported record types `LOC`, `NS`, `SRV`, `SPF` and `CAA` is not
currently supported.

Cloudflare supports (at least) two modes not easily representable in a BIND
zone. To support these features a few magic TTL values are used.

| TTL | Status                                  |
|-----|-----------------------------------------|
| 0   | Automatic TTL, DNS only                 |
| 1   | Automatic TTL, DNS and HTTP proxy (CDN) |
| 2+  | Set as TTL, DNS only                    |

Pull requests welcome :-)


## Running cfzone

cfzone need two environment variables:

- `CF_API_KEY` - Your API key from [Cloudflare](https://support.cloudflare.com/hc/en-us/articles/200167836-Where-do-I-find-my-Cloudflare-API-key-)
- `CF_API_EMAIL` - Your Cloudflare email address.

Run cfzone as with the following command:
`cfzone <zonefile> [-leaveunknown] [-yes]`

Available optional flags:

| Flag            | Description                                                 |
|-----------------|-------------------------------------------------------------|
| `-leaveunknown` | Don't delete unknown records                                |
| `-yes`          | will cause cfzone to continue syncing without confirmation. |

## Building

You'll need a working [Go environment](https://golang.org/doc/install) to build
cfzone.

`go get github.com/cego/cfzone` should retrieve the source code, build it and
place the binary in `$GOPATH/bin/cfzone`.
