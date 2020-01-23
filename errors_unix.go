// +build unix

package reversesearch

/* this file is created for convenience reasons i.e.:
- to make it clear which custom errors exist in this library
- make it easier to reference these custom errors either internally or outside
  of the library code; especially if amendments are made to these strings
*/

// NoLogEntriesInFile is returned when no log entries are detected in the log file
const NoLogEntriesInFile = "No log entries found in file"

// NoMoreLogEntries is returned when LeStartPattern stops matching beyond a certain
// point in the file; it is assumed that LeStartPattern will match at the beginning
// of the file
const NoMoreLogEntries = "No more entries found"

// FileIsEmpty is returned when there're no characters detected in the file
const FileIsEmpty = "File is empty"

// NoLeTimeFormat is returned when the user has not specified LeTimeFormat in their
// search criteria parameter to ReverseSearch at the same time as specifying
// either FromTime or UntilTime. If neither FromTime or UntilTime are specified,
// LeTimeFormat does not have to be specified in the search criteria.
const NoLeTimeFormat = "leTimeFormat must be set if there are time constraints"

// NoLeStartPattern is returned when the user has not specified LeStartPattern in
// their search criteria parameter to ReverseSearch
const NoLeStartPattern = "leStartPattern must be set in search criteria"

// FromTimeAfterUntilTime is returned when the user has specified both FromTime
// and UntilTime, and FromTime > UntilTime
const FromTimeAfterUntilTime = "fromTime needs to be before untilTime"

// MaxBufLenReached is returned when the byte buffer's size has exceded MaxBufLen.
// This most commonly happens when the user has specified an LeStartPattern that
// doesn't match any log entries, but can also happen when MaxBufLen is too small,
// or there is a very large log entry in the log file.
const MaxBufLenReached = "The maximum buffer length has been reached"

// LeTimeFormatMismatch is returned when match group one of LeStartPattern for
// a particular log entry does not match LeTimeFormat
const LeTimeFormatMismatch = "leTimeFormat doesn't match"

// BufOffsetLessThanZero is returned when the program attempts to read bytes before
// the beginning of the log file. If this happens please report it to
// https://github.com/freebiesoft/reversesearch/issues
const BufOffsetLessThanZero = "Buffer offset is less than zero"

// LeStartPatternBadlyFormed is returned when LeStartPattern matches the beginning
// of a log entry, but either none or more than 1 capturing groups where found.
// Only one capturing group should be defined within LeStartPattern which should
// capture the timestamp of the log entry
const LeStartPatternBadlyFormed = "leStartPattern should have 1 capturing group"

// BadLeStartPattern is returned when the search criteria's LeStartPattern field
// won't compile (i.e. regexp.Compile returns an error)
const BadLeStartPattern = "search criteria's LeStartPattern field won't compile"

// BadRegexps is returned when one of the regular expressions in search criteria's
// Regexps field won't compile (i.e. regexp.Compile returns an error)
const BadRegexps = "one of the regex strings in search criteria's Regexps field won't compile"

// BadFilePath is returned when user specifies a non existent filePath parameter
// to ReverseSearch. The value on this is OS dependant and hence depends on which
// build tag was used with go build, i.e. "windows" or "unix"
const BadFilePath = "No such file or directory"
