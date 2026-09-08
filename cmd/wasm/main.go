//go:build js && wasm

package main

import (
	"bytes"
	"fmt"
	"strings"
	"syscall/js"

	"github.com/ExecutiveOrder6102/phoenix-koinly-converter/converter"
)

func convertPhoenixToKoinly(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return "Error: No CSV data provided"
	}
	inputCSV := args[0].String()

	addRoundingCost := false
	if len(args) > 1 && args[1].Truthy() {
		addRoundingCost = args[1].Bool()
	}

	r := strings.NewReader(inputCSV)
	var buf bytes.Buffer

	// Enable verbose if needed, though we don't capture logs here easily unless we redirect log output.
	// converter.Verbose = true

	if err := converter.Convert(r, &buf, addRoundingCost); err != nil {
		return fmt.Sprintf("Error converting: %v", err)
	}

	return buf.String()
}

func convertXapoToKoinly(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 || args[0].Type() != js.TypeObject {
		return "Error: No Xapo CSV data provided"
	}

	files := args[0]
	statements := make([]converter.XapoStatement, 0, files.Length())
	for i := 0; i < files.Length(); i++ {
		statement, err := xapoStatementFromJS(files.Index(i))
		if err != nil {
			return fmt.Sprintf("Error reading Xapo statement %d: %v", i+1, err)
		}
		statements = append(statements, statement)
	}
	if len(statements) == 0 {
		return "Error: No Xapo CSV data provided"
	}

	var buf bytes.Buffer
	if err := converter.ConvertXapoStatements(statements, &buf); err != nil {
		return fmt.Sprintf("Error converting Xapo statements: %v", err)
	}
	return buf.String()
}

func xapoStatementFromJS(file js.Value) (converter.XapoStatement, error) {
	if file.Type() == js.TypeString {
		// Keep accepting the original string-only API, but without a filename a
		// Move to Savings row remains conservatively unclassified.
		return converter.XapoStatement{Reader: strings.NewReader(file.String())}, nil
	}
	if file.Type() != js.TypeObject {
		return converter.XapoStatement{}, fmt.Errorf("expected a CSV string or {name, contents} object")
	}

	contents := file.Get("contents")
	if contents.Type() != js.TypeString {
		return converter.XapoStatement{}, fmt.Errorf("missing CSV contents")
	}
	name := file.Get("name")
	if name.Type() != js.TypeString {
		return converter.XapoStatement{}, fmt.Errorf("missing statement filename")
	}
	return converter.XapoStatement{
		Name:   name.String(),
		Reader: strings.NewReader(contents.String()),
	}, nil
}

func main() {
	c := make(chan struct{}, 0)
	js.Global().Set("convertPhoenixToKoinly", js.FuncOf(convertPhoenixToKoinly))
	js.Global().Set("convertXapoToKoinly", js.FuncOf(convertXapoToKoinly))
	fmt.Println("WASM Initialized: Phoenix and Xapo conversion functions are ready.")
	<-c
}
