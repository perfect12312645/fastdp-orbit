package output

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

// Format represents output format type
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)

// Printer handles output formatting
type Printer struct {
	format Format
}

// NewPrinter creates a new printer
func NewPrinter(format Format) *Printer {
	return &Printer{format: format}
}

// Print outputs data in the specified format
func (p *Printer) Print(data interface{}) error {
	switch p.format {
	case FormatJSON:
		return p.printJSON(data)
	default:
		return p.printText(data)
	}
}

func (p *Printer) printJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func (p *Printer) printText(data interface{}) error {
	_, err := fmt.Printf("%v\n", data)
	return err
}

// PrintTable prints data as a table
func PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print headers
	for i, h := range headers {
		if i > 0 {
			fmt.Fprintf(w, "\t")
		}
		fmt.Fprintf(w, h)
	}
	fmt.Fprintf(w, "\n")

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprintf(w, "\t")
			}
			fmt.Fprintf(w, cell)
		}
		fmt.Fprintf(w, "\n")
	}

	w.Flush()
}

// PrintSuccess prints a success message in green
func PrintSuccess(msg string) {
	fmt.Printf("%s[OK]   %s %s\n", colorGreen, colorReset, msg)
}

// PrintError prints an error message in red
func PrintError(msg string) {
	fmt.Printf("%s[ERR]  %s %s\n", colorRed, colorReset, msg)
}

// PrintWarning prints a warning message in yellow
func PrintWarning(msg string) {
	fmt.Printf("%s[WARN] %s %s\n", colorYellow, colorReset, msg)
}

// PrintInfo prints an info message
func PrintInfo(msg string) {
	fmt.Printf("[INFO] %s\n", msg)
}
