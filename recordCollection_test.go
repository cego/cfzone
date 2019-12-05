package main

import (
	"bufio"
	"bytes"
	"reflect"
	"strings"
	"testing"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

func TestClone(t *testing.T) {
	a := recordCollection{}
	b := a.Clone()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Clone() failed to clone an empty recordCollection")
	}

	a = recordCollection{
		cloudflare.DNSRecord{Type: "A", Name: "a1", Content: "127.0.0.1"},
		cloudflare.DNSRecord{Type: "A", Name: "a2", Content: "127.0.0.2"},
		cloudflare.DNSRecord{Type: "A", Name: "a3", Content: "127.0.0.3"},
		cloudflare.DNSRecord{Type: "A", Name: "a4", Content: "127.0.0.10"},
		cloudflare.DNSRecord{Type: "A", Name: "a4", Content: "127.0.0.11"},
		cloudflare.DNSRecord{Type: "A", Name: "a4", Content: "127.0.0.12"},
		cloudflare.DNSRecord{Type: "A", Name: "a4", Content: "127.0.0.13"},
		cloudflare.DNSRecord{Type: "AAAA", Name: "a1", Content: "::1"},
		cloudflare.DNSRecord{Type: "MX", Name: "@", Content: "mail", Priority: 10},
	}
	b = a.Clone()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Clone() failed to clone a recordCollection")
	}
}

func TestRemove(t *testing.T) {
	in := recordCollection{
		cloudflare.DNSRecord{Type: "A", Name: "a1", Content: "127.0.0.1", TTL: 100},
		cloudflare.DNSRecord{Type: "A", Name: "a2", Content: "127.0.0.2", TTL: 200},
		cloudflare.DNSRecord{Type: "A", Name: "a3", Content: "127.0.0.3", TTL: 300},
		cloudflare.DNSRecord{Type: "A", Name: "a4", Content: "127.0.0.4", TTL: 400},
	}

	a := in.Clone()
	a.Remove(1)
	if !reflect.DeepEqual(a, recordCollection{
		cloudflare.DNSRecord{Type: "A", Name: "a1", Content: "127.0.0.1", TTL: 100},
		cloudflare.DNSRecord{Type: "A", Name: "a3", Content: "127.0.0.3", TTL: 300},
		cloudflare.DNSRecord{Type: "A", Name: "a4", Content: "127.0.0.4", TTL: 400},
	}) {
		t.Errorf("Remove() did not return expected result")
	}

	a2 := in.Clone()
	a2.Remove(1)
	if !reflect.DeepEqual(a2, recordCollection{
		cloudflare.DNSRecord{Type: "A", Name: "a1", Content: "127.0.0.1", TTL: 100},
		cloudflare.DNSRecord{Type: "A", Name: "a3", Content: "127.0.0.3", TTL: 300},
		cloudflare.DNSRecord{Type: "A", Name: "a4", Content: "127.0.0.4", TTL: 400},
	}) {
		t.Errorf("Remove() did not return expected result")
	}
}

func TestFindEmpty(t *testing.T) {
	c := recordCollection{}

	n, r := c.Find(cloudflare.DNSRecord{}, FullMatch)
	if n >= 0 {
		t.Errorf("Find() returned a non-negative value from an empty collection")
	}

	if r != nil {
		t.Errorf("Find() returned a record from an empty collection")
	}
}

func TestFullMatch(t *testing.T) {
	cases := []struct {
		a        cloudflare.DNSRecord
		b        cloudflare.DNSRecord
		expected bool
	}{
		{cloudflare.DNSRecord{Type: "A"}, cloudflare.DNSRecord{Type: "A"}, true},
		{cloudflare.DNSRecord{Type: "A", Name: "a"}, cloudflare.DNSRecord{Type: "A", Name: "a"}, true},
		{cloudflare.DNSRecord{Type: "A", Name: "a"}, cloudflare.DNSRecord{Type: "A", Name: "ab"}, false},
		{cloudflare.DNSRecord{Type: "A", Name: "a", TTL: 0}, cloudflare.DNSRecord{Type: "A", Name: "a", TTL: 0}, true},
		{cloudflare.DNSRecord{Type: "A", Name: "a", TTL: 0}, cloudflare.DNSRecord{Type: "A", Name: "a", TTL: 1}, false},
		{cloudflare.DNSRecord{Type: "A", Name: "a", Proxied: true}, cloudflare.DNSRecord{Type: "A", Name: "a", Proxied: true}, true},
		{cloudflare.DNSRecord{Type: "A", Name: "a", Proxied: true}, cloudflare.DNSRecord{Type: "A", Name: "a"}, false},
		{cloudflare.DNSRecord{Type: "A", Name: "a", TTL: 0}, cloudflare.DNSRecord{Type: "A", Name: "a", TTL: 3600}, false},
	}

	for i, in := range cases {
		result := FullMatch(in.a, in.b)

		if result != in.expected {
			t.Errorf("%d: match() Returned unexpected result for %v, %v: %v (expected %v)", i, in.a, in.b, result, in.expected)
		}
	}
}

func TestUpdatable(t *testing.T) {
	cases := []struct {
		a        cloudflare.DNSRecord
		b        cloudflare.DNSRecord
		expected bool
	}{
		{cloudflare.DNSRecord{Type: "A"}, cloudflare.DNSRecord{Type: "A"}, true},
		{cloudflare.DNSRecord{Type: "A", Name: "a"}, cloudflare.DNSRecord{Type: "A", Name: "a"}, true},
		{cloudflare.DNSRecord{Type: "A", Name: "a"}, cloudflare.DNSRecord{Type: "A", Name: "ab"}, false},
		{cloudflare.DNSRecord{Type: "A", Name: "a", TTL: 0}, cloudflare.DNSRecord{Type: "A", Name: "a", TTL: 0}, true},
		{cloudflare.DNSRecord{Type: "A", Name: "a", TTL: 0}, cloudflare.DNSRecord{Type: "A", Name: "a", TTL: 1}, true},
		{cloudflare.DNSRecord{Type: "A", Name: "a", Proxied: true}, cloudflare.DNSRecord{Type: "A", Name: "a", Proxied: true}, true},
		{cloudflare.DNSRecord{Type: "A", Name: "a", Proxied: true}, cloudflare.DNSRecord{Type: "A", Name: "a"}, true},
		{cloudflare.DNSRecord{Type: "A", Name: "a", TTL: 0}, cloudflare.DNSRecord{Type: "A", Name: "a", TTL: 3600}, true},
		{cloudflare.DNSRecord{Type: "CNAME", Name: "a", TTL: 0}, cloudflare.DNSRecord{Type: "A", Name: "a", TTL: 3600}, false},
	}

	for i, in := range cases {
		result := Updatable(in.a, in.b)

		if result != in.expected {
			t.Errorf("%d: match() Returned unexpected result for %v, %v: %v (expected %v)", i, in.a, in.b, result, in.expected)
		}
	}
}

func TestFind(t *testing.T) {
	c := recordCollection{
		cloudflare.DNSRecord{Type: "A", Name: "a1", Content: "127.0.0.1"},
		cloudflare.DNSRecord{Type: "A", Name: "a2", Content: "127.0.0.2"},
		cloudflare.DNSRecord{Type: "A", Name: "a3", Content: "127.0.0.3"},
		cloudflare.DNSRecord{Type: "A", Name: "a4", Content: "127.0.0.10"},
		cloudflare.DNSRecord{Type: "A", Name: "a4", Content: "127.0.0.11"},
		cloudflare.DNSRecord{Type: "A", Name: "a4", Content: "127.0.0.12"},
		cloudflare.DNSRecord{Type: "A", Name: "a4", Content: "127.0.0.13"},
		cloudflare.DNSRecord{Type: "AAAA", Name: "a1", Content: "::1"},
		cloudflare.DNSRecord{Type: "MX", Name: "@", Content: "mail", Priority: 10},
	}

	cases := []struct {
		needle cloudflare.DNSRecord
		n      int
		r      *cloudflare.DNSRecord
	}{
		{cloudflare.DNSRecord{Type: "A", Name: "a1"}, -1, nil},
		{cloudflare.DNSRecord{Type: "A", Name: "a1", Content: "127.0.0.2"}, -1, nil},
		{cloudflare.DNSRecord{Type: "A", Name: "a1", Content: "127.0.0.1"}, 0, &cloudflare.DNSRecord{}},
		{cloudflare.DNSRecord{Type: "A", Name: "a1", Content: "127.0.0.1"}, 0, &cloudflare.DNSRecord{}},
		{cloudflare.DNSRecord{Type: "A", Name: "a4", Content: "127.0.0.12"}, 5, &cloudflare.DNSRecord{}},
		{cloudflare.DNSRecord{Type: "MX", Name: "a1", Content: "127.0.0.12"}, -1, nil},
		{cloudflare.DNSRecord{Type: "MX", Name: "@", Content: "127.0.0.12"}, -1, nil},
		{cloudflare.DNSRecord{Type: "MX", Name: "@", Content: "::1"}, -1, nil},
		{cloudflare.DNSRecord{Type: "MX", Name: "@", Content: "mail", Priority: 10}, 8, &cloudflare.DNSRecord{}},
	}

	for i, in := range cases {
		n, r := c.Find(in.needle, FullMatch)
		if n != in.n {
			t.Errorf("%d: Find() Returned unexpected n: %d (expected %d)", i, n, in.n)
		}

		if (r == nil) != (in.r == nil) {
			t.Errorf("%d: Find() returned unexpected pointer: %p (expected %p)", i, r, in.r)
		}
	}
}

func TestDifference(t *testing.T) {
	empty := recordCollection{}
	a1 := cloudflare.DNSRecord{Type: "A", Name: "test1", Content: "127.0.0.1"}
	a2 := cloudflare.DNSRecord{Type: "A", Name: "test1", Content: "127.0.0.2"}
	aaaa1 := cloudflare.DNSRecord{Type: "AAAA", Name: "test1", Content: "::1"}
	cases := []struct {
		a        recordCollection
		b        recordCollection
		expected recordCollection
	}{
		{empty, empty, empty},
		{empty, recordCollection{a1}, empty},
		{recordCollection{a1}, empty, recordCollection{a1}},
		{empty, recordCollection{aaaa1}, empty},
		{recordCollection{aaaa1}, empty, recordCollection{aaaa1}},
		{recordCollection{aaaa1}, recordCollection{a1}, recordCollection{aaaa1}},
		{recordCollection{a1, a2}, recordCollection{a1}, recordCollection{a2}},
		{recordCollection{a1, a2, a2}, recordCollection{a1}, recordCollection{a2, a2}},
	}

	for i, in := range cases {
		result := in.a.Difference(in.b, FullMatch)
		if !reflect.DeepEqual(in.expected, result) {
			t.Errorf("%d: aOnly != in.aOnly, Got %+v, expcted %+v", i, result, in.expected)
		}
	}
}

func TestIntersect(t *testing.T) {
	empty := recordCollection{}
	a1 := cloudflare.DNSRecord{Type: "A", Name: "test1", Content: "127.0.0.1"}
	aaaa1 := cloudflare.DNSRecord{Type: "AAAA", Name: "test1", Content: "::1"}
	cases := []struct {
		a        recordCollection
		b        recordCollection
		expected recordCollection
	}{
		{empty, empty, empty},
		{empty, recordCollection{a1}, empty},
		{recordCollection{a1}, empty, empty},
		{recordCollection{a1}, recordCollection{a1}, recordCollection{a1}},
		{empty, recordCollection{aaaa1}, empty},
	}

	for i, in := range cases {
		result := in.a.Intersect(in.b, FullMatch)
		if !reflect.DeepEqual(in.expected, result) {
			t.Errorf("%d: aOnly != in.aOnly, Got %+v, expcted %+v", i, result, in.expected)
		}
	}
}

func TestFprint(t *testing.T) {
	c := recordCollection{
		cloudflare.DNSRecord{Name: "a1", TTL: 0, Type: "A", Content: "127.0.0.1"},
		cloudflare.DNSRecord{Name: "a2", TTL: 1, Type: "A", Content: "127.0.0.2", Proxied: true},
		cloudflare.DNSRecord{Name: "aaaa1", TTL: 0, Type: "AAAA", Content: "::1"},
	}
	expected := `a1.    0 IN A     127.0.0.1
a2.    1 IN A     127.0.0.2 ; PROXIED
aaaa1. 0 IN AAAA  ::1
`

	var b bytes.Buffer
	w := bufio.NewWriter(&b)

	c.Fprint(w)
	w.Flush()

	if b.String() != expected {
		t.Fatalf("Print() returned wrong output, got [%s], expected [%s]", b.String(), expected)
	}
}

func TestParseZone(t *testing.T) {
	zone := `
$ORIGIN example.com.
$TTL 3600

@    86400    IN SOA ns1.example.com. hostmaster.example.com. (
          2015071700 ; serial
          86400 ; refresh
          7200 ; retry
          604800 ; expire
          86400 ; minimum
          )

@     1800     IN NS    ns1.example.com.
@     1800     IN NS    ns2.example.com.
@     1800     IN NS    ns3.example.com.
@     1800     IN MX    10 mail10.example.com.
test1 1800 IN A 127.0.0.1
test2 1800 IN CNAME test1
test3 1800 IN AAAA ::1
test4 1 IN A 127.0.0.4
@     1800 IN TXT "v=spf1 include:spf.example.com -all"
`

	parsed := recordCollection{
		cloudflare.DNSRecord{
			Type:     "MX",
			Priority: 10,
			Name:     "example.com",
			Content:  "mail10.example.com",
			TTL:      1800,
		},
		cloudflare.DNSRecord{
			Type:    "A",
			Name:    "test1.example.com",
			Content: "127.0.0.1",
			TTL:     1800,
		},
		cloudflare.DNSRecord{
			Type:    "CNAME",
			Name:    "test2.example.com",
			Content: "test1.example.com",
			TTL:     1800,
		},
		cloudflare.DNSRecord{
			Type:    "AAAA",
			Name:    "test3.example.com",
			Content: "::1",
			TTL:     1800,
		},
		cloudflare.DNSRecord{
			Type:    "A",
			Name:    "test4.example.com",
			Content: "127.0.0.4",
			TTL:     1,
			Proxied: true,
		},
		cloudflare.DNSRecord{
			Type:    "TXT",
			Name:    "example.com",
			Content: "v=spf1 include:spf.example.com -all",
			TTL:     1800,
		},
	}

	cases := []struct {
		zone         string
		expectedName string
		expected     recordCollection
		err          bool
	}{
		{"broken zone", "", recordCollection{}, true},
		{zone, "example.com", parsed, false},
	}

	for i, in := range cases {
		r := strings.NewReader(in.zone)
		zoneName, records, err := parseZone(r)
		if in.err && err == nil {
			t.Fatalf("%d: parseZone() failed to error on [%s]", i, in.zone)
		}

		if !in.err && err != nil {
			t.Fatalf("%d: parseZone() returned error on [%s]: %s", i, in.zone, err.Error())
		}

		if zoneName != in.expectedName {
			t.Errorf("%d: parseZone() rturned wrong zone name for [%s], got %s, expected %s", i, in.zone, zoneName, in.expectedName)
		}

		if !reflect.DeepEqual(in.expected, records) {
			t.Errorf("%d: parseZone() returned wrong zone for [%s], got:\n%s, expected:\n%s", i, in.zone, zoneString(records), zoneString(in.expected))
		}
	}
}

func TestParseZoneFail(t *testing.T) {
	cases := []string{`$ORIGIN example.com.

@    86400    IN SOA ns1.example.com. hostmaster.example.com. (
          2015071700
          86400
          7200
          604800
          86400
)
test1 1800 IN A 127.0.0.1
loc1 IN LOC 57 2 59.173 N 9 56 42.07 E 0m 10m 100m 10m
`, `@    86400    IN SOA ns1.example.com. hostmaster.example.com. (
	  2015071700
	  86400
	  7200
	  604800
	  86400
)
test2 1800 IN A 127.0.0.2
`,
	}

	for i, in := range cases {
		r := strings.NewReader(in)
		zoneName, records, err := parseZone(r)

		if zoneName != "" {
			t.Errorf("%d parseZone() returned a zonename for a broken zone: %s", i, zoneName)
		}

		if len(records) > 0 {
			t.Errorf("%d: parseZone() returned record for a broken zone", i)
		}

		if err == nil {
			t.Errorf("%d: parseZone() failed to err on broken zone", i)
		}

		parseZone(r)
	}
}

func zoneString(c recordCollection) string {
	var b bytes.Buffer

	w := bufio.NewWriter(&b)
	c.Fprint(w)
	w.Flush()

	return b.String()
}
