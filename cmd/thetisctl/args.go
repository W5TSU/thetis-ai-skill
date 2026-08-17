package main

import (
	"strings"
)

// parsedArgs splits a subcommand's argument list into positionals and
// "--name value" / "--name=value" / bare "--name" flags, in any order. This
// is deliberately simpler than stdlib flag.FlagSet, which stops parsing at
// the first non-flag argument and so can't handle "cmd 0 --file x.wav"
// (positional before flag) — exactly the shape thetisctl's subcommands use.
type parsedArgs struct {
	pos   []string
	flags map[string]string
}

// boolFlagNames lists flags that never take a value (bare "--name").
var boolFlagNames = map[string]bool{
	"audio": true,
}

func parseArgs(args []string) parsedArgs {
	ap := parsedArgs{flags: map[string]string{}}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			ap.pos = append(ap.pos, a)
			continue
		}
		name := strings.TrimPrefix(a, "--")
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			ap.flags[name[:eq]] = name[eq+1:]
			continue
		}
		if boolFlagNames[name] {
			ap.flags[name] = "true"
			continue
		}
		if i+1 < len(args) {
			ap.flags[name] = args[i+1]
			i++
			continue
		}
		ap.flags[name] = "true"
	}
	return ap
}

func (a parsedArgs) flag(name, def string) string {
	if v, ok := a.flags[name]; ok {
		return v
	}
	return def
}

func (a parsedArgs) has(name string) bool {
	_, ok := a.flags[name]
	return ok
}
