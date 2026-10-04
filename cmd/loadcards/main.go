// Command loadcards validates a RON card collection before registration.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/benprew/mage-go/pkg/mage/dsl/ron"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loadcards", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dir := flags.String("dir", "cards/mtgish", "directory containing RON card files")
	jsonOutput := flags.Bool("json", false, "print diagnostics as JSON")
	inventory := flags.Bool("inventory", false, "inspect every expression without claiming rule support")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *inventory {
		report, err := ron.InspectFS(os.DirFS(*dir), "*.ron")
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := json.NewEncoder(stdout).Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		// Inspection must keep the same collection rejection as strict loading.
		if len(report.Issues) != 0 {
			return 1
		}
		if _, err := ron.LoadFS(os.DirFS(*dir), "*.ron"); err != nil {
			return 1
		}
		return 0
	}
	registry, err := ron.LoadFS(os.DirFS(*dir), "*.ron")
	if err != nil {
		if *jsonOutput {
			if report, ok := errors.AsType[*ron.LoadError](err); ok {
				if encodeErr := json.NewEncoder(stdout).Encode(report); encodeErr != nil {
					fmt.Fprintln(stderr, encodeErr)
				}
			} else {
				fmt.Fprintln(stderr, err)
			}
		} else {
			fmt.Fprintln(stderr, err)
		}
		return 1
	}
	names := registry.RegisteredCardNames()
	sort.Strings(names)
	if *jsonOutput {
		if err := json.NewEncoder(stdout).Encode(names); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else {
		fmt.Fprintf(stdout, "RON collection compiled: %d unique cards\n", len(names))
	}
	return 0
}
