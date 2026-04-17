package main

import (
	"fmt"
	"os"
)

const usageText = `Usage: ongc <subcommand> [args] [flags]

Subcommands:
  view NAME [--details]
  nearby RA DEC [--radius N] [--catalog all|NGC|IC]
  neighbors NAME [--radius N] [--catalog all|NGC|IC]
  search [--catalog] [--type] [--constellation] [--minsize] [--maxsize]
         [--uptobmag] [--uptovmag] [--minra] [--maxra] [--mindec] [--maxdec]
         [--named] [--withname]
  separation OBJ1 OBJ2
  stats
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "view":
		cmdView(os.Args[2:])
	case "nearby":
		cmdNearby(os.Args[2:])
	case "neighbors":
		cmdNeighbors(os.Args[2:])
	case "search":
		cmdSearch(os.Args[2:])
	case "separation":
		cmdSeparation(os.Args[2:])
	case "stats":
		cmdStats(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n%s", os.Args[1], usageText)
		os.Exit(1)
	}
}

func cmdView(args []string)       { fmt.Println("not implemented") }
func cmdNearby(args []string)     { fmt.Println("not implemented") }
func cmdNeighbors(args []string)  { fmt.Println("not implemented") }
func cmdSearch(args []string)     { fmt.Println("not implemented") }
func cmdSeparation(args []string) { fmt.Println("not implemented") }
func cmdStats(args []string)      { fmt.Println("not implemented") }
