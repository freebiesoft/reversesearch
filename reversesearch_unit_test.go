package reversesearch

/* In this file is the internal testing of the reversesearch package, i.e. we
test the non-exported functions. ReverseSearch function is not tested in this
package since it is tested thoroughly in the reversesearch_test package with the
integration testing. */

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// apache access log patterns
var apacheStartPattern = `^(?:\S+) (?:\S+) (?:\S+) \[([\w:/]+\s[+\-]\d{4})\]`
var apacheTimeFormat = `02/Jan/2006:15:04:05 -0700`

// ODL log patterns
var odlStartPattern = `^<(\w{3} \d{2}, \d{4} \d{1,2}:\d{2}:\d{2} (?:AM|PM) (?:\S+))>`
var odlTimeFormat = `Jan 2, 2006 3:04:05 PM MST`

// TestFindLogEntries tests the findLogEntries function in reversesearch.
// Both green and red path testing (bunched them together for convenience
// and because there's only 1 red path test). There are two phases of tests in
// this function; the first phase of tests will be general usage of the function
// and the second phase will focus on findLogEntries' capability of interpreting
// newlines and log entries in the buf parameter
func TestFindLogEntries(t *testing.T) {
	/*********************************************************************/
	/************** PHASE 1: GENERAL USAGE *******************************/
	/*********************************************************************/
	// The first phase of testing consists of 3 tests; 2 greenpath tests
	// phase one redpath test. These are general usage tests, and there
	// arent many because the integration tests in the reversesearch_test
	// package indrectly does general testing of findLogEntries.

	// define test lines for general tests
	line1 := "<Jun 16, 2010 6:02:02 AM IST> <Warning>"
	line2 := "<No OES Policy found for keyword1 the given Action.>"
	line3 := "<Jun 17, 2010 11:02:52 PM IST> <Error> keyword2"
	line4 := "<Jun 18, 2010 2:02:02 AM IST> <Warning> keyword1"

	// create 2 test buffers, first one is designed to be used for bOffset > 0,
	// and the second one is designed to be used for bOffset == 0
	testBuf1 := []byte("XXX" + "\n" + line1 + "\n" + line2 + "\n" + line3 + "\n" + line4)
	testBuf2 := []byte(line1 + "\n" + line2 + "\n" + line3 + "\n" + line4)

	// these tests focus on generic usage of the findLogEntries function
	// the tests are mostly the same, we just want to adjust a few things between
	// them such as buf, buf offset, and expected results
	tests := []struct {
		name              string
		buf               []byte // first parameter
		bOffset           int64  // second parameter
		expectedOutput    string // expected matching log entries
		expectedLastLePos int    // expected first return value
		expectedLastNlPos int    // expected second return value
		expectedAbort     bool   // expected third return value
		expectedErr       string // expected fourth return value
	}{
		{
			"test 1: general usage, bOffset = 10", testBuf1, 10,

			// expected results
			line1 + "\n" + line2, 3, 3, false, "",
		},
		{
			"test 2: general usage, bOffset = 0", testBuf2, 0,

			// expected results
			line1 + "\n" + line2, 0, 0, false, "",
		},
		{
			"test 3: redpath test, bOffset = -1", testBuf2, -1,

			// expected results
			"", len(testBuf2), len(testBuf2), false, BufOffsetLessThanZero,
		},
	}

	// define other test parameters to be used with calling findLogEntries
	testLeStartRegexp := compileRegexp(odlStartPattern)                       // 5th test parameter
	testLeTimeFormat := odlTimeFormat                                         // 6th test parameter
	testFromTime := parseTime(odlTimeFormat, `Jun 16, 2010 6:00:00 AM IST`)   // 7th test param
	testUntilTime := parseTime(odlTimeFormat, `Jun 17, 2010 11:30:52 PM IST`) // 8th test param
	testRegexps := compileRegexps([]string{`keyword1`})                       // 9th test param

	// when findLogEntries invokes testOutputHandler (which will happen in the event
	// of a logEntry match being found), it will append the matching log entry
	// to the "output" variable
	output := ""
	testOutputHandler := func(logEntry []byte) {
		if len(output) > 0 {
			output += "\n"
		}
		output += string(logEntry)
	}

	// iterate through the 3 tests defined above
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output = "" // reset output

			// execute test call
			lastLePos, lastNlPos, abort, err := findLogEntries(test.buf, test.bOffset,
				len(test.buf)-1, len(test.buf), testLeStartRegexp, testLeTimeFormat,
				testFromTime, testUntilTime, testRegexps, testOutputHandler)

			// compare output against expected output
			if output != test.expectedOutput {
				t.Errorf("Actual log entry matches did not match as expected.\n"+
					"GOT:\n%s\nWANT:\n%s",
					output, test.expectedOutput)
			}

			// compare lastLePos to expectedLastLePos (1st return val)
			if lastLePos != test.expectedLastLePos {
				t.Errorf("lastLePos does not match expectedLastLePos. Got %d, want %d.",
					lastLePos, test.expectedLastLePos)
			}

			// compare lastNlPos to expectedLastNlPos (2nd return val)
			if lastNlPos != test.expectedLastNlPos {
				t.Errorf("lastNlPos does not match expectedLastNlPos. Got %d, Want %d.",
					lastNlPos, test.expectedLastNlPos)
			}

			// compare abort to expectedAbort (3rd return val)
			if abort != test.expectedAbort {
				t.Errorf("abort does not match expectedAbort. Got %t, want %t.",
					abort, test.expectedAbort)
			}

			// compare err with expectedErr (4th return val)
			if err != nil && test.expectedErr == "" {
				t.Error(err)
				return
			}
			actualErr := ""
			if err != nil {
				actualErr = err.Error()
			}
			if !strings.Contains(actualErr, test.expectedErr) {
				t.Errorf("Got error: \"%s\", Want error: \"%s\"", actualErr,
					test.expectedErr)
			}
		})
	}

	/*********************************************************************/
	/********** PHASE 2: NEWLINE AND LOG ENTRY INTERPRETATION ************/
	/*********************************************************************/
	// These tests focus around find_log_entries' capability to interpret
	// log entries within buf param, and that it interprets and deals with
	// newline characters correctly.

	// handy for specifying default values for 3rd and 4th parameters
	// (to save us the efforts of calculating len(buf))
	defaultVal := -1

	// All tests as part of this will use the same leTimeFormat, leStartPattern,
	// have no time constraints, and no regexps, since variation of these params
	// have no part to play in log entry and newline interpretation. They will
	// also all return abort as false (because of no time constraints).
	var leInterpretationTests = []struct {
		buf               string // 1st parameter (in string form for definition convenience)
		bOffset           int64  // 2nd parameter of findLogEntries
		scanToPos         int    // 3rd parameter of findLogEntries
		lastNlPos         int    // 4th parameter of findLogEntries
		expectedLastLePos int    // 1st expected return value from findLogEntries
		expectedLastNlPos int    // 2nd expected return value from findLogEntries
		expectedLeCount   int    // expected number of log entries to be detected in buf
	}{
		// check a random string with no newlines
		{"some string", 10, defaultVal, defaultVal, 11, 11, 0},

		// check a random string with newlines
		{"some string\nsome string\n", 10, defaultVal, defaultVal, 24, 11, 0},

		// check LE with no NL
		{"<LE Start>", 10, defaultVal, defaultVal, 10, 10, 0},

		// check multiline LE with no NL prefix
		{"<LE Start> line1\nline2", 10, defaultVal, defaultVal, 22, 16, 0},

		// \n as first pos is ignored
		{"\n<LE Start> line1", 10, defaultVal, defaultVal, 17, 17, 0},

		// \n as first pos is not ignored
		{"x\n<LE Start> line2", 10, defaultVal, defaultVal, 1, 1, 1},

		// \n as last position
		{"<LE Start>\n", 10, defaultVal, defaultVal, 11, 10, 0},

		// \r\n as first position
		{"\r\n<LE Start>", 10, defaultVal, defaultVal, 0, 0, 1},

		// \r\n as 2nd last position
		{"<LE Start>\r\n", 10, defaultVal, defaultVal, 12, 10, 0},

		// check CR has no effect
		{"x\r<LE Start>", 10, defaultVal, defaultVal, 12, 12, 0},

		// \n as second position followed by multiline (\n) LE
		{"x\n<LE Start> line1\ntest line2\ntest line3\n", 10, defaultVal, defaultVal, 1, 1, 1},

		// \r\n at first position followed by multiline (\r\n) LE
		{"\r\n<LE Start> line2\ntest line2\ntest line3\n", 10, defaultVal, defaultVal, 0, 0, 1},

		// \n as first position followed by multiple LEs
		{"\n<LE Start> line1\ntest line2\ntest line3\n" +
			"<LE Start>\ntest line1\ntest line2\n" +
			"<LE Start> single line log entry", 10, defaultVal, defaultVal, 39, 17, 2},

		// \r\n as first position followed by multiple LEs
		{"\r\n<LE Start> line1\r\ntest line2\r\ntest line3\r\n" +
			"<LE Start>\r\ntest line1\r\ntest line2\r\n" +
			"<LE Start> single line log entry", 10, defaultVal, defaultVal, 0, 0, 3},

		// multiple LEs prefixed by random string
		{"some string\n<LE Start> line1\ntest line2\ntest line3\n" +
			"<LE Start>\ntest line1\ntest line2\n" +
			"<LE Start> single line log entry", 10, defaultVal, defaultVal, 11, 11, 3},

		// offset = 0, no NLs or LEs
		{"some string", 0, defaultVal, defaultVal, 11, 0, 0},

		// offset == 0, random string prefixed with \n
		{"\nsome string", 0, defaultVal, defaultVal, 12, 0, 0},

		// offset == 0, random string prefixed with \r\n
		{"\r\nsome string", 0, defaultVal, defaultVal, 13, 0, 0},

		// offset = 0, LE with no newline prefixes
		{"<LE Start> single line log entry", 0, defaultVal, defaultVal, 0, 0, 1},

		// offset = 0, \n as first position followed by LE
		{"\n<LE Start> single line log entry", 0, defaultVal, defaultVal, 0, 0, 1},

		// offset = 0, \r\n as first position followed by LE
		{"\r\n<LE Start> single line log entry", 0, defaultVal, defaultVal, 0, 0, 1},

		// offset = 0, multiple log entries
		{"<LE Start> line1\ntest line2\ntest line3\n" +
			"<LE Start>\ntest line1\ntest line2\n" +
			"<LE Start> single line log entry", 0, defaultVal, defaultVal, 0, 0, 3},

		// offset = 0, \n as first and second positions, followed by LE
		{"\n\n<LE Start> single line log entry", 0, defaultVal, defaultVal, 1, 0, 1},

		// offset = 0, \r\n as first and third positions, followed by LE
		{"\r\n\r\n<LE Start> single line log entry", 0, defaultVal, defaultVal, 2, 0, 1},

		// offset = 0, random string followed by LE
		{"some string\n<LE Start> single line log entry", 0, defaultVal, defaultVal, 11, 0, 1},

		// offset = 0, random string prefixed with \n followed by LE
		{"\nsome string\n<LE Start> single line log entry", 0, defaultVal, defaultVal, 12, 0, 1},

		// the following tests focus around files that contain a few characters
		// and newlines that could cause problems for the intricate logic at work
		{"X", 0, defaultVal, defaultVal, 1, 0, 0},           // 1 char
		{"XXX", 0, defaultVal, defaultVal, 3, 0, 0},         // random string
		{"\n", 0, defaultVal, defaultVal, 1, 0, 0},          // 1 newline (Unix)
		{"\n\n", 0, defaultVal, defaultVal, 2, 0, 0},        // 2 newlines (Unix)
		{"\nX", 0, defaultVal, defaultVal, 2, 0, 0},         // 1 newline followed by 1 char (Unix)
		{"\n\nX", 0, defaultVal, defaultVal, 3, 0, 0},       // 2 newlies followed by 1 char (Unix)
		{"\nXXX", 0, defaultVal, defaultVal, 4, 0, 0},       // 1 newline followed by random string (Unix)
		{"\n\nXXX", 0, defaultVal, defaultVal, 5, 0, 0},     // 2 newlines followed by random string (Unix)
		{"\nXXX\n", 0, defaultVal, defaultVal, 5, 0, 0},     // newline followed by random string and newline (Unix)
		{"\r\n", 0, defaultVal, defaultVal, 2, 0, 0},        // 1 newline (Win)
		{"\r\n\r\n", 0, defaultVal, defaultVal, 4, 0, 0},    // 2 newlines (Win)
		{"\r\nX", 0, defaultVal, defaultVal, 3, 0, 0},       // 1 newline followed by 1 char (Win)
		{"\r\n\r\nX", 0, defaultVal, defaultVal, 5, 0, 0},   // 2 newlies followed by 1 char (Win)
		{"\r\nXXX", 0, defaultVal, defaultVal, 5, 0, 0},     // 1 newline followed by random string (Win)
		{"\r\n\r\nXXX", 0, defaultVal, defaultVal, 7, 0, 0}, // 2 newlines followed by random string (Win)
		{"\r\nXXX\r\n", 0, defaultVal, defaultVal, 7, 0, 0}, // newline followed by random string and newline (Win)

		// check multi-byte utf-8 character (£ in this test)
		{"test line£", 10, defaultVal, defaultVal, 11, 11, 0},

		// the following tests focus on the scanFrom and lastNlPos parameters which
		// were added to improve efficiency
		{"<LE Start>\ntest line1\ntest line1", 10, 11, 21, 32, 10, 0},              // check lastNlPos is acknowledged as 10 (Unix)
		{"<LE Start>\ntest line2\ntest line2", 10, 10, 21, 32, 10, 0},              // check lastNlPos is acknowledged as 10 (Unix)
		{"<LE Start>\ntest line3\ntest line3", 10, 9, 21, 32, 10, 0},               // check lastNlPos is acknowledged as 10 (Unix)
		{"<LE Start>\ntest line4\ntest line4", 10, 8, 10, 32, 10, 0},               // check lastNlPos is acknowledged as 10 (Unix)
		{"<LE Start>\ntest line5\ntest line5", 10, 8, 21, 32, 21, 0},               // check lastNlPos is acknowledged as 21 (Unix)
		{"some string\n<LE Start>\ntest line1\ntest line1", 10, 11, 22, 11, 11, 1}, // check LE does get acknowledged (Unix)
		{"some string\n<LE Start>\ntest line1\ntest line1", 10, 10, 22, 11, 11, 1}, // check LE does get acknowledged (Unix)
		{"some string\n<LE Start>\ntest line1\ntest line1", 10, 9, 11, 44, 11, 0},  // check LE doesn't get acknowledged (Unix)
		{"some string\n<LE Start>\ntest line1\ntest line1", 10, 9, 22, 44, 22, 0},  // check LE doesn't get acknowledged (Unix)
		{"\n<LE Start>\ntest line6\ntest line6", 10, 1, 11, 33, 11, 0},             // check LE and first NL don't get acknowledged (Unix)
		{"\n<LE Start>\ntest line7\ntest line7", 10, 0, 11, 33, 11, 0},             // check LE and first NL don't get acknowledged (Unix)
		{"\n<LE Start>\ntest line8\ntest line8", 0, 1, 11, 0, 0, 1},                // check LE is acknowledged (offset = 0) (Unix)
		{"\n<LE Start>\ntest line9\ntest line9", 0, 0, 11, 0, 0, 1},                // check LE is acknowledged (offset = 0) (Unix)
		// and same for windows ...
		{"<LE Start>\r\ntest lineA\r\ntest lineA", 10, 12, 22, 34, 10, 0},                // check lastNlPos is acknowledged as 10 (Win)
		{"<LE Start>\r\ntest lineB\r\ntest lineB", 10, 11, 22, 34, 10, 0},                // check lastNlPos is acknowledged as 10 (Win)
		{"<LE Start>\r\ntest lineC\r\ntest lineC", 10, 10, 22, 34, 10, 0},                // check lastNlPos is acknowledged as 10 (Win)
		{"<LE Start>\r\ntest lineD\r\ntest lineD", 10, 9, 10, 34, 10, 0},                 // check lastNlPos is acknowledged as 10 (Win)
		{"<LE Start>\r\ntest lineE\r\ntest lineE", 10, 9, 22, 34, 22, 0},                 // check lastNlPos is acknowledged as 22 (Win)
		{"some string\r\n<LE Start>\r\ntest line1\r\ntest line1", 10, 12, 23, 11, 11, 1}, // check LE does get acknowledged (Win)
		{"some string\r\n<LE Start>\r\ntest line1\r\ntest line1", 10, 11, 23, 11, 11, 1}, // check LE does get acknowledged (Win)
		{"some string\r\n<LE Start>\r\ntest line1\r\ntest line1", 10, 10, 11, 47, 11, 0}, // check LE doesn't get acknowledged (Win)
		{"some string\r\n<LE Start>\r\ntest line1\r\ntest line1", 10, 10, 23, 47, 23, 0}, // check LE doesn't get acknowledged (Win)
		{"\r\n<LE Start>\r\ntest lineF\r\ntest lineF", 10, 2, 12, 0, 0, 1},               // check LE and first NL do get acknowledged (Win)
		{"\r\n<LE Start>\r\ntest lineG\r\ntest lineG", 10, 1, 12, 0, 0, 1},               // check LE and first NL do get acknowledged (Win)
		{"\r\n<LE Start>\r\ntest lineH\r\ntest lineH", 10, 0, 12, 0, 0, 1},               // check LE and first NL do get acknowledged (Win)
		{"\r\n<LE Start>\r\ntest lineI\r\ntest lineI", 0, 2, 12, 0, 0, 1},                // check LE is acknowledged (offset = 0) (Win)
		{"\r\n<LE Start>\r\ntest lineJ\r\ntest lineJ", 0, 1, 12, 0, 0, 1},                // check LE is acknowledged (offset = 0) (Win)
		{"\r\n<LE Start>\r\ntest lineK\r\ntest lineK", 0, 0, 12, 0, 0, 1},                // check LE is acknowledged (offset = 0) (Win)
		// offset = 0, no newlines at beginning
		{"<LE Start>\ntest lineL\ntest lineL", 0, 4, 10, 0, 0, 1}, // check LE is acknowledged (offset = 0)
		{"<LE Start>\ntest lineM\ntest lineM", 0, 1, 10, 0, 0, 1}, // check LE is acknowledged (offset = 0)
		{"<LE Start>\ntest lineN\ntest lineN", 0, 0, 10, 0, 0, 1}, // check LE is acknowledged (offset = 0)
		{"<LE Start>\ntest lineO\ntest lineO", 0, 8, 21, 0, 0, 1}, // check LE is acknowledged (offset = 0)
	}

	// when findLogEntries calls testOutputHandler (which will happen in the event
	// of a log entry match being found), it will increment the leCount variable
	leCount := 0
	testOutputHandler = func(logEntry []byte) {
		leCount++
	}

	// define 5th, 6th, 7th, 8th, 9th test parameters which will remain the same
	// for all tests defined in leInterpretationTests
	testLeStartRegexp = compileRegexp(`^<LE Start>`)
	testLeTimeFormat = ""
	testFromTime = time.Time{}
	testUntilTime = time.Time{}
	testRegexps = nil

	// iterate through tests
	for i, test := range leInterpretationTests {
		testName := fmt.Sprintf("test %d", i+1)
		t.Run(testName, func(t *testing.T) {
			// reset LE counter
			leCount = 0

			// substitute defaultVal for scanToPos and lastNlPos if set as such
			scanToPosParam := test.scanToPos
			lastNlPosParam := test.lastNlPos
			if scanToPosParam == defaultVal {
				scanToPosParam = len(test.buf) - 1
			}
			if lastNlPosParam == defaultVal {
				lastNlPosParam = len(test.buf)
			}

			// execute call to findLogEntries
			lastLePos, lastNlPos, abort, err := findLogEntries([]byte(test.buf), test.bOffset,
				scanToPosParam, lastNlPosParam, testLeStartRegexp, testLeTimeFormat, testFromTime,
				testUntilTime, testRegexps, testOutputHandler)
			if err != nil {
				t.Error(err)
				return
			}

			// compare lastLePos to expectedLastLePos (1st return value)
			if lastLePos != test.expectedLastLePos {
				t.Errorf("lastLePos does not match expectedLastLePos. Got %d, want %d.",
					lastLePos, test.expectedLastLePos)
			}

			// compare lastNlPos to expectedLastNlPos (2nd return value)
			if lastNlPos != test.expectedLastNlPos {
				t.Errorf("lastNlPos does not match expectedLastNlPos. Got %d, want %d.",
					lastNlPos, test.expectedLastNlPos)
			}

			// compare abort to expectedAbort (3rd return value)
			if abort {
				t.Errorf("abort should be false but it was returned as true")
			}

			// compare leCount to expectedLeCount (i.e. how many times findLogEntries
			// invoked the output handler)
			if leCount != test.expectedLeCount {
				t.Errorf("leCount does not match expectedLeCount. Got %d, want %d.",
					leCount, test.expectedLeCount)
			}
		})
	}
}

// test processLine method (both greenpath and redpath)
func TestProcessLine(t *testing.T) {
	// test parameter for "line" (1st parameter); testLine is used for all except
	// one test that uses testLineNoLeStart
	testLine := `<Jun 15, 2010 2:02:02 AM IST> <Warning> <oracle.iam>`
	testLineNoLeStart := `<Warning> <oracle.iam>`

	// define tests
	var tests = []struct {
		name           string    // test name/summary
		line           string    // 1st input param
		leStartPattern string    // 2nd input param
		leTimeFormat   string    // 3rd input param
		fromTime       time.Time // 4th input param
		untilTime      time.Time // 5th input param

		// expected return values
		expectedStartOfLe          bool   // expected value of 1st return value
		expectedFromTimeSatisfied  bool   // expected value of 2nd return value
		expectedUntilTimeSatisfied bool   // expected value of 3rd return value
		expectedErr                string // expected error (leave blank if expcting none)
	}{
		// line matches leStartPattern, no time constraints
		{
			name:                       "test: leStartPattern matches, no time constraints",
			line:                       testLine,
			leStartPattern:             odlStartPattern,
			expectedStartOfLe:          true,
			expectedFromTimeSatisfied:  true,
			expectedUntilTimeSatisfied: true,
		},

		// line matches leStartPattern and passes fromTime constraint
		{
			name:                       "test: leStartPattern matches, passes fromTime",
			line:                       testLine,
			leStartPattern:             odlStartPattern,
			leTimeFormat:               odlTimeFormat,
			fromTime:                   parseTime(odlTimeFormat, `Jun 15, 2010 2:00:00 AM IST`),
			expectedStartOfLe:          true,
			expectedFromTimeSatisfied:  true,
			expectedUntilTimeSatisfied: true,
		},

		// line matches leStartPattern and fails fromTime constraint
		{
			name:                       "test: leStartPattern matches, fails fromTime",
			line:                       testLine,
			leStartPattern:             odlStartPattern,
			leTimeFormat:               odlTimeFormat,
			fromTime:                   parseTime(odlTimeFormat, `Jun 15, 2010 3:00:00 AM IST`),
			expectedStartOfLe:          true,
			expectedFromTimeSatisfied:  false,
			expectedUntilTimeSatisfied: true,
		},

		// line matches leStartPattern and passes untilTime
		{
			name:                       "test: leStartPattern matches, passes untilTime",
			line:                       testLine,
			leStartPattern:             odlStartPattern,
			leTimeFormat:               odlTimeFormat,
			untilTime:                  parseTime(odlTimeFormat, `Jun 15, 2010 3:00:00 AM IST`),
			expectedStartOfLe:          true,
			expectedFromTimeSatisfied:  true,
			expectedUntilTimeSatisfied: true,
		},

		// line matches leStartPattern and fails untilTime
		{
			name:                       "test: leStartPattern matches, fails untilTime",
			line:                       testLine,
			leStartPattern:             odlStartPattern,
			leTimeFormat:               odlTimeFormat,
			untilTime:                  parseTime(odlTimeFormat, `Jun 15, 2010 2:00:00 AM IST`),
			expectedStartOfLe:          true,
			expectedFromTimeSatisfied:  true,
			expectedUntilTimeSatisfied: false,
		},

		// line matches leStartPattern and passes fromTime and untilTime
		{
			name:                       "test: leStartPattern matches, passes fromTime & untilTime",
			line:                       testLine,
			leStartPattern:             odlStartPattern,
			leTimeFormat:               odlTimeFormat,
			fromTime:                   parseTime(odlTimeFormat, `Jun 15, 2010 2:00:00 AM IST`),
			untilTime:                  parseTime(odlTimeFormat, `Jun 15, 2010 3:00:00 AM IST`),
			expectedStartOfLe:          true,
			expectedFromTimeSatisfied:  true,
			expectedUntilTimeSatisfied: true,
		},

		// line matches leStartPattern and passes fromTime but fails untilTime
		{
			name:                       "test: leStartPattern matches, passes fromTime, fails untilTime",
			line:                       testLine,
			leStartPattern:             odlStartPattern,
			leTimeFormat:               odlTimeFormat,
			fromTime:                   parseTime(odlTimeFormat, `Jun 15, 2010 1:00:00 AM IST`),
			untilTime:                  parseTime(odlTimeFormat, `Jun 15, 2010 2:00:00 AM IST`),
			expectedStartOfLe:          true,
			expectedFromTimeSatisfied:  true,
			expectedUntilTimeSatisfied: false,
		},

		// line matches leStartPattern and passes untilTime but fails fromTime
		{
			name:                       "test: leStartPattern matches, fails fromTime, passes untilTime",
			line:                       testLine,
			leStartPattern:             odlStartPattern,
			leTimeFormat:               odlTimeFormat,
			fromTime:                   parseTime(odlTimeFormat, `Jun 15, 2010 3:00:00 AM IST`),
			untilTime:                  parseTime(odlTimeFormat, `Jun 15, 2010 4:00:00 AM IST`),
			expectedStartOfLe:          true,
			expectedFromTimeSatisfied:  false,
			expectedUntilTimeSatisfied: true,
		},

		// leStartPattern doesn't match
		{
			name:                       "leStartPattern doesn't match",
			line:                       testLineNoLeStart,
			leStartPattern:             odlStartPattern,
			expectedStartOfLe:          false,
			expectedFromTimeSatisfied:  false,
			expectedUntilTimeSatisfied: false,
		},

		// if leTimeFormat doesn't match when leStartPattern does, and time
		// constraints exist, an error should be thrown
		{
			name:                       "test: leTimeFormatMismatch",
			line:                       testLine,
			leStartPattern:             odlStartPattern,
			leTimeFormat:               odlTimeFormat + `unmatch`,
			fromTime:                   parseTime(odlTimeFormat, `Jun 15, 2010 2:00:00 AM IST`),
			expectedStartOfLe:          true,
			expectedFromTimeSatisfied:  false,
			expectedUntilTimeSatisfied: false,
			expectedErr:                LeTimeFormatMismatch,
		},

		// if user has specified no capturing group in leStartPattern,
		// an error should be thrown
		{
			name:                       "test: no capturing groups in leStartPattern",
			line:                       testLine,
			leStartPattern:             `^<\w{3} \d{2}, \d{4} \d{1,2}:\d{2}:\d{2} (?:AM|PM) (?:\S+)>`,
			leTimeFormat:               odlTimeFormat,
			fromTime:                   parseTime(odlTimeFormat, `Jun 15, 2010 2:00:00 AM IST`),
			expectedStartOfLe:          true,
			expectedFromTimeSatisfied:  false,
			expectedUntilTimeSatisfied: false,
			expectedErr:                LeStartPatternBadlyFormed,
		},

		// if the user has specified more than one capturing groups in leStartPattern,
		// an error should be thrown
		{
			name:                       "test: too many capturing groups in leStartPattern",
			line:                       testLine,
			leStartPattern:             `^<(\w{3} (\d{2}, \d{4}) \d{1,2}:\d{2}:\d{2} (?:AM|PM) (?:\S+))>`,
			leTimeFormat:               odlTimeFormat,
			fromTime:                   parseTime(odlTimeFormat, `Jun 15, 2010 2:00:00 AM IST`),
			expectedStartOfLe:          true,
			expectedFromTimeSatisfied:  false,
			expectedUntilTimeSatisfied: false,
			expectedErr:                LeStartPatternBadlyFormed,
		},

		// check empty line can be handled OK
		{
			name:                       "test: empty line buf",
			line:                       "",
			leStartPattern:             odlStartPattern,
			leTimeFormat:               odlTimeFormat,
			expectedStartOfLe:          false,
			expectedFromTimeSatisfied:  false,
			expectedUntilTimeSatisfied: false,
		},

		// check line with 1 char
		{
			name:                       "test: line with 1 char",
			line:                       "X",
			leStartPattern:             odlStartPattern,
			leTimeFormat:               odlTimeFormat,
			expectedStartOfLe:          false,
			expectedFromTimeSatisfied:  false,
			expectedUntilTimeSatisfied: false,
		},
	}

	// iterate through tests
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// invoke processLine with test parameters
			startOfLe, fromTimeSatisfied, untilTimeSatisfied, err := processLine(
				[]byte(test.line), compileRegexp(test.leStartPattern),
				test.leTimeFormat, test.fromTime, test.untilTime,
			)

			// compare err with expectedErr
			if err != nil && test.expectedErr == "" {
				t.Error(err)
				return
			}
			actualErr := ""
			if err != nil {
				actualErr = err.Error()
			}
			if !strings.Contains(actualErr, test.expectedErr) {
				t.Errorf("Got error: \"%s\", Want error: \"%s\"", actualErr,
					test.expectedErr)
			}

			// compare startOfLe with expectedStartOfLe (1st return value)
			if startOfLe != test.expectedStartOfLe {
				t.Errorf("startOfLe does not match expectedStartOfLe. Got %t, want %t",
					startOfLe, test.expectedStartOfLe)
			}

			// compare fromTimeSatisfied with expectedFromTimeSatisfied (2nd return value)
			if fromTimeSatisfied != test.expectedFromTimeSatisfied {
				t.Errorf("fromTimeSatisfied does not match expectedFromTimeSatisfied."+
					" Got %t, want %t",
					fromTimeSatisfied, test.expectedFromTimeSatisfied)
			}

			// compare untilTimeSatisfied with expectedUntilTimeSatisfied (3rd return value)
			if untilTimeSatisfied != test.expectedUntilTimeSatisfied {
				t.Errorf("untilTimeSatisfied does not match expectedUntilTimeSatisfied."+
					" Got %t, want %t",
					untilTimeSatisfied, test.expectedUntilTimeSatisfied)
			}
		})
	}
}

// test processLogEntry (greenpaths only as there are no custom defined red paths)
func TestProcessLogEntry(t *testing.T) {
	// define test log entry for use in all the tests
	testLogEntry := []byte("[23/Sep/2019:00:35:37 +0200] word1 word2 word3")

	// define tests
	var tests = []struct {
		name           string   // name/summary of test
		logEntry       []byte   // 1st parameter
		regexps        []string // 2nd parameter
		expectingMatch bool     // are we expecting a match and hence the outputHandler to be invoked
	}{
		// no regexps
		{
			name:           "test: no regexps",
			logEntry:       testLogEntry,
			expectingMatch: true,
		},

		// one regexp match
		{
			name:           "test: 1 regexp match, single line",
			logEntry:       testLogEntry,
			regexps:        []string{`word1`},
			expectingMatch: true,
		},

		// one unmatching regexp
		{
			name:           "test: 1 regexp unmatch, single line",
			logEntry:       testLogEntry,
			regexps:        []string{`word4`},
			expectingMatch: false,
		},

		// 3 matching regexps
		{
			name:     "test: 3 matching regexps, single line",
			logEntry: testLogEntry,
			regexps: []string{
				`word1`,
				`word2`,
				`word3`,
			},
			expectingMatch: true,
		},

		// 2 matching regexps and 1 unmatching regexp
		{
			name:     "test: 2 matching regexps, 1 unmatching regexp, single line",
			logEntry: testLogEntry,
			regexps: []string{
				`word1`,
				`word2`,
				`word4`,
			},
			expectingMatch: false,
		},
	}

	// when processLogEntry calls testOutputHandler (which will happen in the
	// event of a log entry match being found), it will change the matchFound
	// variable to true, and pass the reference to it's input param, "logEntry",
	// to logEntryOutputParam
	matchFound := false
	logEntryOutputParam := []byte{}
	testOutputHandler := func(logEntry []byte) {
		matchFound = true
		logEntryOutputParam = logEntry
	}

	// iterate over tests
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// reset matchFound
			matchFound = false

			// call processLogEntry
			processLogEntry(test.logEntry, compileRegexps(test.regexps), testOutputHandler)

			// compare matchFound with expectingMatch, and check the expected value
			// (logEntry) is being passed to outputHandler
			if matchFound != test.expectingMatch {
				t.Errorf("matchFound does not match expectingMatch. Got %t, Want %t",
					matchFound, test.expectingMatch)
			} else if matchFound && (string(logEntryOutputParam) != string(test.logEntry)) {
				t.Errorf("output handler's logEntry param does not match test.logEntry")
			}
		})
	}
}
