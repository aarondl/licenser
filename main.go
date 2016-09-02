package main

import (
	"fmt"
	"os"

	"github.com/aarondl/licenser/licenselib"
	"github.com/olekukonko/tablewriter"
)

func main() {
	matches, err := licenselib.File(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	writer := tablewriter.NewWriter(os.Stdout)

	for _, match := range matches {
		writer.Append([]string{match.License.SpdxID, fmt.Sprintf("%0.2f", 100.0*match.Coefficient)})
	}

	writer.Render()
}
