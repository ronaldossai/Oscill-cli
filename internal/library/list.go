package library

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ronaldossai/Oscill-cli/internal/audio"
)

// DirListing is the single-level contents of a directory, split into
// subdirectories and supported audio samples. Hidden entries (dotfiles) are
// excluded.
type DirListing struct {
	Dirs    []string
	Samples []Sample
}

// ListDir reads a single directory level (not recursive) and returns its
// subdirectories and audio samples, both sorted by name.
func ListDir(dir string) (DirListing, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return DirListing{}, err
	}

	var listing DirListing
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		if e.IsDir() {
			listing.Dirs = append(listing.Dirs, name)
			continue
		}

		format, ok := extensionFormats[strings.ToLower(filepath.Ext(name))]
		if !ok {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		path := filepath.Join(dir, name)
		sample := Sample{
			Path:   path,
			Name:   name,
			Format: format,
			Size:   info.Size(),
		}
		if meta, err := audio.Probe(path, string(format)); err == nil {
			sample.Duration = meta.Duration
			sample.SampleRate = meta.SampleRate
			sample.Channels = meta.Channels
			sample.BitDepth = meta.BitDepth
		}
		listing.Samples = append(listing.Samples, sample)
	}

	sort.Strings(listing.Dirs)
	sort.Slice(listing.Samples, func(i, j int) bool {
		return listing.Samples[i].Name < listing.Samples[j].Name
	})

	return listing, nil
}
