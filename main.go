package main

import (
	"flag" // Import the flag package for command-line argument parsing
	"log"
	"os"

	"github.com/ExecutiveOrder6102/phoenix-koinly-converter/converter"
)

func main() {

	// Define command line flags.
	flag.BoolVar(&converter.Verbose, "v", false, "Enable verbose logging for debugging.")
	addRoundingCost := flag.Bool("r", false, "Add a cost entry adjusting for rounding differences.")
	flag.BoolVar(addRoundingCost, "rounding-cost", false, "Alias for -r")
	xapoMode := flag.Bool("xapo", false, "Convert and consolidate one or more Xapo statement CSV files.")
	flag.Parse() // Parse command-line arguments.

	// Check if a file path is provided after parsing flags.
	if flag.NArg() < 1 {
		log.Fatal("Please provide a Phoenix CSV file, or use -xapo with one or more Xapo CSV files.")
	}

	// Create the Koinly CSV file.
	koinlyFile, err := os.Create("koinly.csv")
	if err != nil {
		log.Fatalf("Error creating Koinly CSV: %v", err)
	}
	defer koinlyFile.Close()

	if *xapoMode {
		statements := make([]converter.XapoStatement, 0, flag.NArg())
		files := make([]*os.File, 0, flag.NArg())
		for _, filePath := range flag.Args() {
			f, err := os.Open(filePath)
			if err != nil {
				log.Fatalf("Error opening Xapo CSV %q: %v", filePath, err)
			}
			files = append(files, f)
			statements = append(statements, converter.XapoStatement{
				Name:   filePath,
				Reader: f,
			})
		}
		defer func() {
			for _, f := range files {
				f.Close()
			}
		}()
		if err := converter.ConvertXapoStatements(statements, koinlyFile); err != nil {
			log.Fatalf("Xapo conversion failed: %v", err)
		}
		log.Println("Xapo conversion complete: consolidated koinly.csv created successfully.")
		return
	}

	filePath := flag.Arg(0)
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Error opening Phoenix CSV: %v", err)
	}
	defer f.Close()

	if err := converter.Convert(f, koinlyFile, *addRoundingCost); err != nil {
		log.Fatalf("Conversion failed: %v", err)
	}

	log.Println("Conversion complete: koinly.csv created successfully.")
}
