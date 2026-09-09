// Command oscill is a terminal toolkit for browsing and analysing audio
// sample libraries.
package main

import (
	"fmt"
	"os"
	"text/tabwriter"

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
  oscill help               Show this message.`)
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

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tFORMAT\tSIZE\tDURATION\tSAMPLE RATE\tCHANNELS\tBIT DEPTH")
	for _, s := range samples {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			s.Name,
			s.Format,
			format.HumanSize(s.Size),
			format.OptionalDuration(s.Duration, s.HasAudioMetadata()),
			format.OptionalInt(s.SampleRate, s.HasAudioMetadata(), "Hz"),
			format.OptionalInt(s.Channels, s.HasAudioMetadata(), ""),
			format.OptionalInt(s.BitDepth, s.HasAudioMetadata(), "-bit"),
		)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	fmt.Printf("\n%d sample(s) found\n", len(samples))
	return nil
}
