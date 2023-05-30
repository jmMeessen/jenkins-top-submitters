/*
Copyright © 2023 Jean-Marc Meessen jean-marc@meessen-web.org

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"

	"github.com/spf13/cobra"
)

var isVerboseCheck bool

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check [input file]",
	Short: "Validates if input file has the correct format",
	Long: `The CHECK command validates whether the input file is processable.
	It must absolutely be generated by the GNU "datamash" pivot function in
	order to be successfully processed.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
			return err
		}
		if !isFileValid(args[0]) {
			return fmt.Errorf("Invalid file")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("checking", args[0], " with isVerboseCheck =", isVerboseCheck)

		if !checkFile(args[0]) {
			fmt.Print("Check failed.")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)

	checkCmd.PersistentFlags().BoolVar(&isVerboseCheck, "verbose", false, "Displays useful info about the checked file")
}

// Loads the data from a file and try to parse it as a CSV
func checkFile(fileName string) bool {

	var isValidTable = true

	f, err := os.Open(fileName)
	if err != nil {
		log.Printf("Unable to read input file "+fileName+"\n", err)
		return false
	}
	defer f.Close()

	r := csv.NewReader(f)

	//The first record is not properly formatted, we skip it
	firstLine, err1 := r.Read()
	if err1 != nil {
		log.Printf("Unexpected error loading"+fileName+"\n", err)
		return false
	}

	if isVerboseCheck {
		nbrOfColumns := len(firstLine)
		fmt.Println("Checking file format")
		fmt.Printf("  - Number of columns defined in header: %d\n", nbrOfColumns)
	}

	// first column should be empty
	if firstLine[0] != "" {
		fmt.Println("Not the expected first column name (should be empty)")
		return false
	}
	if isVerboseCheck {
		fmt.Println("  - File's header start with empty column name.")
	}

	//loop through columns to check headings
	month_regexp, _ := regexp.Compile("20[0-9]{2}-[0-9]{2}")
	for i, s := range firstLine {
		if i != 0 {
			if !month_regexp.MatchString(s) {
				fmt.Printf("Column header %s is not of the expected format (YYYY-MM)\n", s)
				return false
			}
		}
	}
	if isVerboseCheck {
		fmt.Println("  - File's header data column format (\"20YY-MM\")")
	}

	records, err := r.ReadAll()
	if err != nil {
		log.Printf("Unexpected error loading"+fileName+"\n", err)
		return false
	}

	//The GitHub user validation regexp (see https://stackoverflow.com/questions/58726546/github-username-convention-using-regex)
	// should be regexp.Compile(`^[a-zA-Z0-9]+(?:-[a-zA-Z0-9]+)*$`). But the dataset contains "invalid" data: username ending with a "-" or
	// a double "-" in the name.
	name_exp, _ := regexp.Compile(`^[a-zA-Z0-9\-]+$`)

	//Check the loaded data
	for i, dataLine := range records {
		//Skip header line as it has already been checked
		if i == 0 {
			continue
		}
		for ii, column := range dataLine {
			//check the GitHub user (first columns)
			if ii == 0 {
				if !(len(column) < 40 && len(column) > 0 && name_exp.MatchString(column)) {
					fmt.Printf("Submitter \"%s\" at line %d does not follow GitHub rules\n", column, i)
					return false
				}
			} else {
				// check the other columns is an integer (we don't check the sign)
				if _, err := strconv.Atoi(column); err != nil {
					fmt.Printf("Value \"%s\" at line %d (column %d) isn't an integer\n", column, i, ii)
					return false
				}
			}
		}
	}

	if isVerboseCheck {
		fmt.Println("  - Number of data columns match header columns.")
		fmt.Printf("  - Records have a valid GitHub username and number of submitted PRs. (%d data records)\n", len(records)-1)
	}

	fmt.Printf("\nSuccessfully checked \"%s\"\n   It is a valid Jenkins Submitter Pivot Table and can be processes\n\n", fileName)

	return isValidTable
}
