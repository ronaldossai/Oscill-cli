// Command oscill is a terminal toolkit for browsing and analysing audio
// sample libraries.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ronaldossai/Oscill-cli/internal/format"
	"github.com/ronaldossai/Oscill-cli/internal/library"
	"github.com/ronaldossai/Oscill-cli/internal/ui"
)

func main() {
	if len(os.Args) < 2 {
		if err := runBrowse(nil); err != nil {
			fmt.Fprintln(os.Stderr, "oscill:", err)
			os.Exit(1)
		}
		return
	}

	switch os.Args[1] {
	case "browse":
		if err := runBrowse(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "oscill:", err)
			os.Exit(1)
		}
	case "scan":
		if err := runScan(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "oscill:", err)
			os.Exit(1)
		}
	case "find":
		if err := runFind(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "oscill:", err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		printUsage()
	default:
		// Not a known subcommand — treat it as "oscill [directory]".
		if err := runBrowse(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "oscill:", err)
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Oscill — a terminal toolkit for music sample libraries.

Usage:
  oscill [directory]        Launch the interactive sample browser (default: current directory).
  oscill browse [directory] Same as above.
  oscill scan [directory]   Recursively discover samples and print their metadata as a table.
  oscill find [directory] [flags]
                            Recursively search a library and print matches as a table.
                            Run "oscill find -h" for the available flags.
  oscill help               Show this message.

Inside the browser: press "/" to search live across the whole library.`)
}

func runBrowse(args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	m, err := ui.NewBrowser(dir)
	if err != nil {
		return fmt.Errorf("opening %s: %w", dir, err)
	}

	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func runScan(args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	samples, err := library.Scan(dir)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", dir, err)
	}

	if len(samples) == 0 {
		fmt.Printf("No supported samples found under %s\n", dir)
		return nil
	}

	printSampleTable(samples, func(s library.Sample) string { return s.Name })
	fmt.Printf("\n%d sample(s) found\n", len(samples))
	return nil
}

func runFind(args []string) error {
	fs := flag.NewFlagSet("find", flag.ContinueOnError)
	name := fs.String("name", "", "filter by substring in filename or path (case-insensitive)")
	formatFlag := fs.String("format", "", "filter by format: wav, mp3, flac, ogg")
	minDur := fs.String("min-duration", "", `minimum duration, e.g. "500ms" or "1.5s"`)
	maxDur := fs.String("max-duration", "", `maximum duration, e.g. "2s"`)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: oscill find [directory] [flags]

Recursively search a sample library and print matching samples as a table.

Flags:`)
		fs.PrintDefaults()
	}
	// flag.Parse stops at the first non-flag argument, so a directory given
	// before its flags (oscill find ~/Samples --name kick, matching scan's
	// and browse's directory-first convention) would otherwise silently
	// disable every flag after it. Separate flags from positional args
	// ourselves so either order works.
	flagArgs, positional := splitFlagsAndPositional(args)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	dir := "."
	if len(positional) > 0 {
		dir = positional[0]
	}

	q := library.Query{Name: *name}
	if *formatFlag != "" {
		q.Format = library.Format(strings.ToLower(*formatFlag))
	}
	if *minDur != "" {
		d, err := time.ParseDuration(*minDur)
		if err != nil {
			return fmt.Errorf("invalid --min-duration %q: %w", *minDur, err)
		}
		q.MinDuration = d
	}
	if *maxDur != "" {
		d, err := time.ParseDuration(*maxDur)
		if err != nil {
			return fmt.Errorf("invalid --max-duration %q: %w", *maxDur, err)
		}
		q.MaxDuration = d
	}

	all, err := library.Scan(dir)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", dir, err)
	}
	matches := library.Filter(all, q)

	if len(matches) == 0 {
		fmt.Printf("No samples matched under %s\n", dir)
		return nil
	}

	printSampleTable(matches, func(s library.Sample) string {
		rel, err := filepath.Rel(dir, s.Path)
		if err != nil {
			return s.Path
		}
		return rel
	})
	fmt.Printf("\n%d sample(s) matched (of %d scanned)\n", len(matches), len(all))
	return nil
}

// splitFlagsAndPositional separates args into flag tokens and positional
// tokens, regardless of how they're interleaved. Every flag find defines
// takes a value, so a "-"-prefixed token not using "=value" form consumes
// the following token as its value too.
func splitFlagsAndPositional(args []string) (flags []string, positional []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
			continue
		}
		flags = append(flags, a)
		if !strings.Contains(a, "=") && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return flags, positional
}

// printSampleTable prints samples as an aligned table. label formats each
// sample's first column — a bare filename for scan, a library-relative path
// for find (whose results can span multiple directories).
func printSampleTable(samples []library.Sample, label func(library.Sample) string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tFORMAT\tSIZE\tDURATION\tSAMPLE RATE\tCHANNELS\tBIT DEPTH")
	for _, s := range samples {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			label(s),
			s.Format,
			format.HumanSize(s.Size),
			format.OptionalDuration(s.Duration, s.HasAudioMetadata()),
			format.OptionalInt(s.SampleRate, s.HasAudioMetadata(), "Hz"),
			format.OptionalInt(s.Channels, s.HasAudioMetadata(), ""),
			format.OptionalInt(s.BitDepth, s.HasAudioMetadata(), "-bit"),
		)
	}
	w.Flush()
}
