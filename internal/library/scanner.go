package library

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/ronaldossai/Oscill-cli/internal/audio"
)

var extensionFormats = map[string]Format{
	".wav":  FormatWAV,
	".mp3":  FormatMP3,
	".flac": FormatFLAC,
	".ogg":  FormatOGG,
}

// Scan walks root recursively and returns every supported audio file found,
// sorted by path. Files that cannot be read are skipped rather than aborting
// the whole scan.
func Scan(root string) ([]Sample, error) {
	var samples []Sample

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip unreadable entries (e.g. permission errors) but keep scanning.
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}

		format, ok := extensionFormats[strings.ToLower(filepath.Ext(path))]
		if !ok {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		sample := Sample{
			Path:   path,
			Name:   d.Name(),
			Format: format,
			Size:   info.Size(),
		}

		if meta, err := audio.Probe(path, string(format)); err == nil {
			sample.Duration = meta.Duration
			sample.SampleRate = meta.SampleRate
			sample.Channels = meta.Channels
			sample.BitDepth = meta.BitDepth
		}

		samples = append(samples, sample)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return samples, nil
}
