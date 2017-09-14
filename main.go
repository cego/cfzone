package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudflare/cloudflare-go"
)

var (
	// These can be overridden for testing.
	exit   = os.Exit
	stdout = io.Writer(os.Stdout)
	stdin  = io.Reader(os.Stdin)
	stderr = io.Writer(os.Stderr)

	// yes can be set to true to disable the confirmation dialog and sync
	// without asking the user. Will be set to true by the "-yes" flag.
	yes = false
)

var (
	apiKey   = os.Getenv("CF_API_KEY")
	apiEmail = os.Getenv("CF_API_EMAIL")
)

// parseArguments tries to pass the arguments in args. For most uses it would
// make sense to simple pass os.Args. The function will call exit(1) on any
// error. It will return the first ńon-flag argument.
func parseArguments(args []string) string {
	// We do our own flagset to be able to test arguments.
	flagset := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flagset.SetOutput(stderr)
	flagset.BoolVar(&yes, "yes", false, "Don't ask before syncing")
	err := flagset.Parse(args[1:])
	if err != nil {
		flagset.PrintDefaults()
		exit(1)
	}

	if flagset.NArg() < 1 {
		fmt.Fprintf(stderr, "Too few arguments\n")
		exit(1)
	}

	return flagset.Arg(0)
}

func main() {
	path := parseArguments(os.Args)

	if apiKey == "" || apiEmail == "" {
		fmt.Fprintf(stderr, "Please set CF_API_KEY and CF_API_EMAIL environment variables\n")
		exit(1)
	}

	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(stderr, "Error opening '%s': %s\n", path, err.Error())
		exit(1)
	}

	zoneName, localRecords, err := parseZone(f)
	if err != nil {
		fmt.Fprintf(stderr, "Error reading '%s': %s\n", path, err.Error())
		exit(1)
	}

	api, err := cloudflare.New(apiKey, apiEmail)
	if err != nil {
		fmt.Fprintf(stderr, "Error contacting Cloudflare: %s\n", err.Error())
		exit(1)
	}

	id, err := api.ZoneIDByName(zoneName)
	if err != nil {
		fmt.Fprintf(stderr, "Can't get zone ID for '%s': %s\n", zoneName, err.Error())
		exit(1)
	}

	records, err := api.DNSRecords(id, cloudflare.DNSRecord{})
	if err != nil {
		fmt.Fprintf(stderr, "Can't get zone records for '%s': %s\n", id, err.Error())
		exit(1)
	}

	adds, deletes := localRecords.Diff(records)

	numChanges := len(adds) + len(deletes)

	if numChanges > 0 && !yes {
		if len(deletes) > 0 {
			fmt.Fprintf(stdout, "Records to delete:\n")
			deletes.Fprint(stdout)
			fmt.Printf("\n")
		}

		if len(adds) > 0 {
			fmt.Fprintf(stdout, "Records to add:\n")
			adds.Fprint(stdout)
			fmt.Printf("\n")
		}

		fmt.Fprintf(stdout, "Summary:\n")
		fmt.Fprintf(stdout, "Records to delete: %d\n", len(deletes))
		fmt.Fprintf(stdout, "Records to add: %d\n", len(adds))
		fmt.Fprintf(stdout, "Unchanged records: %d\n", len(records)-len(deletes))
		fmt.Fprintf(stdout, "%d change(s). Continue (y/N)? ", numChanges)

		if !yesNo(stdin) {
			fmt.Fprintf(stdout, "Aborting...\n")
			exit(0)
		}
	}

	for _, r := range deletes {
		err = api.DeleteDNSRecord(id, r.ID)
		if err != nil {
			fmt.Fprintf(stderr, "Failed to delete record %+v: %s\n", r, err.Error())
			exit(1)
		}
	}

	for _, r := range adds {
		_, err = api.CreateDNSRecord(id, r)
		if err != nil {
			fmt.Fprintf(stderr, "Failed to add record %+v: %s\n", r, err.Error())
			exit(1)
		}
	}
}

// yesNo will return true if the user entered Y or y + enter. False in all
// other cases.
func yesNo(r io.Reader) bool {
	line, _, _ := bufio.NewReader(r).ReadLine()

	if strings.ToLower(string(line)) == "y" {
		return true
	}

	return false
}
