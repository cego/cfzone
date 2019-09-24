# cfzone

This is a utility to update a cloudflare zone based on a bind-style [zone file](https://en.wikipedia.org/wiki/Zone_file).


## Limitations

Only `A`, `AAAA`, `CNAME`, `MX`, and `TXT` records are supported.

Cloudflare supported record types `LOC`, `NS`, `SRV`, `SPF` and `CAA` are not
currently supported.

Flags exist to skip `SRV` and `SPF` types.

Cloudflare supports (at least) two modes not easily representable in a BIND
zone. To support these features a few magic TTL values are used.

| TTL           | Status                                  |
|---------------|-----------------------------------------|
| -autottl  (0) | Automatic TTL, DNS only                 |
| -cachettl (1) | Automatic TTL, DNS and HTTP proxy (CDN) |
| Other values  | Set as TTL, DNS only                    |

Pull requests welcome :-)


## Running cfzone

cfzone need two environment variables:

- `CF_API_KEY` - Your API key from [Cloudflare](https://support.cloudflare.com/hc/en-us/articles/200167836-Where-do-I-find-my-Cloudflare-API-key-)
- `CF_API_EMAIL` - Your Cloudflare email address.

Run cfzone as with the following command:
`cfzone [-leaveunknown] [-yes] <zonefile>`

Available optional flags:

| Flag              | Description                                                        |
|-------------------|--------------------------------------------------------------------|
| `-leaveunknown`   | Don't delete unknown records                                       |
| `-yes`            | will cause cfzone to continue syncing without confirmation.        |
| `-autottl <int>`  | Specify the TTL to interpret as 'Auto' for Cloudflare (default 0)  |
| `-cachettl <int>` | Specify the TTL to interpret as 'Cache' for Cloufdlare (default 1) |
| `-ignorespf`      | Skip SPF records in the BIND zone file rather than erroring        |
| `-ignoresrv`      | Skip SRV records in the BIND zone file rather than erroring        |
| `-origin`         | Specify zone origin to resolve @ at the top level

## Building

You'll need a working [Go environment](https://golang.org/doc/install) to build
cfzone.

`go get github.com/third-light/cfzone` should retrieve the source code, build it and
place the binary in `$GOPATH/bin/cfzone`.
