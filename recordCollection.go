package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/miekg/dns"
)

const cfAutoTTL = 0  // This is the literal TTL in Cloudflare auto-TTL records
const cfCacheTTL = 1 // This is the literal TTL in Cloudflare CDN records

type (
	recordCollection []cloudflare.DNSRecord

	// FilterFunc is used for finding records in a recordCollection. The
	// function must return true if there is a hit, false otherwise.
	FilterFunc func(a cloudflare.DNSRecord, b cloudflare.DNSRecord) bool
)

// Clone will make a copy of a recordCollection.
func (c recordCollection) Clone() recordCollection {
	result := recordCollection{}

	result = append(result, c...)

	return result
}

// Remove will remove the n'th element of c.
func (c *recordCollection) Remove(n int) {
	*c = append((*c)[:n], (*c)[n+1:]...)
}

// Find will search for needle in a recordCollection.
func (c recordCollection) Find(needle cloudflare.DNSRecord, match FilterFunc) (int, *cloudflare.DNSRecord) {
	for i, r := range c {
		if match(r, needle) {
			return i, &r
		}
	}

	return -1, nil
}

// Difference will find all the elements in c not present in remote [c \ remote].
func (c recordCollection) Difference(remote recordCollection, match FilterFunc) recordCollection {
	result := recordCollection{}
	B := remote.Clone()

	for _, r := range c {
		n, _ := B.Find(r, match)

		if n < 0 {
			result = append(result, r)
		} else {
			B.Remove(n)
		}
	}

	return result
}

// Intersect will find the intersection between c and remote [c ∩ remote] with
// the caveat that the ID from c will be used in the result - while all other
// properties will be copied from remote.
// If multiple record from a collection matches, only one will be present in
// the returned collection.
func (c recordCollection) Intersect(remote recordCollection, match FilterFunc) recordCollection {
	// Clone the inputs - we do this to be able to remove from these
	// collections when a match is found.
	A := c.Clone()
	B := remote.Clone()
	intersect := recordCollection{}

	for i := 0; i < len(A); i++ {
		found, hit := B.Find(A[i], match)
		if found >= 0 {
			// We do this trickery to keep the ID from the left part.
			record := *hit
			record.ID = A[i].ID

			intersect = append(intersect, record)

			// To make sure we're not double-spending we remove the found
			// record from both inputs.
			A.Remove(i)
			B.Remove(found)

			// Rewind the index to compensate for the item we just removed.
			i--
		}
	}

	return intersect
}

// Fprint will output a textual representation of a recordCollection resembling
// the BIND zone file format.
func (c recordCollection) Fprint(w io.Writer) {
	maxName := 0
	for _, r := range c {
		if len(r.Name) > maxName {
			maxName = len(r.Name)
		}
	}

	for _, r := range c {
		name := r.Name + "." + strings.Repeat(" ", maxName-len(r.Name))

		proxied := ""
		if r.Proxied {
			proxied = " ; PROXIED"
		}

		fmt.Fprintf(w, "%s %d %-8s %s%s\n", name, r.TTL, "IN "+r.Type, r.Content, proxied)
	}
}

// parseZone will parse a BIND style zone file and return the zone name and
// a recordCollection.
func parseZone(r io.Reader) (string, recordCollection, error) {
	return parseZoneWithOrigin(r, "")
}

func parseZoneWithOrigin(r io.Reader, origin string) (string, recordCollection, error) {
	return parseZoneWithOriginAndTTLs(r, origin, cfAutoTTL, cfCacheTTL)
}
func parseZoneWithOriginAndTTLs(r io.Reader, origin string, autoTTL, cacheTTL int) (string, recordCollection, error) {
	var zoneName string
	records := recordCollection{}

	p := dns.NewZoneParser(r, "", "")

	for rr, ok := p.Next(); ok; rr, ok = p.Next() {
		// Search for zonename while we're at it.
		soa, found := rr.(*dns.SOA)
		if found {
			zoneName = strings.Trim(soa.Header().Name, ".")
		}

		r, err := newRecord(rr, autoTTL, cacheTTL)
		if err != nil {
			return "", recordCollection{}, err
		}

		if r != nil {
			records = append(records, *r)
		}
	}

	if err := p.Err(); err != nil {
		return "", recordCollection{}, err
	}

	if zoneName == "" {
		return "", recordCollection{}, errors.New("Zone name not found")
	}

	return zoneName, records, nil
}

// newRecord will instantiate a new cloudflare-compatible DNS record based on
// a token from miekg/dns..
// If the TTL has a value of 1 Proxied will be set to true in the resulting
// DNSRecord mimicking Cloudflare internal TTL's.
// A TTL of 0 will result in "automatic" TTL.
func newRecord(in dns.RR, autoTTL, cacheTTL int) (*cloudflare.DNSRecord, error) {
	record := &cloudflare.DNSRecord{
		Name: strings.Trim(in.Header().Name, "."),
		TTL:  int(in.Header().Ttl),
	}

	if record.TTL == cacheTTL {
		record.Proxied = true
		record.TTL = cfCacheTTL
	} else if record.TTL == autoTTL {
		record.TTL = cfAutoTTL
	} else if record.TTL < 1 {
		record.TTL = 1
	}

	switch v := in.(type) {
	case *dns.A:
		record.Content = v.A.String()
		record.Type = "A"

		return record, nil

	case *dns.AAAA:
		record.Content = v.AAAA.String()
		record.Type = "AAAA"

		return record, nil

	case *dns.CNAME:
		record.Content = v.Target
		record.Type = "CNAME"

		// CloudFlare does not use the "FQDN-dot". We remove it.
		if strings.HasSuffix(record.Content, ".") {
			record.Content = record.Content[:len(record.Content)-1]
		}

		return record, nil

	case *dns.MX:
		record.Content = strings.Trim(v.Mx, ".")
		record.Priority = int(v.Preference)
		record.Type = "MX"

		return record, nil

	case *dns.TXT:
		record.Type = "TXT"
		if len(v.Txt) > 0 {
			record.Content = strings.Join(v.Txt, "")
		}

		return record, nil

	case *dns.SPF:
		if ignoreSpf {
			// If the user specifically asked, ignore these records rather than raising an error
			return nil, nil
		}

	case *dns.SRV:
		if ignoreSrv {
			// If the user specifically asked, ignore these records rather than raising an error
			return nil, nil
		}

	case *dns.NS, *dns.SOA:
		// We silently ignore NS and SOA because Cloudflare does not allow
		// the user to change nameservers and SOA doesn't make sense.
		return nil, nil
	}

	return nil, fmt.Errorf("record type %T is not supported", in)
}

// FullMatch will do matching between two DNS records while ignoring CF specific
// details.
func FullMatch(a cloudflare.DNSRecord, b cloudflare.DNSRecord) bool {
	if a.Type != b.Type {
		return false
	}

	if a.Name != b.Name {
		return false
	}

	if a.Proxied != b.Proxied {
		return false
	}

	if a.TTL != b.TTL {
		return false
	}

	switch a.Type {
	case "A", "AAAA", "CNAME", "TXT":
		if a.Content == b.Content {
			return true
		}

	case "MX":
		if a.Content == b.Content && a.Priority == b.Priority {
			return true
		}
	}

	return false
}

// Updatable will return true if it makes sense to update (instead of
// add/delete) from a to b or b to a.
func Updatable(a cloudflare.DNSRecord, b cloudflare.DNSRecord) bool {
	if a.Type != b.Type {
		return false
	}

	if a.Name != b.Name {
		return false
	}

	return true
}
