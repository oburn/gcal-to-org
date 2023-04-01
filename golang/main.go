package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	backDaysPtr := flag.Int("backDays", 720, "How many days back to process events")
	forwardDaysPtr := flag.Int("forwardDays", 365, "How many days forward to process events")
	portPtr := flag.Int("port", 3000, "The port to run the callback server on localhost")
	storePtr := flag.String("store", "$HOME/.local/share/gcal-to-org", "Directory to store tokens")

	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Println("Expected FILE to output to")
		fmt.Printf("Usage: %s <flags> FILE", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	fmt.Println("backDays:", *backDaysPtr)
	fmt.Println("forwardDays:", *forwardDaysPtr)
	fmt.Println("port:", *portPtr)
	fmt.Println("store:", *storePtr)

	store, err := storeDir(*storePtr)
	if err != nil {
		fmt.Printf("Unable to create %s directory\n", *storePtr)
		os.Exit(1)
	}
	fmt.Println("store:", store)
	fmt.Println("dest:", flag.Arg(0))
}

func storeDir(unexanded string) (string, error) {
	expanded := os.ExpandEnv(unexanded)
	fmt.Printf("Ensuring that %s exists\n", expanded)
	err := os.MkdirAll(expanded, 0700)
	if err != nil {
		return "", err
	}
	return expanded, nil
}
