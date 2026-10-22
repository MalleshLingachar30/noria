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
	checkHost := flag.Bool("check-host", false, "check whether the current desktop can display a tray icon")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected tray arguments")
		os.Exit(2)
	}
	if *check {
		fmt.Println("noria-tray " + version + " ready")
		return
	}
	if *checkHost {
		if err := tray.CheckHost(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("Desktop status icon host available")
		return
	}
	if err := tray.Run(*home, version); err != nil {
		fmt.Fprintln(os.Stderr, "Noria tray:", err)
		os.Exit(1)
	}
}
