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
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Validates that the input file is a real file (and not a directory)
func isFileValid(fileName string) bool {
	info, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// validates whether  the month parameter has the correct format ("YYYY-MM" or "latest")
func isValidMonth(month string, isVerbose bool) bool {
	if month == "" {
		if isVerbose {
			fmt.Print("Empty month\n")
		}
		return false
	}
	if strings.ToUpper(month) == "LATEST" {
		return true
	}

	regexpMonth := regexp.MustCompile(`20[12][0-9]-(0[1-9]|1[0-2])`)
	if !regexpMonth.MatchString(month) {
		if isVerbose {
			fmt.Printf("Supplied data (%s) is not in a valid month format. Should be \"YYYY-MM\" and later than 2010\n", month)
		}
		return false
	}

	return true
}

// Write the string slice to a file formatted as a CSV
func writeCSVtoFile(outputFileName string, csv_output_slice [][]string) {
	//Open output file
	out, err := os.Create(outputFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	//Write the collected data as a CSV file
	csv_out := csv.NewWriter(out)
	write_err := csv_out.WriteAll(csv_output_slice)
	if write_err != nil {
		log.Fatal(err)
	}
	csv_out.Flush()
}

// returns true if the file extension is .md.
// It returns false in other cases, thus assuming a CSV output
func isWithMDfileExtension(filename string) bool {
	extension := filepath.Ext(filename)
	if strings.ToLower(extension) == ".md" {
		return true
	} else {
		return false
	}
}

// TODO: externalize the header creation
// TODO: return error
// Writes the data as Markdown
func writeDataAsMarkdown(outputFileName string, output_data_slice [][]string, introductionText string) {
	//Open output file
	f, err := os.Create(outputFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	out := bufio.NewWriter(f)

	width_slice, err := get_columnsWidth(output_data_slice)
	if err != nil {
		log.Fatal(err)
	}

	//Write the intro text if present
	if len(introductionText) > 0 {
		fmt.Fprintf(out, "%s\n", introductionText)
	}

	for lineNumber, dataLine := range output_data_slice {
		//Are we dealing with the title (and underline) ?
		isHeaderUnderline := false
		if lineNumber == 1 {
			isHeaderUnderline = true
		}

		writeBuffer := "|"
		underlineBuffer := "|"
		for columnNbr, data := range dataLine {
			//Check whether the value is numerical (we don't treat the case of float data)
			_, atoi_err := strconv.Atoi(data)
			exact_width := 0
			if atoi_err != nil {
				//not integer -> left align
				exact_width = 0 - width_slice[columnNbr]
			} else {
				//Integer -> right align
				exact_width = width_slice[columnNbr]
			}

			// We are dealing with the logic of the underline
			headerUnderline := ""
			if isHeaderUnderline {
				if exact_width <= 0 {
					headerUnderline = strings.Repeat("-", width_slice[columnNbr])
				} else {
					headerUnderline = strings.Repeat("-", width_slice[columnNbr]-1) + ":"
				}
				underlineBuffer = underlineBuffer + " " + headerUnderline + " |"
			}

			formattedData := fmt.Sprintf(" %*s", exact_width, data)
			writeBuffer = writeBuffer + formattedData + " |"
		}
		if isHeaderUnderline {
			fmt.Fprint(out, underlineBuffer+"\n")
		}
		fmt.Fprint(out, writeBuffer+"\n")
	}

	out.Flush()
}

// Returns a list of the maximum width of data supplied in data slice
func get_columnsWidth(output_data_slice [][]string) (width_slice []int, err error) {

	announced_nbr_columns := len(output_data_slice[0])
	for i := 0; i < announced_nbr_columns; i++ {
		width_slice = append(width_slice, 0)
	}

	//Walk through every line
	for lineNbr, slice_line := range output_data_slice {
		//Check column numbers for mismatch
		nbr_columns := len(output_data_slice[lineNbr])
		if nbr_columns != announced_nbr_columns {
			err = fmt.Errorf("line #%d has %d column while expecting %d \n", lineNbr+1, nbr_columns, announced_nbr_columns)
			return nil, err
		}

		//get the size of each data cell and update the counter slice if necessary
		for columnNbr, data_cell := range slice_line {
			if len(data_cell) > width_slice[columnNbr] {
				width_slice[columnNbr] = len(data_cell)
			}
		}
	}
	return width_slice, nil
}

// CheckDir verifies a given path/file string actually exists. If it does not
// then exit with an error.
func CheckDir(file string) error {
	path := filepath.Dir(file)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("The directory of specified output file (%s) does not exist.", path)
		}
	}
	return nil
}

// Based on the requested output filename (pivot table), builds a filename to store the history
func generateHistoryFilename(outputFilename string, dataType InputType, isCompare bool) (historyFilename string) {

	//Get the path part from the output filename
	path := filepath.Dir(outputFilename)

	//Compute filename elements based on parameters
	historyFilenameType := ""
	if dataType == InputTypeCommenters {
		historyFilenameType = "commenters"
	} else {
		historyFilenameType = "submitters"
	}
	extractType := ""
	if isCompare {
		extractType = "_evolution"
	}

	historyFilename = path + "/" + "top_" + historyFilenameType + extractType + "_fullHistory.csv"

	return historyFilename
}

// Will retrieve and write the history line for all the top users
func writeHistoryOutput(historyOutputFilename string, inputFilename string, csv_output_slice [][]string) (err error) {
	// TODO: need to mark churned or new users

	// Check is the csv_output_slice is at least 1 record + tile long
	if len(csv_output_slice) <= 2 {
		return fmt.Errorf("The generated top user data seems empty.")
	}

	// Load the pivot table in memory
	pivotRecords, loadErr := loadInputPivotTable(inputFilename)
	if loadErr != nil {
		return loadErr
	}

	//do we have data in the pivot table ?
	if len(pivotRecords) <= 2 {
		return fmt.Errorf("The pivot table (%s) seems empty.", inputFilename)
	}

	//This is a new slice that will contain the data to write
	var historicDataSlice [][]string

	//Get the title line and add it to the output
	historicDataSlice = append(historicDataSlice, pivotRecords[0])

	for topUser_index, topUser_line := range csv_output_slice {

		//We are dealing with the title line that we just want to skip
		if topUser_index == 0 {
			continue
		}

		//get the line index of the line containing the top user's data
		name := topUser_line[0]
		index := getIndexInPivotTable(pivotRecords, name)

		//check that return value is not negative (not found)
		if index == -1 {
			return fmt.Errorf("Supplied name (%s) was not found in input pivot table file", name)
		}

		// Add the collected data
		historicDataSlice = append(historicDataSlice, pivotRecords[index])
	}
	//Write the CSV
	writeCSVtoFile(historyOutputFilename, historicDataSlice)

	return nil
}

// returns the index in the pivot record's slice with the supplied name.
// Returns -1 if not found
func getIndexInPivotTable(pivotRecords [][]string, name string) (index int) {
	index = -1

	for indexNbr, line := range pivotRecords {
		if line[0] == name {
			return indexNbr
		}
	}

	return index

}
