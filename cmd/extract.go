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
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// Variables set from the command line
var outputFileName string
var topSize int
var period int
var endMonth string
var isVerboseExtract bool
var argInputType string
var isOutputHistory bool
var inputType InputType

type InputType uint8

const (
	InputTypeUnknown InputType = iota
	InputTypeSubmitters
	InputTypeCommenters
)

type totalized_record struct {
	User string //Submitter name
	Pr   int    //Number of PRs
}

// extractCmd represents the extract command
var extractCmd = &cobra.Command{
	Use:   "extract [input file]",
	Short: "Extracts the top submitters from the supplied pivot table",
	Long: `This command extract the top submitter for a given period (by default 12 months).
This interval is counted, by default, from the last month available in the pivot table.
The input file is first validated before being processed.

If not specified, the output file name is hardcoded to "top-submitters_YYYY-MM.csv". 
The "YYYY-MM" stands for the specified end month (see "--month" flag). It is "LATEST"
if not end month was specified (default).

The "months" parameter is the number of months used to compute the top users, 
counting from backwards from the last month. If a 0 months is specified, all the 
available months are counted.

The "topSize" parameter defines the number of users considered as top users.
If more submitters with the same amount of total PRs exist ("ex aequo"), they are included in 
the list (resulting in more thant the specified number of top users).  
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
			return err
		}
		if !isFileValid(args[0]) {
			return fmt.Errorf("Invalid file\n")
		}
		if !isValidMonth(endMonth, isVerboseExtract) {
			return fmt.Errorf("Invalid month\n")
		}

		// check the input type
		switch strings.ToLower(argInputType) {
		case "submitters":
			inputType = InputTypeSubmitters
		case "commenters":
			inputType = InputTypeCommenters
		default:
			inputType = InputTypeUnknown
		}

		if inputType == InputTypeUnknown {
			return fmt.Errorf("%s is an invalid input type\n", argInputType)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// When called standalone, we want to give the minimal information
		isSilent := true

		inputPivotTableName := args[0]

		// Check input file
		if !checkFile(inputPivotTableName, isSilent) {
			return fmt.Errorf("Invalid input file.")
		}

		// Extract the data (with no offset)
		result, real_endDate, csv_output_slice := extractData(inputPivotTableName, topSize, endMonth, period, 0, inputType, isVerboseExtract)
		if !result {
			return fmt.Errorf("Failed to extract data")
		}

		//FIXME: change default filename when specifying another type of input
		// If the default value is specified, update that default with the month being used for the calculation
		if outputFileName == "top-submitters_YYYY-MM.csv" {
			outputFileName = "top-submitters_" + strings.ToUpper(endMonth) + ".csv"
		}
		isMDoutput := isWithMDfileExtension(outputFileName)

		if isVerboseExtract {
			fileTypeText := "(CSV format)"
			if isMDoutput {
				fileTypeText = "(Markdown format)"
			}
			fmt.Printf("Writing extraction to \"%s\" %s\n\n", outputFileName, fileTypeText)
		}

		// Check that the output directory exists
		dirErr := CheckDir(outputFileName)
		if dirErr != nil {
			return dirErr
		}

		if isMDoutput {
			introduction := ""
			if inputType == InputTypeSubmitters {
				introduction = "# Top Submitters\n"
				buffer := fmt.Sprintf("\nExtraction of the %d top submitters (non-bot PR creators) \nover the %d months before \"%s\".\n\n", topSize, period, real_endDate)
				introduction = introduction + buffer
			}
			if inputType == InputTypeCommenters {
				introduction = "# Top Commenters\n"
				buffer := fmt.Sprintf("\nExtraction of the %d top (non-bot) commenters \nover the %d months before \"%s\".\n\n", topSize, period, real_endDate)
				introduction = introduction + buffer
			}
			writeDataAsMarkdown(outputFileName, csv_output_slice, introduction)
		} else {
			writeCSVtoFile(outputFileName, csv_output_slice)
		}

		//TODO: if requested, write the history based the supplied top user slice
		if isOutputHistory {
			isCompare := false
			historyOutputFilename := generateHistoryFilename(outputFileName, inputType, isCompare)

			if err := writeHistoryOutput(historyOutputFilename, inputPivotTableName, csv_output_slice); err != nil {
				return err
			}
		}

		return nil
	},
}

// Initialize the Cobra processor
func init() {
	rootCmd.AddCommand(extractCmd)

	// definition of flags and configuration settings.
	extractCmd.PersistentFlags().StringVarP(&outputFileName, "out", "o", "top-submitters_YYYY-MM.csv", "Output file name. Using the \".md\" extension will generate a markdown file ")
	extractCmd.PersistentFlags().StringVarP(&argInputType, "type", "", "submitters", "The type of data being analyzed. Can be either \"submitters\" or \"commenters\"")
	extractCmd.PersistentFlags().IntVarP(&topSize, "topSize", "t", 35, "Number of top submitters to extract.")
	extractCmd.PersistentFlags().IntVarP(&period, "period", "p", 12, "Number of months to accumulate.")
	extractCmd.PersistentFlags().StringVarP(&endMonth, "month", "m", "latest", "Month to extract top submitters.")
	extractCmd.PersistentFlags().BoolVarP(&isOutputHistory, "history", "", false, "Outputs the available activity history for the top submitters")

	extractCmd.PersistentFlags().BoolVarP(&isVerboseExtract, "verbose", "v", false, "Displays useful info during the extraction")
}

// Extracts the top submitters for a given period and writes it to a file.
// Offset defines the number of months before the specified endMonth the extraction must be done (needed for the COMPARE command).
func extractData(inputFilename string, topSize int, endMonth string, period int, offset int, inputType InputType, isVerboseExtract bool) (result bool, real_endDate string, outputSlice [][]string) {
	if isVerboseExtract {
		fmt.Printf("Extracting from \"%s\" the %d top submitters during the last %d months\n\n", inputFilename, topSize, period)
	}

	records, loadErr := loadInputPivotTable(inputFilename)
	if loadErr != nil {
		return false, "", nil
	}

	firstDataColumn, lastDataColumn, oldestDate, mostRecentDate := getBoundaries(records, endMonth, period, offset)

	if strings.ToUpper(endMonth) != "LATEST" {
		if endMonth != mostRecentDate {
			log.Printf("Unexpected error computing boundaries (\"%s\" != \"%s\"\n", endMonth, mostRecentDate)
			return false, "", nil
		}
	}

	//We need to make that information available to caller
	real_endDate = mostRecentDate

	fmt.Printf("Accumulating data between %s and  %s (columns %d and %d)\n",
		oldestDate, mostRecentDate, firstDataColumn, lastDataColumn)

	//Slice that will contain all the totalized records
	var new_output_slice []totalized_record

	for i, dataLine := range records {

		//Skip header line as it has already been checked
		if i == 0 {
			continue
		}

		recordTotal := 0
		for ii, column := range dataLine {
			if ii >= firstDataColumn && ii <= lastDataColumn {
				// fmt.Printf(", %s", column)

				// We don't treat conversion errors or negative values as the file has already been checked
				columnValue, _ := strconv.Atoi(column)
				recordTotal = recordTotal + columnValue
			}
		}

		//Add the total to the full list
		a_totalized_record := totalized_record{dataLine[0], recordTotal}
		new_output_slice = append(new_output_slice, a_totalized_record)
	}

	// Sort the slice, based on the number of PRs, in descending order
	sort.Slice(new_output_slice, func(i, j int) bool { return new_output_slice[i].Pr > new_output_slice[j].Pr })

	//Loop through list to find the top submitters (and ex-aequo) to load the final list
	current_total := 0
	isListComplete := false

	var csv_output_slice [][]string
	var header_row []string

	if inputType == InputTypeSubmitters {
		header_row = []string{"Submitter", "Total_PRs"}
	}
	if inputType == InputTypeCommenters {
		header_row = []string{"Commenter", "Total_Comments"}
	}

	csv_output_slice = append(csv_output_slice, header_row)
	for i, total_record := range new_output_slice {
		if i < topSize {
			current_total = total_record.Pr

			var work_row []string
			work_row = append(work_row, total_record.User, strconv.Itoa(total_record.Pr))
			csv_output_slice = append(csv_output_slice, work_row)
		} else {
			if !isListComplete {
				if current_total == total_record.Pr {
					//This is an ex-aequo, so add it to the list
					var work_row []string
					work_row = append(work_row, total_record.User, strconv.Itoa(total_record.Pr))
					csv_output_slice = append(csv_output_slice, work_row)
				} else {
					// we have all we need
					isListComplete = true
				}
			}
		}
	}

	return true, real_endDate, csv_output_slice
}

// Opens and reads the input as a CSV file
func loadInputPivotTable(inputFilename string) (loadedRecords [][]string, err error) {
	//TODO: add some unit tests here ?
	//At this stage of the processing, we assume that the input file is correctly formatted
	f, err := os.Open(inputFilename)
	if err != nil {
		return nil, fmt.Errorf("Unable to read input file "+inputFilename+"\n", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("Unexpected error loading"+inputFilename+"\n", err)
	}

	return records, nil
}

// Based on the number of months requested, computes the start/end column and associated date for the given dataset.
// Offset defines the number of months before the specified endMonth the extraction must be done
func getBoundaries(records [][]string, endMonthStr string, period int, offset int) (startColumn int, endColumn int, startMonth string, endMonth string) {
	isWithOffset := (offset != 0)
	nbrOfColumns := len(records[0])

	if strings.ToUpper(endMonthStr) == "LATEST" {
		endColumn = nbrOfColumns - 1
	} else {
		// Search the requested end month.
		endColumn = searchStringMonth(records[0], endMonthStr)
		//If not found, reset to "latest"
		if endColumn == -1 {
			fmt.Printf("Warning: %s not found in dataset, reverting to latest available month\n", endMonthStr)
			endColumn = nbrOfColumns - 1
		}
	}

	if isWithOffset {
		endColumn = endColumn - offset
		if endColumn <= 0 {
			fmt.Printf("FATAL: requested offset-ted end period not available.\n")
			return 0, 0, "", ""
		}
	}

	if period >= nbrOfColumns {
		period = 0
	}

	if period == 0 {
		startColumn = 1
	} else {
		startColumn = (endColumn - period) + 1
	}

	startMonth = records[0][startColumn]
	endMonth = records[0][endColumn]

	return startColumn, endColumn, startMonth, endMonth
}

// Searches the loaded records for the request month string
func searchStringMonth(headerRecords []string, endMonthStr string) (endColumn int) {
	nbrOfColumns := len(headerRecords)
	endColumn = -1 //not found value
	for i := 0; i < nbrOfColumns; i++ {
		if headerRecords[i] == endMonthStr {
			endColumn = i
			break
		}
	}
	return endColumn
}
