/* reversesearch_test package contains integration tests for ReverseSearch. This
includes a green paths test function and a red paths test function. Only exported
members of reversesearch package are tested. */
package reversesearch_test

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	. "github.com/freebiesoft/reversesearch"
)

// test files
var logsDir = `./testdata/test_logs/`
var accessLog = logsDir + `access.log`
var accessLogNoMoreEntries = logsDir + `access_no_more_entries.log`
var odlLog = logsDir + `odl.log`
var singleLog = logsDir + `single_log_file.log`
var singleLineLog = logsDir + `single_line_log_file.log`
var longLineLog = logsDir + `long_line_log.log`
var accessLogNlPrefixUnix = logsDir + `access_NL_prefix_UNIX.log`
var accessLogNlPrefixWin = logsDir + `access_NL_prefix_win.log`
var odlLogNoNlSuffixUnix = logsDir + `odl_no_NL_suffix_UNIX.log`
var odlLogNoNlSuffixWin = logsDir + `odl_no_NL_suffix_win.log`
var emptyFile = logsDir + `empty_file.log`
var nlOnlyUnix = logsDir + `NL_only_UNIX.log`
var nlOnlyWin = logsDir + `NL_only_win.log`
var oneLargeOneSmall = logsDir + `1_large_1_small.log`

// apache access log patterns (used throughout various tests)
var apacheStartPattern = `^(?:\S+) (?:\S+) (?:\S+) \[([\w:/]+\s[+\-]\d{4})\]`
var apacheTimeFormat = `02/Jan/2006:15:04:05 -0700`

// ODL log patterns (used throughout various tests)
var odlStartPattern = `^<(\w{3} \d{2}, \d{4} \d{1,2}:\d{2}:\d{2} (?:AM|PM) (?:\S+))>`
var odlTimeFormat = `Jan 2, 2006 3:04:05 PM MST`

// utility function for checking errors
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// utility method for parsing time (useful for inline definitions in structs)
func parseTime(format string, timeStr string) time.Time {
	t, err := time.Parse(format, timeStr)
	check(err)
	return t
}

// Green path testing of ReverseSearch function in reversesearch package
func TestGreenPathsReverseSearchFile(t *testing.T) {
	// set fairly small StartBufLen for test context, so bugs are more likely
	// to be caught
	StartBufLen = 256

	// required for some tests' cleanUp functions
	origStartBufLen := StartBufLen

	// define tests (which will be iterated over further down)
	var tests = []struct {
		name               string         // test name (also description summary)
		filePath           string         // path of file to be searched
		searchCriteria     SearchCriteria // second parameter of ReverseSearch
		expectedExitStatus int            // i.e. the first return val of ReverseSearch
		expectedOutput     string         // file name (within testdata/test_logs) detailing log entries that should be matched
		setUp              func()         // set up actions
		cleanUp            func()         // clean up actions
	}{
		// test 1: regexp matches with no time constraints
		{
			name:     "test 1: regexp match",
			filePath: accessLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
				LeStartPattern: apacheStartPattern,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test1.txt",
		},

		// test 2: regexp that no log entries match
		{
			name:     "test 2: regexp unmatch",
			filePath: accessLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`nothing will match this regexp`,
				},
				LeStartPattern: apacheStartPattern,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test2.txt",
		},

		// test 3: matching regexp with "from" time constraint
		{
			name:     "test 3: fromTime constraint",
			filePath: accessLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
				FromTime:       parseTime(apacheTimeFormat, `23/Sep/2019:00:00:00 +0200`),
				LeStartPattern: apacheStartPattern,
				LeTimeFormat:   apacheTimeFormat,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test3.txt",
		},

		// test 4: matching regexp with "until" time constraint
		{
			name:     "test 4: untilTime constraint",
			filePath: accessLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
				UntilTime:      parseTime(apacheTimeFormat, `23/Sep/2019:00:00:00 +0200`),
				LeStartPattern: apacheStartPattern,
				LeTimeFormat:   apacheTimeFormat,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test4.txt",
		},

		// test 5: matching regexp within "from" and "until" time constraints
		{
			name:     "test 5: time range constraint",
			filePath: accessLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
				FromTime:       parseTime(apacheTimeFormat, `23/Sep/2019:00:00:00 +0200`),
				UntilTime:      parseTime(apacheTimeFormat, `23/Sep/2019:00:30:00 +0200`),
				LeStartPattern: apacheStartPattern,
				LeTimeFormat:   apacheTimeFormat,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test5.txt",
		},

		// test 6: no regexp matches within time constraints
		{
			name:     "test 6: regex unmatch in time constraints",
			filePath: accessLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
				UntilTime:      parseTime(apacheTimeFormat, `22/Sep/2019:01:00:00 +0200`),
				LeStartPattern: apacheStartPattern,
				LeTimeFormat:   apacheTimeFormat,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test6.txt",
		},

		// test 7: match multiline log entries
		{
			name:     "test 7: multiline match",
			filePath: odlLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`IAM-1010032`,
				},
				FromTime:       parseTime(odlTimeFormat, `Jun 17, 2010 11:00:00 PM IST`),
				LeStartPattern: odlStartPattern,
				LeTimeFormat:   odlTimeFormat,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test7.txt",
		},

		// test 8 - 14 focus on corner cases
		// test 8: match only first log entry
		{
			name:     "test 8: first log entry",
			filePath: odlLog,
			searchCriteria: SearchCriteria{
				UntilTime:      parseTime(odlTimeFormat, `Jun 15, 2010 2:01:21 AM IST`),
				LeStartPattern: odlStartPattern,
				LeTimeFormat:   odlTimeFormat,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test8.txt",
		},

		// test 9: match last log entry in file with fromTime
		{
			name:     "test 9: last log entry",
			filePath: odlLog,
			searchCriteria: SearchCriteria{
				FromTime:       parseTime(odlTimeFormat, `Jun 18, 2010 2:02:52 AM IST`),
				LeStartPattern: odlStartPattern,
				LeTimeFormat:   odlTimeFormat,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test9.txt",
		},

		// test 10: match the only multiline entry single-entry log file
		{
			name:     "test 10: single multiline log entry",
			filePath: singleLog,
			searchCriteria: SearchCriteria{
				FromTime:       parseTime(odlTimeFormat, `Jun 15, 2010 2:00:00 AM IST`),
				LeStartPattern: odlStartPattern,
				LeTimeFormat:   odlTimeFormat,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test10.txt",
		},

		// test 11: match the only single line entry in a single line log file
		{
			name:     "test 11: single multiline log entry",
			filePath: singleLineLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`oracle`,
				},
				LeStartPattern: odlStartPattern,
				LeTimeFormat:   odlTimeFormat,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test11.txt",
		},

		// test 12: process file that starts with \n
		{
			name:     `test 12: \n prefix`,
			filePath: accessLogNlPrefixUnix,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`41\.0\.2272\.96`,
				},
				LeStartPattern: apacheStartPattern,
				LeTimeFormat:   apacheTimeFormat,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test12.txt",
		},

		// test 13: process file that starts with \r\n
		{
			name:     `test 13: \r\n prefix`,
			filePath: accessLogNlPrefixWin,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`41\.0\.2272\.96`,
				},
				LeStartPattern: apacheStartPattern,
				LeTimeFormat:   apacheTimeFormat,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test13.txt",
		},

		// test 14: process file that does not end with \n
		{
			name:     `test 14: no \n suffix`,
			filePath: odlLogNoNlSuffixUnix,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`User Type does not exist!`,
				},
				LeStartPattern: odlStartPattern,
				LeTimeFormat:   odlTimeFormat,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test14.txt",
		},

		// test 15: process file that does not end with \r\n
		{
			name:     `test 15: no \r\n suffix`,
			filePath: odlLogNoNlSuffixWin,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`User Type does not exist!`,
				},
				LeStartPattern: odlStartPattern,
				LeTimeFormat:   odlTimeFormat,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test15.txt",
		},

		// test 16: process empty file
		{
			name:     "test 16: process empty file",
			filePath: emptyFile,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
				LeStartPattern: apacheStartPattern,
			},
			expectedExitStatus: 1,
			expectedOutput:     "test16.txt",
		},

		// test 17: process file that contains \n only
		{
			name:     `test 17: procses file that contains \n only`,
			filePath: nlOnlyUnix,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
				LeStartPattern: apacheStartPattern,
			},
			expectedExitStatus: 1,
			expectedOutput:     "test17.txt",
		},

		// test 18: process file that contains \r\n only
		{
			name:     `test 18: process file that contains \r\n only`,
			filePath: nlOnlyWin,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
				LeStartPattern: apacheStartPattern,
			},
			expectedExitStatus: 1,
			expectedOutput:     "test18.txt",
		},

		// test 19: test intricate logic around buf len increasing including when
		// buf offset is 0
		{
			name:     `test 19: test buf len increasing`,
			filePath: oneLargeOneSmall,
			searchCriteria: SearchCriteria{
				LeStartPattern: odlStartPattern,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test19.txt",
			setUp:              func() { StartBufLen = 1 },
			cleanUp:            func() { StartBufLen = origStartBufLen },
		},

		// test 20: StartBufLen is larger than the size of the file
		{
			name:     `test 20: StartBufLen is more than file size`,
			filePath: odlLog,
			searchCriteria: SearchCriteria{
				LeStartPattern: odlStartPattern,
			},
			expectedExitStatus: 0,
			expectedOutput:     "test20.txt",
			setUp:              func() { StartBufLen = 25000 },
			cleanUp:            func() { StartBufLen = origStartBufLen },
		},

		// test 21: StartBufLen is larger than the size of the file and the last
		// log entry match is after the first log entry
		{
			name:     `test 21: StartBufLen is more than file size, first entry no match`,
			filePath: odlLog,
			searchCriteria: SearchCriteria{
				LeStartPattern: odlStartPattern,
				LeTimeFormat:   odlTimeFormat,
				FromTime:       parseTime(odlTimeFormat, `Jun 17, 2010 11:00:00 PM IST`),
			},
			expectedExitStatus: 0,
			expectedOutput:     "test21.txt",
			setUp:              func() { StartBufLen = 25000 },
			cleanUp:            func() { StartBufLen = origStartBufLen },
		},
	}

	// outHandler will write log entries to outputBuffer as they're matched
	// so that we can compare the contents with the expected matches
	// for each test
	var outputBuffer bytes.Buffer
	outHandler := func(logEntry []byte) {
		logEntryStr := string(logEntry)

		// convert newlines to be same as they are in expected output files
		logEntryStr = strings.ReplaceAll(logEntryStr, "\r\n", "\n")

		// write to bytes buffer
		if outputBuffer.Len() > 0 {
			outputBuffer.WriteString("\n")
		}
		outputBuffer.WriteString(logEntryStr)
	}

	// iterate through tests
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outputBuffer.Reset()

			// run setUp function if specified
			if test.setUp != nil {
				test.setUp()
			}

			exitStatus, err := ReverseSearch(test.filePath, &test.searchCriteria,
				outHandler)
			if err != nil {
				t.Error(err)
				return
			}

			// run cleanUp function if specified
			if test.cleanUp != nil {
				test.cleanUp()
			}

			// retrieve expected output file contents into expectedOutput var
			expectedOutputRaw, err := ioutil.ReadFile(
				`./testdata/int_expected_output/` + test.expectedOutput,
			)
			check(err)
			expectedOutput := string(expectedOutputRaw)
			expectedOutput = strings.ReplaceAll(expectedOutput, "\r\n", "\n")

			// compare actual output against expected output
			if outputBuffer.String() != expectedOutput {
				// write actual results into
				// ./testdata/int_actual_output/${test.expectedOutput}
				err := ioutil.WriteFile(
					`./testdata/int_actual_output/`+test.expectedOutput,
					outputBuffer.Bytes(), 0644,
				)
				check(err)

				t.Errorf("Actual log entry matches did not match expected results.\n"+
					"Please compare ./testdata/int_output_actual/%s and "+
					"./testdata/int_output_expected/%s.", test.expectedOutput,
					test.expectedOutput)
			}

			// compare actual exit status to expected exist status
			if exitStatus != test.expectedExitStatus {
				t.Errorf("Got exit status: %d, want exit status: %d", exitStatus,
					test.expectedExitStatus)
			}
		})
	}
}

// Red path testing of ReverseSearchFile
func TestRedPathsReverseSearchFile(t *testing.T) {
	// set fairly small StartBufLen for test context, so bugs are more likely
	// to be caught
	StartBufLen = 256

	// required by some tests' cleanUp functions
	origMaxBufLen := MaxBufLen

	// define tests (which will be iterated over further down)
	var tests = []struct {
		name           string         // test name (also description summary)
		filePath       string         // path to log file
		searchCriteria SearchCriteria // searchCriteria ; 2nd param of ReverseSearch
		expectedErr    string         // string that should be contained within the error obj
		setUp          func()         // set up function
		cleanUp        func()         // clean up function
	}{
		// test 1: if time constraints exist, then date leTimeFormat must be specified
		// otherwise an error must be thrown
		{
			name:     "test 1: leTimeFormat must be set if time constraints exist",
			filePath: accessLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
				FromTime:       parseTime(apacheTimeFormat, `23/Sep/2019:00:00:00 +0200`),
				LeStartPattern: apacheStartPattern,
			},
			expectedErr: NoLeTimeFormat,
		},

		// test 2: leStartPattern must be specified; error to be thrown if not
		{
			name:     "test 2: leStartPattern must be set",
			filePath: accessLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
			},
			expectedErr: NoLeStartPattern,
		},

		// test 3: bad filePath
		{
			name:     "test 3: bad filepath",
			filePath: "./testing/logs/non_existent.log",
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
				LeStartPattern: apacheStartPattern,
				LeTimeFormat:   apacheTimeFormat,
			},
			expectedErr: BadFilePath,
		},

		// test 4: bad regexps
		{
			name:     "test 4: bad regexps",
			filePath: accessLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
					`~((?<=xxx)~`,
				},
				LeStartPattern: apacheStartPattern,
				LeTimeFormat:   apacheTimeFormat,
			},
			expectedErr: BadRegexps,
		},

		// test 5: fromTime is after untilTime
		{
			name:     "test 5: fromTime after untilTime",
			filePath: accessLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
				FromTime:       parseTime(apacheTimeFormat, `23/Sep/2019:10:00:00 +0200`),
				UntilTime:      parseTime(apacheTimeFormat, `22/Sep/2019:23:00:00 +0200`),
				LeStartPattern: apacheStartPattern,
				LeTimeFormat:   apacheTimeFormat,
			},
			expectedErr: FromTimeAfterUntilTime,
		},

		// test 6: erroneous regex for LeStartPattern (should get regex compilation
		// error)
		{
			name:     "test 6: bad regex for leStartPattern",
			filePath: accessLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
				LeStartPattern: `~((?<=xxx)~`,
				LeTimeFormat:   apacheTimeFormat,
			},
			expectedErr: BadLeStartPattern,
		},

		// test 7: wrong (unmatching) LeStartPattern for a small file.
		// error saying no log entries found
		{
			name:     "test 7: bad leStartPattern for small file",
			filePath: odlLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`No OES Policy found`,
				},
				LeStartPattern: odlStartPattern + `unmatch$`,
				LeTimeFormat:   odlTimeFormat,
			},
			expectedErr: NoLogEntriesInFile,
		},

		// test 8: A file with no matching log entries beyond a certain point
		// should throw an error
		{
			name:     "test 8: No more log entries",
			filePath: accessLogNoMoreEntries,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
				LeStartPattern: apacheStartPattern,
				LeTimeFormat:   apacheTimeFormat,
			},
			expectedErr: NoMoreLogEntries,
		},

		// test 9: leTimeFormat mismatch should cause error
		{
			name:     "test 9: leTimeFormat mismatch",
			filePath: accessLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
				FromTime:       parseTime(apacheTimeFormat, `23/Sep/2019:00:00:00 +0200`),
				LeStartPattern: apacheStartPattern,
				LeTimeFormat:   apacheTimeFormat + `unmatch`,
			},
			expectedErr: LeTimeFormatMismatch,
		},

		// test 10: wrong (unmatching) leStartPattern for a big file. Should receive
		// a max buf length reached error
		{
			name:     "test 10: bad leStartPattern for big file",
			filePath: accessLog,
			searchCriteria: SearchCriteria{
				Regexps: []string{
					`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
				},
				LeStartPattern: apacheStartPattern + ` unmatch$`,
				LeTimeFormat:   apacheTimeFormat,
			},
			expectedErr: MaxBufLenReached,
			setUp:       func() { MaxBufLen = 5000 },
			cleanUp:     func() { MaxBufLen = origMaxBufLen },
		},

		// test 11: A log entry with length exceding that of MaxBufLen in a log file
		// should throw MaxBufLen reached error
		{
			name:     "test 11: log entry length excedes MaxBufLen",
			filePath: longLineLog,
			searchCriteria: SearchCriteria{
				LeStartPattern: odlStartPattern,
				LeTimeFormat:   odlTimeFormat,
			},
			expectedErr: MaxBufLenReached,
			setUp:       func() { MaxBufLen = 5000 },
			cleanUp:     func() { MaxBufLen = origMaxBufLen },
		},
	}

	// iterate through tests
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// run setUp function if specified
			if test.setUp != nil {
				test.setUp()
			}

			exitStatus, err := ReverseSearch(test.filePath, &test.searchCriteria,
				func(logEntry []byte) {})

			// run cleanUp function if specified
			if test.cleanUp != nil {
				test.cleanUp()
			}

			// compare actual error returned with expected error
			if err == nil {
				t.Error("No error returned")
			} else if !strings.Contains(err.Error(), test.expectedErr) {
				t.Errorf("Got error: \"%s\", want error that contains: \"%s\"",
					err.Error(), test.expectedErr)
			}

			// compare actual exit status to expected exit status
			if exitStatus != -1 {
				t.Errorf("Got exit status: %d, want exit status: -1", exitStatus)
			}
		})
	}
}
