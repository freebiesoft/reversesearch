package main

// there are multiple examples in this file (please refer to main function)
// please configure the "logsDir" variable if required (the working directory
// can change depending on IDE and other factors)

import (
	"encoding/json"
	"fmt"
	"github.com/freebiesoft/reversesearch"
	"strings"
	"time"
)

// it is assumed that parent directory (i.e. reversesearch) is the working directory
// in this default definition
var logsDir = `.\testdata\test_logs\`

// apache access log patterns
var apacheStartPattern = `^(?:\S+) (?:\S+) (?:\S+) \[([\w:/]+\s[+\-]\d{4})\]`
var apacheTimeFormat = `02/Jan/2006:15:04:05 -0700`

// ODL log patterns
var odlStartPattern = `^<(\w{3} \d{2}, \d{4} \d{1,2}:\d{2}:\d{2} (?:AM|PM) (?:\S+))>`
var odlTimeFormat = `Jan 2, 2006 3:04:05 PM MST`

// utility method for parsing time objects (useful for inline definitions in structs)
func parseTime(format string, timeStr string) time.Time {
	t, err := time.Parse(format, timeStr)
	if err != nil {
		panic(err)
	}
	return t
}

func main() {
	// file paths
	accessLog := logsDir + `access.log`
	odlLog := logsDir + `odl.log`

	// declare variables that'll be used throughout the examples
	var searchCriteria reversesearch.SearchCriteria
	var outputHandler reversesearch.OutputHandler

	/*********************************************************************/
	/************** EXAMPLE 1: VERBOSE WALKTHROUGH ***********************/
	/*********************************************************************/
	// this example will walk you through all the search criteria fields
	// in detail
	fmt.Println("EXAMPLE 1\n======================================")

	// define search criteria struct
	searchCriteria = reversesearch.SearchCriteria{
		// Take some time to notice how apacheStartPattern has been defined. The
		// timestamp has been encapsulated in a capturing group, but nothing else has.
		// There must be 1 (and only 1) capturing group defined which must capture
		// the timestamp of the log entry.
		LeStartPattern: apacheStartPattern,

		// The LeTimeFormat field will be used to parse the timestamp captured from
		// the regular expression defined in LeStartPattern to a time.Time struct,
		// when processing log etnries, so it can be determined if they match the
		// FromTime and UntilTime constraints if specified.
		LeTimeFormat: apacheTimeFormat,

		// FromTime and UntilTime fields specify time constraints that log entries
		// must match in order to be considered for matching.
		// It is not necessary to specify both FromTime and UntilTime, one or the
		// other can be ommitted, or even both.
		// When the first log entry during the reverse traversal of the log file
		// that fails the FromTime constraint is discovered, the search process will
		// terminate, thus saving time in large files.
		FromTime:  parseTime(apacheTimeFormat, `23/Sep/2019:00:00:00 +0200`),
		UntilTime: parseTime(apacheTimeFormat, `23/Sep/2019:00:30:00 +0200`),

		// This will be a list of regexes that will be tested against log entries
		// that satisfy the specified time constraints. Log entries that match both
		// the tiem constraints and all the regular expressions defined here are
		// considered a match and will be passed to the outputHandler (more on this
		// further down). It is not imperative to include a Regexps field and if left
		// blank, all log entries that match time constraints will be considered a match.
		Regexps: []string{
			`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
		},
	}

	// pass log file, and searchCriteria struct to the ReverseSearch function.
	// Note that when the outputHandler parameter is set as nil, the default
	// output handler will be used which is effectively to print matching log entries
	// to stdout as they are found.
	_, err := reversesearch.ReverseSearch(accessLog, &searchCriteria, nil)
	if err != nil {
		panic(err)
	}

	/*********************************************************************/
	/************** EXAMPLE 2: MULTILINE LOG ENTRIES *********************/
	/*********************************************************************/
	// ReverseSearch handles multiline log entries seamlessly; it works in
	// exactly the same way as with single line log entres (i.e. example 1)
	fmt.Println("\n\nEXAMPLE 2\n======================================")
	// define search criteria
	searchCriteria = reversesearch.SearchCriteria{
		LeStartPattern: odlStartPattern,
		LeTimeFormat:   odlTimeFormat,
		Regexps: []string{
			// in the test log file, odlLog, 'IAM-1010032' and 'blahblah' are
			// on separate lines; this is matching over multiple lines
			`(?s)IAM-1010032.*blahblah`,
		},
		FromTime: parseTime(odlTimeFormat, `Jun 17, 2010 11:00:00 PM IST`),
	}
	_, err = reversesearch.ReverseSearch(odlLog, &searchCriteria, nil)
	if err != nil {
		panic(err)
	}

	/*********************************************************************/
	/************** EXAMPLE 3: CUSTOM OUTPUT HANDLER *********************/
	/*********************************************************************/
	// You may define custom output handlers, rather than just having
	// log entries printed to STDOUT as they're matched. In this example
	// we create a custom output handler that encapsulates log entry
	// matches into a JSON string
	fmt.Println("\n\nEXAMPLE 3\n======================================")

	// define search criteria struct
	searchCriteria = reversesearch.SearchCriteria{
		LeStartPattern: apacheStartPattern,
		LeTimeFormat:   apacheTimeFormat,
		FromTime:       parseTime(apacheTimeFormat, `23/Sep/2019:00:00:00 +0200`),
		UntilTime:      parseTime(apacheTimeFormat, `23/Sep/2019:00:30:00 +0200`),
		Regexps: []string{
			`/modules/mod_araticlhess1/mod_araticlhess1\.php`,
		},
	}

	// declare/initialise json string
	var jsonString string
	jsonString = `{ "matches" : `

	// declare string slice for log entry matches that the output handler can
	// append to
	logEntryMatches := []string{}

	// define custom output handler
	outputHandler = func(logEntry []byte) {
		logEntryStr := string(logEntry)
		logEntryMatches = append(logEntryMatches, logEntryStr)
	}

	// pass log file, search criteria, and our custom output handler to ReverseSearch
	_, err = reversesearch.ReverseSearch(accessLog, &searchCriteria, outputHandler)
	if err != nil {
		panic(err)
	}

	// create the rest of the json string and output to console
	matchesB, err := json.Marshal(logEntryMatches)
	if err != nil {
		panic(err)
	}
	jsonString = jsonString + string(matchesB) + ` }`
	fmt.Println(jsonString)

	/*********************************************************************/
	/************** EXAMPLE 4: CUSTOM MATCH LOGIC ************************/
	/*********************************************************************/
	// If you don't want to use the standard log entry matching mechanics,
	// i.e. to specify regular expressions that all log entries must match,
	// then you can omit the Regexps field in the search criteria struct
	// and specify your own match logic within the output handler. In
	// this example we specify our own match logic, in which matching
	// log entries are defined as those that contain any 2 out of 3
	// keywords.
	fmt.Println("\n\nEXAMPLE 4\n======================================")

	// define search criteria (notice how we've ommitted Regexps field)
	searchCriteria = reversesearch.SearchCriteria{
		LeStartPattern: odlStartPattern,
		LeTimeFormat:   odlTimeFormat,
		FromTime:       parseTime(odlTimeFormat, `Jun 15, 2010 00:00:00 AM IST`),
		UntilTime:      parseTime(odlTimeFormat, `Jun 19, 2010 00:00:00 AM IST`),
	}

	// define keywords that we want at least two of in matching log entries
	keywords := []string{
		"<No OES Policy found for the given Action.>",
		"<blahblah>",
		"<IAM-1010232>",
	}

	// define custom output handler with custom match logic
	outputHandler = func(logEntry []byte) {
		matches := 0
		logEntryStr := string(logEntry)
		for _, str := range keywords {
			if strings.Contains(logEntryStr, str) {
				matches++
			}
		}
		if matches >= 2 {
			fmt.Println(logEntryStr)
		}
	}

	// pass log file, search criteria, and our custom output handler to ReverseSearch
	_, err = reversesearch.ReverseSearch(odlLog, &searchCriteria, outputHandler)
	if err != nil {
		panic(err)
	}
}
