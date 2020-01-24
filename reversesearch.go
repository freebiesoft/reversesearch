/*Package reversesearch manages the reverse searching of log files. The idea
behind this is that a lot of the time large log files want to be searched,
technicians are only interested in searching within the recent past. Under such
scenarios it would be much more effiicient to search log files in reverse, and
then terminate the search upon finding a log entry that was logged before a
specified time. */
package reversesearch

/* All the main functions are contained in this file:
- increaseBufLen
- processLogEntry
- processLine
- findLogEntries
- ReverseSearch (exported)

There are also 2 exported variables in this file:
- MaxBufLen
- StartBufLen
*/

import (
	"errors"
	"fmt"
	"github.com/golang-collections/collections/stack"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MaxBufLen defines the maximum size of the bytes buffer that is used to read
// log files with. This number should be large enough to fit the largest log
// entry that you might work with but not too large so that there could be a
// chance of using too much memory.
var MaxBufLen = 2000000 // 2MB

// StartBufLen defines the starting length of the bytes buffer which is used to
// read bytes from log files. It is advised that this value is sufficietly large
// so as to reduce the number of file accesses and hence boost performance.
var StartBufLen = 25000 // default is 25KB

// OutputHandler is an interface for functions that can optionally be provided
// provided to ReverseSearch as a parameter. When no output handler is passed to
// ReverseSearch (i.e. outputHandler parameter is set as nil), matching log entries
// will just be printed to STDOUT, however, when a function that implements this
// interface is passed as the outputHandler parameter, matching log entries will
// be passed to that function as they're discovered.
type OutputHandler func(logEntry []byte)

// SearchCriteria is a struct that defines the search criteria that is passed
// to ReverseSearch. ReverseSearch then uses this search criteria to search the
// log file passed to it for matching log entries. Please see examples/main.go
// for instantiation examples of this struct.
type SearchCriteria struct {
	// Regexps is a slice of strings. Each string needs to represent a valid golang
	// regular expression. Matching log entries must match each regular expression
	// defined in this slice of strings. This field is optional and when ommitted,
	// all log entries (that pass the time constraints) will match.
	Regexps []string

	// FromTime is an optional field struct that can be set by the user. When set,
	// all matching log entries' time stamps must be more than or equal to this time.
	// A performant side effect of this field being set is that the moment the code
	// detects that a log entry was logged before FromTime, it will abort any further
	// processing of the log file, since the assumption is that log entries are
	// logged in chronological order
	FromTime time.Time

	// UntilTime is an optional field that can be set by the user. When set, all
	// matching log entries' time stamps must be less than this time.
	UntilTime time.Time

	// LeStartPattern is the only mandatory field of this struct for ReverseSearch
	// to operate with minimal search criteria. This string must represent a valid
	// golang regular expression which (generally) matches the beginning of all log
	// entries in the log file being searched. This is required so that the code can
	// distinguish where each log entry begins and ends. If time constraints are
	// included within the search criteria, then the timestamp of the log entry must
	// be captured within the first capturing group of the regexp.
	LeStartPattern string

	// LeTimeFormat represents the golang time format of the log file's log entries'
	// timestamps. This field is required only when at least one of FromTime or
	// UntilTime are set. This will be used to parse the string in the first
	// capturing group of LeStartPattern's match to a time.Time struct. More information
	// can be found about time formats here https://golang.org/pkg/time/#pkg-constants.
	LeTimeFormat string
}

// increaseBufLen increases the length of the bytes buffer and returns the number
// of elements added, and an error if one is encountered. After the increase,
// the existing elements in buf will be shifted rightwards as much as possible
// in the same order (so that the next elements from the file can be read into the
// buffer in relative order to the shifted elements.)
func increaseBufLen(buf *[]byte) (int, error) {
	// throw an error if maximum buffer length has already been reached
	if len(*buf) >= MaxBufLen {
		return 0, errors.New(MaxBufLenReached)
	}

	// determine new buf length
	var newBufLen int
	if len(*buf) == 0 { // sanity check
		newBufLen = 1
	} else {
		newBufLen = len(*buf) * 2
		if newBufLen > MaxBufLen {
			newBufLen = MaxBufLen
		}
	}

	// work out the number of elements to add to current buf
	nAdded := newBufLen - len(*buf)

	// create new buf using newBufLen
	newBuf := make([]byte, newBufLen)

	// copy old buf's elements to newBuf
	for i, e := range *buf {
		newBuf[nAdded+i] = e
	}

	*buf = newBuf
	return nAdded, nil
}

// processLogEntry takes a byte slice representing a log entry, and if all the
// regexps in the "regexps" param match the logEntry, then the logEntry is considered
// a match and passed to outputHandler
func processLogEntry(logEntry []byte, regexps []*regexp.Regexp, outputHandler OutputHandler) {
	if regexps != nil {
		for _, re := range regexps {
			if !re.Match(logEntry) {
				return
			}
		}
	}
	outputHandler(logEntry)
}

// processLine checks to see if "line" param matches leStartRegexp. If it does,
// and at least one of fromTime or untilTime are not nil, it will infer the
// time of logging from the leStartRegexp's match's first capturing group,
// and then compare this time with fromTime and untilTime. The return values are:
// 1) startOfLe (bool): indicates if the line matches leStartRegexp
// 2) fromTimeSatisfied (bool): indicates if fromTime is satisfied
// 3) untilTimeSatisfied (bool): indicates if untilTime is satisfied
// 4) err (error): indicates if an error was encountered during execution
func processLine(line []byte, leStartRegexp *regexp.Regexp, leTimeFormat string,
	fromTime time.Time, untilTime time.Time) (bool, bool, bool, error) {
	// find matches in "line" with leStartRegexp
	matches := leStartRegexp.FindSubmatch(line)

	if matches == nil {
		// line does not resemble the first line of a log entry, so return
		return false, false, false, nil
	} // beyond this if statement, it is assumed that the line is the first line of
	// a log entry because leStartRegexp has matched

	// if there're no user-specified time constraints, return (indicating all time
	// constraints are satisfied)
	if fromTime.IsZero() && untilTime.IsZero() {
		return true, true, true, nil
	}

	// check that there was one (and only one) capturing group defined in leStartRegexp
	if len(matches) == 0 { // sanity check
		return true, false, false, errors.New(`matches is empty`)
	} else if len(matches) < 2 {
		return true, false, false, errors.New(LeStartPatternBadlyFormed +
			", a capturing group is needed to identify log time")
	} else if len(matches) > 2 {
		return true, false, false, errors.New(LeStartPatternBadlyFormed +
			", there should only be one capturing group to identify log time")
	}

	// retrieve capturing group 1 (i.e. the log entry's time stamp)
	leTimeB := matches[1]

	// create Time struct that represents log entry's time of logging
	leTime, err := time.Parse(leTimeFormat, string(leTimeB))
	if leTime.IsZero() { // leTimeFormat doesn't match
		// if time constraints exist, it must be possible to infer the log entry's
		// time of logging, so an error must be returned
		return true, false, false, errors.New(LeTimeFormatMismatch)
	}
	if err != nil { // sanity check
		return true, false, false, err
	}

	// check leTime against time constraints
	fromTimeSatisfied, untilTimeSatisfied := true, true
	if !fromTime.IsZero() {
		fromTimeSatisfied = fromTime.Before(leTime) || fromTime.Equal(leTime)
	}
	if !untilTime.IsZero() {
		untilTimeSatisfied = untilTime.After(leTime)
	}

	return true, fromTimeSatisfied, untilTimeSatisfied, nil
}

// findLogEntries starts by analysing buf for newline characters. After finding
// the newline characters and their positions, it has enough information to infer
// where lines begin and end. findLogEntries will then traverse these lines in
// reverse; when it finds a line that matches leStartRegexp while satisfying both
// fromTime and untilTime, it will pass this line's bytes, along with all bytes
// up until the last position at which leStartRegexp matched, to processLogEntry.
// If a line matches leStartRegexp but fails to satisfy untilTime, it'll continue
// to traverse, but when a line matches leStartRegexp and fails to satisfy fromTime,
// findLogEntries will stop traversal and return abort status indicator as true.
// Upon calling findLogEntries, it is assumed that the last position at which
// leStartRegexp matched is len(buf). scanToPos and lastNlPos parameters exist
// as a means for code that calls findLogEntries iteratively to tell findLogEntries
// where it last "finished off"; scanToPos indicates the position in buf from which
// findLogEntries has already analysed the bytes in a previous call. lastNlPos
// was the position past this point in which the last newline was found and hence
// from where line traversal can continue. The following values are returned:
// 1) lastLePos (int): indicates the first position in the buf at which the last
//    log entry was discovered
// 2) lastNlPos (int): indicates the first position in the buf at which the last
//		newline was found - this helps to save re-analysing bytes which currently
//		exist between buf[0:lastLePos]
// 3) abort (bool): indicates if fromTime is no longer satisfied
// 4) err (error)
func findLogEntries(buf []byte, bOffset int64, scanToPos int, lastNlPos int,
	leStartRegexp *regexp.Regexp, leTimeFormat string, fromTime time.Time, untilTime time.Time,
	regexps []*regexp.Regexp, outputHandler OutputHandler) (int, int, bool, error) {

	/* --- initialise variable for tracking analysis of buf --- */
	// nlPosStack stacks variables of the form [2]int where [0] denotes the position
	// a newline was found at within buf, and [1] is the size in bytes of that newline
	// i.e. \r\n found at buf[12] would be recorded as [2]int{12, 2}
	nlPosStack := stack.New()

	bufLen := len(buf)

	// it is assumed last log entry was found after the contents of this buffer
	// (relative to buf's offset in the log file)
	lastLePos := bufLen
	bufIndex := 1

	// if bOffset == 0, it means this is the last buf load of bytes in the file,
	// which means it's necessary for this corner case logic to kick in
	if bOffset <= 0 {
		if bOffset == 0 {
			// it is worth noting here that we allow log files to be prefixed with a
			// newline character, but no other characters, before the first log entry
			if buf[0] == '\n' { // first character in file is \n
				nlPosStack.Push([2]int{0, 1})
			} else if bufLen >= 2 && buf[0] == '\r' && buf[1] == '\n' {
				// first character in file is \r\n
				nlPosStack.Push([2]int{0, 2})
				// required so we don't stack this newline char again in the following loop
				bufIndex = 2
			} else { // no newline (most common case) found at beginning of file
				nlPosStack.Push([2]int{0, 0})
			}
		} else { // bOffset < 0 ; return error
			return lastLePos, lastNlPos, false, errors.New(BufOffsetLessThanZero)
		}
	}

	// this is necessary in case buf[scanToPos+1] is a \n character; as such character
	// would not have been acknowledged as a newline in the previous call to findLogEntries
	// because findLogEntries cannot determine if \n is part of a \r\n when it is
	// found at buf[0] and there are more bytes to be read from the file.
	if scanToPos < bufLen-1 {
		scanToPos++
	}

	// find index positions of newlines in buf (and their corresponding byte sizes)
	for bufIndex <= scanToPos {
		if buf[bufIndex-1] == '\r' && buf[bufIndex] == '\n' {
			nlPosStack.Push([2]int{bufIndex - 1, 2})
		} else if buf[bufIndex] == '\n' {
			nlPosStack.Push([2]int{bufIndex, 1})
		}

		bufIndex++
	}

	// iterative through all newlines in reverse
	nlData := nlPosStack.Pop()
	for nlData != nil {
		// retrieve newline info from nlData
		nlInfo := nlData.([2]int)
		nlPos := nlInfo[0]
		nlSize := nlInfo[1]

		// determine if the bytes between nlPos and lastNlPos is the first line of a
		// log entry and if so, if it satisfies time constraints
		startOfLe, fromTimeSatisfied, untilTimeSatisfied, err := processLine(
			buf[nlPos+nlSize:lastNlPos], leStartRegexp, leTimeFormat, fromTime, untilTime,
		)
		if err != nil {
			if startOfLe {
				lastLePos = nlPos
			}
			return lastLePos, nlPos, false, err
		}

		if startOfLe { // leStartRegexp matched bytes between nlPos and lastNlPos
			if !fromTimeSatisfied {
				// if fromTime failed, no further log entries in the log file can match,
				// so return abort status as true
				return nlPos, nlPos, true, nil
			}
			if untilTimeSatisfied {
				processLogEntry(buf[nlPos+nlSize:lastLePos], regexps, outputHandler)
			}
			// update position at which last log entry has been found
			lastLePos = nlPos
		}

		lastNlPos = nlPos
		nlData = nlPosStack.Pop()
	}

	return lastLePos, lastNlPos, false, nil
}

// ReverseSearch searches the log file specified by filePath for matching
// log entries. "Matching log entries" are those that match all regular expressions
// contained in searchCriteria.Regexps, while satisfying any specified time constraints.
// In addition, the first log entry in the reverse traversal of the log file that fails
// the searchCriteria.FromTime constraint will trigger the abort mechanism, which will
// end the search process. Matching log entries are passed to outputHandler as
// they're found. There are two return variables:
//
// 1) exitStatus (int): -1 indicates an error was found, 0 indicates normal
// execution without issues, 1 indicates file is empty (not considered an error)
//
// 2) err (error)
//
// Please refer to examples/main.go for examples of function usage.
func ReverseSearch(filePath string, searchCriteria *SearchCriteria,
	outputHandler OutputHandler) (int, error) {

	// validate parameters
	if (!searchCriteria.FromTime.IsZero() || !searchCriteria.UntilTime.IsZero()) &&
		searchCriteria.LeTimeFormat == "" {
		return -1, errors.New(NoLeTimeFormat)
	}
	if searchCriteria.LeStartPattern == "" {
		return -1, errors.New(NoLeStartPattern)
	}
	if (!searchCriteria.FromTime.IsZero() && !searchCriteria.UntilTime.IsZero()) &&
		(searchCriteria.FromTime.After(searchCriteria.UntilTime) ||
			searchCriteria.UntilTime.Equal(searchCriteria.FromTime)) {
		return -1, errors.New(FromTimeAfterUntilTime)
	}

	// open file
	file, err := os.Open(filePath)
	if err != nil {
		return -1, err
	}
	defer file.Close()

	// get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return -1, err
	}
	fileSize := fileInfo.Size()

	// declare and initialise slice of compiled regexps
	var regexps []*regexp.Regexp
	if searchCriteria.Regexps != nil {
		regexps = make([]*regexp.Regexp, len(searchCriteria.Regexps),
			len(searchCriteria.Regexps))
		// compile searchCriteria.regExps and store them in regexps
		for i, regStr := range searchCriteria.Regexps {
			var err error
			regexps[i], err = regexp.Compile(regStr)
			if err != nil {
				if strings.Contains(err.Error(), `error parsing regexp`) {
					return -1, errors.New(BadRegexps)
				}
				return -1, err
			}
		}
	}

	// compile searchCriteria.LeStartPattern
	leStartRegexp, err := regexp.Compile(searchCriteria.LeStartPattern)
	if err != nil {
		if strings.Contains(err.Error(), `error parsing regexp`) {
			return -1, errors.New(BadLeStartPattern)
		}
		return -1, err
	}

	// if user did not specify an output handler, set it to fmt.Println
	var outHandler func([]byte)
	if outputHandler == nil {
		outHandler = func(logEntry []byte) { fmt.Println(string(logEntry)) }
	} else {
		outHandler = outputHandler
	}

	// required because the last char in a log file is usually a newline - we remove
	// it because otherwise it would be considered as part of the last log entry
	// in the file which would be inconsistent & incorrect
	if fileSize >= 2 {
		b := make([]byte, 2)
		_, err = file.ReadAt(b, fileSize-2)
		if err != nil {
			return -1, err
		}
		if b[0] == '\r' && b[1] == '\n' {
			fileSize = fileSize - 2
		} else if b[1] == '\n' {
			fileSize = fileSize - 1
		}
	} else if fileSize == 1 {
		b := make([]byte, 1)
		_, err = file.ReadAt(b, 0)
		if err != nil {
			return -1, err
		}
		if b[0] == '\n' {
			fileSize = 0
		}
	}

	// check file is not empty
	if fileSize < 1 {
		if fileSize < 0 { // sanity check
			return -1, errors.New("file size is less than 0")
		}
		return 1, nil
	}

	// initialise buf related variables
	bufOffset := fileSize
	var bufLen int
	if int64(StartBufLen) > fileSize {
		bufLen = int(fileSize)
	} else {
		bufLen = StartBufLen
	}
	buf := make([]byte, bufLen)

	// denotes buf position of the start of the last log entry found in buf
	var lastLePos int

	// on the next iteration after a call to findLogEntries in which at least one log entry was found,
	// buf[0:lastLePos] will be shifted rightwards so that those bytes can be
	// used again in the next call to findLogEntries without having to read them
	// from the file again. These bytes were already analysed in the previous call
	// to findLogEntries so it would seem wasteful to have to anaylse them again.
	// scanToPos tells the subsequent call to findLogEntries where to analyse bytes
	// up to so we don't repeat the work with those bytes again, and lastNlPos tells
	// the subsequent call the position of first newline in those bytes (which is
	// all thats needed since we know there is no pssibility of a leStartPattern match
	// in any of the later lines in those bytes if there are more than one)
	var scanToPos int
	var lastNlPos int

	// signal for when a found log entry fails searchCriteria.fromTime constraint
	abort := false

	// traverse file backwards, taking a buf's load of bytes at a time from bufOffset,
	// stopping when bufOFfset > 0 or when fromTime can no longer be satisfied
	for bufOffset > 0 && !abort {
		if lastLePos < bufLen {
			// at least 1 log entry was detected in buf during the call to findLogEntries
			// (or it's the first iteration)

			// shift previously read bytes before lastLePos as far right as they can go
			// because these bytes are part of the next log entry in the file that is
			// yet to be fully read into buf
			for i := len(buf[:lastLePos]) - 1; i >= 0; i-- {
				buf[i+bufLen-lastLePos] = buf[i]
			}

			// findLogEntries only needs to analyse the new bytes
			scanToPos = bufLen - lastLePos - 1
			lastNlPos = lastNlPos + bufLen - lastLePos

			// determine where bytes should be read from in the next read operation
			bufOffset = bufOffset - int64(bufLen-lastLePos)

			// if bufOffset < 0, truncate buf (at the left) and update related
			// variables as necessary so that we don't attempt to read before the
			// beginning of the file
			if bufOffset < 0 {
				buf = buf[-bufOffset:]
				bufLen = len(buf)
				scanToPos += int(bufOffset)
				lastNlPos += int(bufOffset)
				bufOffset = 0
			}

			// reads bytes from bufOffset up to just before the first position of
			// the bytes we should shifted
			file.ReadAt(buf[:bufLen-lastLePos], bufOffset)
		} else if lastLePos == bufLen {
			// no log entries were detected in buf which suggests buf length may be too
			// small

			// increase length of buffer & update related variables
			nAdded, err := increaseBufLen(&buf)
			if err != nil {
				return -1, err
			}
			bufLen += nAdded
			bufOffset -= int64(nAdded)

			// if bufOffset < 0, truncate buf (at the left) and update related
			// variables as necessary so that we don't attempt to read before the
			// beginning of the file
			if bufOffset < 0 {
				buf = buf[-bufOffset:]
				nAdded += int(bufOffset)
				bufLen += int(bufOffset)
				bufOffset = 0
			}

			// findLogEntries only needs to analyse the new bytes
			scanToPos = nAdded - 1
			lastNlPos += nAdded

			// reads bytes from bufOffset up to just before the first position of
			// the bytes that were shifted during the increaseBufLen function call
			file.ReadAt(buf[:nAdded], bufOffset)
		} else { // sanity check
			return -1, errors.New("lastLePos is more than bufLen")
		}

		// find log entries in buf, and pass the ones that match the specified regexps
		// while satisfying the time constraints to the outputHandler. abort will be
		// returned as true if any found log entries fail searchCriteria.FromTime
		lastLePos, lastNlPos, abort, err = findLogEntries(buf, bufOffset, scanToPos,
			lastNlPos, leStartRegexp, searchCriteria.LeTimeFormat, searchCriteria.FromTime,
			searchCriteria.UntilTime, regexps, outHandler)
		if err != nil {
			return -1, err
		}
	}

	// check to see if we found no log entries, or if log entries appeared to have
	// stopped beyond a certain point
	if bufOffset == 0 && lastLePos != 0 && !abort {
		if int64(lastLePos) == fileSize {
			return -1, errors.New(NoLogEntriesInFile)
		}
		return -1, errors.New(NoMoreLogEntries + `, last log entry found at ` +
			strconv.FormatInt((bufOffset+int64(lastLePos)), 10))
	}

	return 0, nil
}
