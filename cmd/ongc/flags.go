// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package main

import (
	"flag"
	"strings"
)

// boolFlag is the unexported interface that flag.Value types implement when
// they do not consume a following argument (see stdlib's boolValue type).
// Matching on this lets us distinguish --dryrun from --radius 5 correctly.
type boolFlag interface {
	IsBoolFlag() bool
}

// reorderFlags moves any flag-shaped token (-foo / --foo / --foo=bar) from
// anywhere in args to the front. stdlib flag.FlagSet.Parse stops at the
// first non-flag token; hoisting lets every registered flag be seen
// regardless of original order. Unknown flag tokens are also hoisted so
// Parse rejects them loudly instead of silently leaving them in fs.Args().
//
// A bare "--" terminates flag hoisting: everything after it (including
// flag-shaped tokens) is preserved as positional arguments, matching
// POSIX and stdlib flag conventions. The "--" itself is forwarded to
// Parse so it also treats it as a terminator.
//
// This mirrors PyOngc's CLI (Click), which accepts interspersed flags, and
// fixes the class of silent-drop bug where `ongc neighbors NAME --radius 5`
// left --radius at its default value.
func reorderFlags(fs *flag.FlagSet, args []string) []string {
	flagsFront := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))

	i := 0
	for i < len(args) {
		tok := args[i]

		// POSIX end-of-flags marker: preserve "--" plus everything after
		// it verbatim as positionals, without any further hoisting.
		if tok == "--" {
			flagsFront = append(flagsFront, tok)
			positionals = append(positionals, args[i+1:]...)
			break
		}

		name, _, hasInline := splitFlagToken(tok)

		if name == "" {
			// Not a -flag / --flag form; treat as positional.
			positionals = append(positionals, tok)
			i++
			continue
		}

		f := fs.Lookup(name)

		// Unknown flags go to the front too, so fs.Parse rejects them.
		if f == nil {
			flagsFront = append(flagsFront, tok)
			i++
			continue
		}

		if hasInline || isBoolFlag(f) {
			flagsFront = append(flagsFront, tok)
			i++
			continue
		}

		// Value flag without inline "=value" — pull the following token too.
		flagsFront = append(flagsFront, tok)
		i++
		if i < len(args) {
			flagsFront = append(flagsFront, args[i])
			i++
		}
	}

	return append(flagsFront, positionals...)
}

// splitFlagToken recognizes "-name", "--name", "-name=val", "--name=val".
// The name must start with an ASCII letter so negative numbers and signed
// coordinates like "-00:09:01.3" are treated as positionals, not flags.
// Returns ("", "", false) for anything that isn't a flag form.
func splitFlagToken(tok string) (name, value string, hasValue bool) {
	if tok == "-" || tok == "--" || !strings.HasPrefix(tok, "-") {
		return "", "", false
	}
	rest := strings.TrimLeft(tok, "-")
	if rest == "" {
		return "", "", false
	}
	c := rest[0]
	if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
		return "", "", false
	}
	if idx := strings.Index(rest, "="); idx >= 0 {
		return rest[:idx], rest[idx+1:], true
	}
	return rest, "", false
}

func isBoolFlag(f *flag.Flag) bool {
	b, ok := f.Value.(boolFlag)
	return ok && b.IsBoolFlag()
}
