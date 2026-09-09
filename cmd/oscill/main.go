// Command oscill is a terminal toolkit for browsing and analysing audio
// sample libraries.
package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/ronaldossai/Oscill-cli/internal/library"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "scan":
		if err := runScan(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "oscill:", err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "oscill: unknown command %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Oscill — a terminal toolkit for music sample libraries.

Usage:
  oscill scan [directory]   Recursively discover samples and print their metadata.
  oscill help               Show this message.

The interactive browser (oscill / oscill browse) is not built yet.`)
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

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tFORMAT\tSIZE\tDURATION\tSAMPLE RATE\tCHANNELS\tBIT DEPTH")
	for _, s := range samples {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			s.Name,
			s.Format,
			humanSize(s.Size),
			optionalDuration(s.Duration, s.HasAudioMetadata()),
			optionalInt(s.SampleRate, s.HasAudioMetadata(), "Hz"),
			optionalInt(s.Channels, s.HasAudioMetadata(), ""),
			optionalInt(s.BitDepth, s.HasAudioMetadata(), "-bit"),
		)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	fmt.Printf("\n%d sample(s) found\n", len(samples))
	return nil
}

func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func optionalDuration(d time.Duration, known bool) string {
	if !known {
		return "-"
	}
	return d.Round(time.Millisecond).String()
}

func optionalInt(v int, known bool, suffix string) string {
	if !known {
		return "-"
	}
	return fmt.Sprintf("%d%s", v, suffix)
}
