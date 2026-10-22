//go:build windows || linux

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/amxv/noria/internal/tray"
)

var version = "dev"

func main() {
	home := flag.String("home", "", "Noria home directory")
	check := flag.Bool("check", false, "validate tray runtime without opening a graphical session")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected tray arguments")
		os.Exit(2)
	}
	if *check {
		fmt.Println("noria-tray " + version + " ready")
		return
	}
	if err := tray.Run(*home, version); err != nil {
		fmt.Fprintln(os.Stderr, "Noria tray:", err)
		os.Exit(1)
	}
}
