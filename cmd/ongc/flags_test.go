// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package main

import (
	"flag"
	"reflect"
	"testing"
)

// TestReorderFlagsHoistsPostPositional locks in the B1 invariant: flags
// placed anywhere in args must end up in front of positionals, so the
// stdlib flag.FlagSet.Parse (which stops at the first non-flag token)
// sees every registered flag.
func TestReorderFlagsHoistsPostPositional(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "flag after positional — value flag, space-separated",
			args: []string{"NGC7000", "--radius", "5"},
			want: []string{"--radius", "5", "NGC7000"},
		},
		{
			name: "flag after positional — value flag, =-separated",
			args: []string{"NGC7000", "--radius=5"},
			want: []string{"--radius=5", "NGC7000"},
		},
		{
			name: "bool flag after positional",
			args: []string{"M1", "--details"},
			want: []string{"--details", "M1"},
		},
		{
			name: "two positionals with flag in the middle",
			args: []string{"20:59:17", "--catalog", "IC", "+44:31:43"},
			want: []string{"--catalog", "IC", "20:59:17", "+44:31:43"},
		},
		{
			name: "already-ordered flags stay put",
			args: []string{"--radius", "5", "--catalog", "IC", "NGC7000"},
			want: []string{"--radius", "5", "--catalog", "IC", "NGC7000"},
		},
		{
			name: "unknown flag is hoisted so fs.Parse rejects it",
			args: []string{"NGC0001", "--bogus"},
			want: []string{"--bogus", "NGC0001"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.Float64("radius", 60, "")
			fs.String("catalog", "all", "")
			fs.Bool("details", false, "")

			got := reorderFlags(fs, tc.args)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("reorderFlags(%q) = %q, want %q", tc.args, got, tc.want)
			}
		})
	}
}

// TestReorderFlagsPreservesBoolDetection ensures value flags that look like
// booleans on their own (e.g. "--radius" with no following token) are still
// pulled with the next token to keep flag.Parse's value-flag contract.
func TestReorderFlagsValueFlagConsumesNext(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.Float64("radius", 60, "")

	got := reorderFlags(fs, []string{"NAME", "--radius", "5"})
	want := []string{"--radius", "5", "NAME"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("reorderFlags = %q, want %q", got, want)
	}
}

// TestReorderFlagsHonorsDashDashTerminator locks in the F4 invariant:
// a bare "--" terminates flag hoisting (POSIX / stdlib convention), so
// flag-shaped tokens after it stay literal positionals.
func TestReorderFlagsHonorsDashDashTerminator(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.Bool("details", false, "")
	fs.Float64("radius", 0, "")

	cases := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "flag after -- stays positional",
			args: []string{"NGC0001", "--", "--details"},
			want: []string{"--", "NGC0001", "--details"},
		},
		{
			name: "value-flag-shaped tokens after -- stay positional",
			args: []string{"--", "--radius", "5"},
			want: []string{"--", "--radius", "5"},
		},
		{
			name: "flag before --, literal after",
			args: []string{"NGC0001", "--details", "--", "--radius", "5"},
			want: []string{"--details", "--", "NGC0001", "--radius", "5"},
		},
		{
			name: "no -- behaves as before",
			args: []string{"NGC0001", "--details"},
			want: []string{"--details", "NGC0001"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := reorderFlags(fs, tc.args)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("reorderFlags(%q) = %q, want %q", tc.args, got, tc.want)
			}
		})
	}
}

// TestReorderFlagsIgnoresNegativeCoords confirms that "-00:09:01.3" and
// negative numbers are treated as positional arguments, not flags — splitting
// on the first character after the leading dashes. Essential for the nearby
// subcommand which accepts Dec like "-00:09:01.3" as a positional.
func TestReorderFlagsIgnoresNegativeCoords(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.Float64("radius", 60, "")

	got := reorderFlags(fs, []string{"11:08:44", "-00:09:01.3"})
	want := []string{"11:08:44", "-00:09:01.3"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("reorderFlags = %q, want %q", got, want)
	}

	// Same guarantee when a real flag is interleaved.
	got = reorderFlags(fs, []string{"11:08:44", "-00:09:01.3", "--radius", "5"})
	want = []string{"--radius", "5", "11:08:44", "-00:09:01.3"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("reorderFlags = %q, want %q", got, want)
	}
}
