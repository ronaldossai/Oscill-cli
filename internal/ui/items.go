package ui

import "github.com/ronaldossai/Oscill-cli/internal/library"

// dirItem is a subdirectory entry shown in the folders pane.
type dirItem string

func (d dirItem) FilterValue() string { return string(d) }

// sampleItem is an audio file entry shown in the samples pane.
type sampleItem library.Sample

func (s sampleItem) FilterValue() string { return s.Name }
