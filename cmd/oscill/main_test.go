package main

import (
	"reflect"
	"testing"
)

func TestSplitFlagsAndPositional(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantFlags      []string
		wantPositional []string
	}{
		{
			name:           "directory before flags",
			args:           []string{"~/Samples", "--name", "kick"},
			wantFlags:      []string{"--name", "kick"},
			wantPositional: []string{"~/Samples"},
		},
		{
			name:           "flags before directory",
			args:           []string{"--name", "kick", "~/Samples"},
			wantFlags:      []string{"--name", "kick"},
			wantPositional: []string{"~/Samples"},
		},
		{
			name:           "flags only",
			args:           []string{"--format", "wav", "--max-duration", "2s"},
			wantFlags:      []string{"--format", "wav", "--max-duration", "2s"},
			wantPositional: nil,
		},
		{
			name:           "positional only",
			args:           []string{"~/Samples"},
			wantFlags:      nil,
			wantPositional: []string{"~/Samples"},
		},
		{
			name:           "equals form",
			args:           []string{"~/Samples", "--name=kick"},
			wantFlags:      []string{"--name=kick"},
			wantPositional: []string{"~/Samples"},
		},
		{
			name:           "directory sandwiched between flags",
			args:           []string{"--name", "kick", "~/Samples", "--format", "wav"},
			wantFlags:      []string{"--name", "kick", "--format", "wav"},
			wantPositional: []string{"~/Samples"},
		},
		{
			name:           "empty",
			args:           nil,
			wantFlags:      nil,
			wantPositional: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, positional := splitFlagsAndPositional(tt.args)
			if !reflect.DeepEqual(flags, tt.wantFlags) {
				t.Errorf("flags = %#v, want %#v", flags, tt.wantFlags)
			}
			if !reflect.DeepEqual(positional, tt.wantPositional) {
				t.Errorf("positional = %#v, want %#v", positional, tt.wantPositional)
			}
		})
	}
}
