package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

func init() {
	exit = panicExit
	stderr = ioutil.Discard
}

func panicExit(code int) {
	panic(code)
}

func expectExit(t *testing.T, code int) {
	if code < 0 {
		got := recover()
		if got != nil {
			t.Fatalf("exit unexpectedly called with value %v", got)
		}

		return
	}

	got := recover()
	if got != code {
		t.Fatalf("exited with wrong error %d, expected %d", got, code)
	}
}

func TestMissingKey(t *testing.T) {
	defer expectExit(t, 1)

	apiKey = ""
	apiEmail = ""

	main()
}

func TestMissingArgument(t *testing.T) {
	defer expectExit(t, 1)

	apiKey = "nonempty"
	apiEmail = "nonempty"

	os.Args = []string{"./test"}
	main()
}

func TestNonexisting(t *testing.T) {
	defer expectExit(t, 1)

	apiKey = "nonempty"
	apiEmail = "nonempty"

	os.Args = []string{"./test", "/non/existing/zone"}
	main()
}

func TestBrokenZone(t *testing.T) {
	defer expectExit(t, 1)

	apiKey = "nonempty"
	apiEmail = "nonempty"

	os.Args = []string{"./test", "/dev/null"}
	main()
}

func TestYesNo(t *testing.T) {
	cases := []struct {
		line     string
		expected bool
	}{
		{"y\n", true},
		{"Y\n", true},
		{"n\n", false},
		{"N\n", false},
		{"\n", false},
		{"-\n", false},
		{"\ny\n", false},
		{"", false},
		{" ", false},
	}

	for i, in := range cases {
		b := bytes.NewBufferString(in.line)
		result := yesNo(b)
		if result != in.expected {
			t.Errorf("%d: yesNo() returned wrong result for '%s', got %v, expected %v", i, in.line, result, in.expected)
		}
	}
}
