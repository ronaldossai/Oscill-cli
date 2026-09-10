package ui

import "github.com/ronaldossai/Oscill-cli/internal/library"

// dirItem is a subdirectory entry shown in the folders pane.
type dirItem string

func (d dirItem) FilterValue() string { return string(d) }

// sampleItem is an audio file entry shown in the samples pane. Display is
// what the delegate renders: just the filename when browsing a single
// directory, or a library-relative path when showing search results that
// may span several directories.
type sampleItem struct {
	Sample  library.Sample
	Display string
}

func (s sampleItem) FilterValue() string { return s.Display }
