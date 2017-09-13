package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/miekg/dns"
)

type (
	recordCollection []cloudflare.DNSRecord
)

// Find will search for needle in a recordCollection.
func (c recordCollection) Find(needle cloudflare.DNSRecord) (int, *cloudflare.DNSRecord) {
	for i, r := range c {
		if match(r, needle) {
			return i, &r
		}
	}

	return -1, nil
}

// Diff will find the differences between two recordCollections.
func (c recordCollection) Diff(remote recordCollection) (recordCollection, recordCollection) {
	localOnly := recordCollection{}
	remoteOnly := recordCollection{}

	for _, l := range c {
		n, _ := remote.Find(l)

		if n < 0 {
			localOnly = append(localOnly, l)
		}
	}

	for _, r := range remote {
		n, _ := c.Find(r)

		if n < 0 {
			remoteOnly = append(remoteOnly, r)
		}
	}

	return localOnly, remoteOnly
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
	var zoneName string
	records := recordCollection{}

	for t := range dns.ParseZone(r, "", "") {
		if t.Error != nil {
			return "", recordCollection{}, t.Error
		}

		// Search for zonename while we're at it.
		soa, found := t.RR.(*dns.SOA)
		if found {
			zoneName = strings.Trim(soa.Header().Name, ".")
		}

		r := newRecord(t)
		if r != nil {
			records = append(records, *r)
		}
	}

	if zoneName == "" {
		return "", recordCollection{}, errors.New("Zone name not found")
	}

	return zoneName, records, nil
}

// newRecord will instantiate a new cloudflare-compatible DNS record based on
// a token from our zone file parser.
func newRecord(in *dns.Token) *cloudflare.DNSRecord {
	record := &cloudflare.DNSRecord{
		Name: strings.Trim(in.Header().Name, "."),
		TTL:  int(in.Header().Ttl),
	}

	if record.TTL == 1 {
		record.Proxied = true
	}

	switch in.RR.(type) {
	case *dns.A:
		a := in.RR.(*dns.A)
		record.Content = a.A.String()
		record.Type = "A"
		return record

	case *dns.AAAA:
		a := in.RR.(*dns.AAAA)
		record.Content = a.AAAA.String()
		record.Type = "AAAA"
		return record

	case *dns.CNAME:
		cname := in.RR.(*dns.CNAME)
		record.Content = cname.Target
		record.Type = "CNAME"

		// CloudFlare does not use the "FQDN-dot". We remove it.
		if strings.HasSuffix(record.Content, ".") {
			record.Content = record.Content[:len(record.Content)-1]
		}
		return record

	case *dns.MX:
		mx := in.RR.(*dns.MX)
		record.Content = strings.Trim(mx.Mx, ".")
		record.Priority = int(mx.Preference)
		record.Type = "MX"
		return record

	// We ignore these for now.
	case *dns.LOC:
	case *dns.SRV:
	case *dns.SPF:
	case *dns.TXT:
		txt := in.RR.(*dns.TXT)
		if len(txt.Txt) > 0 {
			record.Content = txt.Txt[0]
		}
		record.Type = "TXT"
		return record

	case *dns.NS:
	case *dns.CAA:
	}

	return nil
}

// match will do matching between two DNS records while ignoring CF specific
// details.
func match(a cloudflare.DNSRecord, b cloudflare.DNSRecord) bool {
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
