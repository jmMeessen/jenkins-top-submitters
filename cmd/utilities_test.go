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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_isFileValid(t *testing.T) {
	type args struct {
		fileName string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"Happy case",
			args{"../test_data/not_a_csv.txt"},
			true,
		},
		{
			"File does not exist",
			args{"unexistantFile.txt"},
			false,
		},
		{
			"File is a directory in fact",
			args{"../test_data"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFileValid(tt.args.fileName); got != tt.want {
				t.Errorf("isFileValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateMonth(t *testing.T) {
	type args struct {
		month     string
		isVerbose bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"lowercase latest",
			args{"latest", true},
			true,
		},
		{
			"uppercase latest",
			args{"LATEST", true},
			true,
		},
		{
			"empty month",
			args{"", true},
			false,
		},
		{
			"happy case 1",
			args{"2023-08", true},
			true,
		},
		{
			"happy case 2",
			args{"2013-08", true},
			true,
		},
		{
			"happy case 3",
			args{"2023-12", true},
			true,
		},
		{
			"happy case 4",
			args{"2020-12", true},
			true,
		},
		{
			"invalid month 1",
			args{"2023-13", true},
			false,
		},
		{
			"invalid month 2",
			args{"2023-00", true},
			false,
		},
		{
			"invalid year (too old)",
			args{"2003-08", true},
			false,
		},
		{
			"plain junk 1",
			args{"2023", true},
			false,
		},
		{
			"plain junk 2",
			args{"blaah", true},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidMonth(tt.args.month, tt.args.isVerbose); got != tt.want {
				t.Errorf("validateMonth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isWithMDfileExtension(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"Markdown extension",
			args{filename: "myfile.md"},
			true,
		},
		{
			"Markdown extension (mixed case)",
			args{filename: "myfile.mD"},
			true,
		},
		{
			"CSV extension",
			args{filename: "myfile.csv"},
			false,
		},
		{
			"no extension",
			args{filename: "myfile"},
			false,
		},
		{
			"just the dot",
			args{filename: "myfile."},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWithMDfileExtension(tt.args.filename); got != tt.want {
				t.Errorf("isWithMDfileExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_writeMarkdownFile(t *testing.T) {
	// Setup environment
	tempDir := t.TempDir()
	goldenMarkdownFilename, err := duplicateFile("../test_data/Reference_extract_output.md", tempDir)

	assert.NoError(t, err, "Unexpected File duplication error")
	assert.NotEmpty(t, goldenMarkdownFilename, "Failure to duplicate test file")

	// Setup input data
	testOutputFilename := tempDir + "markdown_output.md"
	introductionText := "# Extract\n"
	data := [][]string{
		{"Submitter", "Total_PRs"},
		{"basil", "1245"},
		{"MarkEWaite", "1150"},
		{"lemeurherve", "939"},
		{"NotMyFault", "926"},
		{"dduportal", "859"},
		{"jonesbusy", "415"},
		{"jglick", "378"},
		{"smerle33", "353"},
		{"timja", "250"},
		{"uhafner", "215"},
		{"gounthar", "208"},
		{"mawinter69", "179"},
		{"daniel-beck", "164"}}

	// Execute function under test
	writeDataAsMarkdown(testOutputFilename, data, introductionText)

	// result validation
	assert.True(t, isFileEquivalent(testOutputFilename, goldenMarkdownFilename))
}

func Test_writeHistoryOutput(t *testing.T) {
	// Setup environment
	inputPivotTableName := "../test_data/overview.csv"
	tempDir := t.TempDir()
	goldenHistoryFilename, err := duplicateFile("../test_data/historicExtract_reference.csv", tempDir)

	assert.NoError(t, err, "Unexpected File duplication error")
	assert.NotEmpty(t, goldenHistoryFilename, "Failure to duplicate test file")

	// Setup input data
	testOutputFilename := tempDir + "/history_output.csv"
	data := [][]string{
		{"Submitter", "Total_PRs"},
		{"basil", "1245"},
		{"MarkEWaite", "1150"},
		{"lemeurherve", "939"},
		{"NotMyFault", "926"},
		{"dduportal", "859"},
		{"jonesbusy", "415"},
		{"jglick", "378"},
		{"smerle33", "353"},
		{"timja", "250"},
		{"uhafner", "215"},
		{"gounthar", "208"},
		{"mawinter69", "179"},
		{"daniel-beck", "164"}}

	// Execute function under test
	writeErr := writeHistoryOutput(testOutputFilename, inputPivotTableName, data)
	assert.NoError(t, writeErr, "Function under test returned an unexpected error")

	// result validation
	assert.True(t, isFileEquivalent(testOutputFilename, goldenHistoryFilename))
}

func Test_writeHistoryOutput_notFoundUser(t *testing.T) {
	// Setup environment
	inputPivotTableName := "../test_data/overview.csv"
	tempDir := t.TempDir()

	// Setup input data
	testOutputFilename := tempDir + "/history_output.csv"
	data := [][]string{
		{"Submitter", "Total_PRs"},
		{"basil", "1245"},
		{"MarkEWaite", "1150"},
		{"lemeurherve", "939"},
		{"NotMyFault", "926"},
		{"dduportal", "859"},
		{"jonesbusy", "415"},
		{"jglick", "378"},
		{"smerle33", "353"},
		{"timja", "250"},
		{"uhafner", "215"},
		{"unknownUser", "208"},
		{"mawinter69", "179"},
		{"daniel-beck", "164"}}

	// Execute function under test
	writeErr := writeHistoryOutput(testOutputFilename, inputPivotTableName, data)

	assert.EqualErrorf(t, writeErr, "Supplied name (unknownUser) was not found in input pivot table file", "Function under test should have failed")

}

func Test_writeHistoryOutput_noTopUserData(t *testing.T) {
	// Setup environment
	inputPivotTableName := "../test_data/overview.csv"
	tempDir := t.TempDir()

	// Setup input data
	testOutputFilename := tempDir + "/history_output.csv"
	data := [][]string{
		{"Submitter", "Total_PRs"},
	}

	// Execute function under test
	writeErr := writeHistoryOutput(testOutputFilename, inputPivotTableName, data)

	assert.EqualErrorf(t, writeErr, "The generated top user data seems empty.", "Function under test should have failed")
}

func Test_writeHistoryOutput_noPivotTableData(t *testing.T) {
	// Setup environment
	inputPivotTableName := "../test_data/noData_overview.csv"
	tempDir := t.TempDir()

	// Setup input data
	testOutputFilename := tempDir + "/history_output.csv"
	data := [][]string{
		{"Submitter", "Total_PRs"},
		{"basil", "1245"},
		{"MarkEWaite", "1150"},
		{"lemeurherve", "939"},
		{"NotMyFault", "926"},
		{"dduportal", "859"},
		{"jonesbusy", "415"},
		{"jglick", "378"},
		{"smerle33", "353"},
		{"timja", "250"},
		{"uhafner", "215"},
		{"unknownUser", "208"},
		{"mawinter69", "179"},
		{"daniel-beck", "164"}}

	// Execute function under test
	writeErr := writeHistoryOutput(testOutputFilename, inputPivotTableName, data)

	assert.EqualErrorf(t, writeErr, "The pivot table (../test_data/noData_overview.csv) seems empty.", "Function under test should have failed")
}

func Test_getIndexInPivotTable(t *testing.T) {
	testInputSlice := [][]string{
		{"", "month_1", "month_2", "month_3"},
		{"basil", "1245", "1", "2"},
		{"MarkEWaite", "1150", "1", "2"},
		{"lemeurherve", "939", "1", "2"},
		{"NotMyFault", "926", "1", "2"},
		{"dduportal", "859", "1", "2"},
		{"jonesbusy", "415", "1", "2"},
		{"jglick", "378", "1", "2"},
		{"smerle33", "353", "1", "2"},
		{"timja", "250", "1", "2"},
		{"uhafner", "215", "1", "2"},
		{"gounthar", "208", "1", "2"},
		{"mawinter69", "179", "1", "2"},
		{"daniel-beck", "164", "1", "2"}}

	type args struct {
		pivotRecords [][]string
		name         string
	}
	tests := []struct {
		name      string
		args      args
		wantIndex int
	}{
		{
			"Happy case - 1",
			args{pivotRecords: testInputSlice, name: "basil"},
			1,
		},
		{
			"Happy case - 2",
			args{pivotRecords: testInputSlice, name: "uhafner"},
			10,
		},
		{
			"Happy case - 3",
			args{pivotRecords: testInputSlice, name: "daniel-beck"},
			13,
		},
		{
			"notfound",
			args{pivotRecords: testInputSlice, name: "jmm"},
			-1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotIndex := getIndexInPivotTable(tt.args.pivotRecords, tt.args.name); gotIndex != tt.wantIndex {
				t.Errorf("getIndexInPivotTable() = %v, want %v", gotIndex, tt.wantIndex)
			}
		})
	}
}
func Test_CheckDir(t *testing.T) {
	type args struct {
		file string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"Valid directory",
			args{file: "../test_data/fle-1.txt"},
			false,
		},
		{
			"Invalid directory",
			args{file: "../junkDir/fle-1.txt"},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckDir(tt.args.file); (err != nil) != tt.wantErr {
				t.Errorf("CheckDir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_generateHistoryFilename(t *testing.T) {
	type args struct {
		outputFilename string
		dataType       InputType
		isCompare      bool
	}
	tests := []struct {
		name                string
		args                args
		wantHistoryFilename string
	}{
		{
			"Happy case - submitters",
			args{
				outputFilename: "output.md",
				dataType:       InputTypeSubmitters,
				isCompare:      false,
			},
			"./top_submitters_fullHistory.csv",
		},
		{
			"Happy case - submitters - evolution",
			args{
				outputFilename: "output.md",
				dataType:       InputTypeSubmitters,
				isCompare:      true,
			},
			"./top_submitters_evolution_fullHistory.csv",
		},
		{
			"Happy case - commenters - evolution - with path",
			args{
				outputFilename: "consolidated_data/output.md",
				dataType:       InputTypeCommenters,
				isCompare:      true,
			},
			"consolidated_data/top_commenters_evolution_fullHistory.csv",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHistoryFilename := generateHistoryFilename(tt.args.outputFilename, tt.args.dataType, tt.args.isCompare)
			if gotHistoryFilename != tt.wantHistoryFilename {
				t.Errorf("generateHistoryFilename() = %v, want %v", gotHistoryFilename, tt.wantHistoryFilename)
			}
		})
	}
}

// ------------------------------
//
// Test Utilities
//
// ------------------------------

// duplicate test file as a temporary file.
// The temporary directory should be created in the calling test so that it gets cleaned at test completion.
func duplicateFile(originalFileName, targetDir string) (tempFileName string, err error) {

	//Check the status and size of the original file
	sourceFileStat, err := os.Stat(originalFileName)
	if err != nil {
		return "", err
	}
	if !sourceFileStat.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", originalFileName)
	}
	sourceFileSize := sourceFileStat.Size()

	//Open the original file
	source, err := os.Open(originalFileName)
	if err != nil {
		return "", err
	}
	defer source.Close()

	//Get the original file's extension
	originalFileExtension := filepath.Ext(originalFileName)

	// generate temporary file name in temp directory
	file, err := os.CreateTemp(targetDir, "testData.*"+originalFileExtension)
	if err != nil {
		return "", err
	}
	tempFileName = file.Name()

	// create the new file duplication
	destination, err := os.Create(tempFileName)
	if err != nil {
		return "", err
	}
	defer destination.Close()

	// Do the actual copy
	bytesCopied, err := io.Copy(destination, source)
	if err != nil {
		return tempFileName, err
	}
	if bytesCopied != sourceFileSize {
		return tempFileName, fmt.Errorf("Source and destination file size do not match after copy (%s is %d bytes and %s is %d bytes", originalFileName, sourceFileSize, tempFileName, bytesCopied)
	}

	// All went well
	return tempFileName, nil
}

func isFileEquivalent(tempFileName, goldenFileName string) bool {

	//FIXME: change this to an error return instead of boolean return

	// Is the size the same
	tempFileSize := getFileSize(tempFileName)
	goldenFileSize := getFileSize(goldenFileName)

	if tempFileSize == 0 || goldenFileSize == 0 {
		fmt.Printf("0 byte file length\n")
		return false
	}

	if tempFileSize != goldenFileSize {
		fmt.Printf("Files are of different sizes: found %d bytes while expecting reference %d bytes \n", tempFileSize, goldenFileSize)
		return false
	}

	// load both files
	err, tempFile_List := loadFileToTest(tempFileName)
	if err != nil {
		fmt.Printf("Unexpected error loading %s : %v \n", tempFileName, err)
		return false
	}

	err, goldenFile_List := loadFileToTest(goldenFileName)
	if err != nil {
		fmt.Printf("Unexpected error loading %s : %v \n", goldenFileName, err)
		return false
	}

	//Compare the two lists
	for index, line := range tempFile_List {
		if line != goldenFile_List[index] {
			fmt.Printf("Compare failure: line %d do not match\n", index)
			return false
		}
	}

	//If we reached this, we are all good
	return true
}

// load input file
func loadFileToTest(fileName string) (error, []string) {

	f, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("Unable to read input file %s: %v\n", fileName, err), nil
	}
	defer f.Close()

	var loadedFile []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		loadedFile = append(loadedFile, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Error loading \"%s\": %v", fileName, err), nil
	}

	if len(loadedFile) <= 1 {
		return fmt.Errorf("Error: \"%s\" seems empty. Retrieved %d lines.", fileName, len(loadedFile)), nil
	}

	return nil, loadedFile
}

// Gets the size of a file
func getFileSize(fileName string) int64 {
	tempFileStat, err := os.Stat(fileName)
	if err != nil {
		fmt.Printf("Unexpected error getting details of %s: %v\n", fileName, err)
		return 0
	}
	if !tempFileStat.Mode().IsRegular() {
		fmt.Printf("%s is not a regular file\n", fileName)
		return 0
	}
	return tempFileStat.Size()
}

func Test_get_columnsWidth(t *testing.T) {
	type args struct {
		output_data_slice [][]string
	}
	tests := []struct {
		name    string
		args    args
		want    []int
		wantErr bool
	}{
		{
			"Happy case",
			args{
				[][]string{
					{"aaa aaa", "12", "ccccccc"},
					{"aaa aaa aa", "124", "cccccccccc"},
					{"aaa", "12", "cccccccccc"},
					{"aaa", "1024", "cccccccccccc"},
				},
			},
			[]int{10, 4, 12},
			false,
		},
		{
			"empty field",
			args{
				[][]string{
					{"aaa aaa", "12", ""},
					{"aaa aaa aa", "124", "cccccccccc"},
					{"aaa", "12", "cccccccccc"},
					{"aaa", "1024", "cccccccccccc"},
				},
			},
			[]int{10, 4, 12},
			false,
		},
		{
			"Column number mismatch",
			args{
				[][]string{
					{"aaa aaa", "12", ""},
					{"aaa aaa aa", "124"},
					{"aaa", "12", "cccccccccc"},
					{"aaa", "1024", "cccccccccccc"},
				},
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := get_columnsWidth(tt.args.output_data_slice)
			if (err != nil) != tt.wantErr {
				t.Errorf("get_columnsWidth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("get_columnsWidth() = %v, want %v", got, tt.want)
			}
		})
	}
}
