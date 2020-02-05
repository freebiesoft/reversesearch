# ReverseSearch

ReverseSearch is a library that allows callers to search log files in reverse. Time constraints may be specified such as "FromTime" and "UntilTime"; when a log entry is discovered with a timestamp that is before the FromTime constraint, the process terminates and no further log entries are searched within the log file, thus saving time & resources. The library works seamlessly for single and multi line log entries alike.

The idea behind this is that most of the time I have wanted to search through large log files (on the scale of gigabytes), I am usually only interested in log entries that were logged within a relatively short amount of time ago. Under such scenarios it would be much more efficient to search log files in reverse, and terminate the search upon finding a log entry that was logged before a specified time.

One of the main reasons I started this project was to learn TDD and increase my Go programming skills, however, if enough people show interest in this project, or if I found any use for a high performance log searching library in the future, I will continue work as per the vision, the backlog (see docs/backlog.md), and community suggestions (please feel free to suggest features).

### The Vision

The vision is to create a high performance general log file searching library that will offer the following functionalities:

- <b>ForwardSearch:</b> a forward searching function that would terminate searches after failing UntilTime conditions, and would be designed to be more performant when only UntilTime is specified, or if you know the information you're looking for will be closer to the beginning of the log file
- <b>ReverseSearch:</b> (already implemented of course!) a reverse searching function that would terminate searches after failing FromTime conditions, and would be designed to be more performant when only FromTime is specified, or if you know the information you're looking for will be closer to the end of the log file
- <b>BinarySearch:</b> a function designed to be more performant for time ranges (i.e. when FromTime and UntilTime are both specified) which would employ a binary search based algorithm to find the point at which the time range starts within the log file, then use forward search mechanics until the UntilTime constraint fails.

Possibly a wrapping function could be implemented too, which would guess which of the aforementioned functions would be the most performant based on specified information such as the search criteria and log file.

Although lots of focus has already gone into making the ReverseSearch function as optimised as possible, there're ideas to make it further optimised that're awaiting implementation (see docs/backlog.md for more info). Further on, it would be great to create a C translation of the library (along with provided bindings for Python and Go) as C's regexp library (PCRE) is a lot more performant than Go's, and this library's performance is highly dependent on the regular expression engine used.


## Getting Started

To get started, run through the prerequisites below and then check the examples/main.go for a quick walkthrough.

### Prerequisites

Before being able to run reversesearch there is a package dependency that you will need to download. Please open up a terminal and run the following command:

```
go get github.com/golang-collections/collections
```

After this you can download the reversesearch library with the following command:

```
go get github.com/freebiesoft/reversesearch
```

## Running the tests

As this project was developed using TDD and due to the nature of the code it is very bug prone, there are lots of tests and they are very thorough! To run them all, simply type the following command from the root project directory:

```
go test
```

This will run both unit tests and integration tests; i.e. the tests in reversesearch_unit_test.go and reversesearch_int_test.go. The unit tests focus on all the internal functions whilst the integration tests focus on the one exported function, ReverseSearch.

## Features

- Reversesearch is 100% thread-safe.
- Can specify own match mechanics via a custom output handler (see example 4 in examples/main.go).
- Works seamlessly with log files that use single or multi line log entries.
- Works seamlessly with log files that use either windows style newlines (CRLF) or Unix style newlines (LF).
- Code has been commented in line with GoDoc standards.

## Limitations and Assumptions

- This library has only been tested with UTF-8 encoded files (but ANSI encoded files should work fine too), moreover, UTF-8 (variable length encoding) & ANSI encodings were the only ones considered during development.
- This library has only been tested on Windows and Linux.
- Log files must be standardised and predictable in nature i.e.:
  - must be able to define how a log entry "starts" via regular expressions and be able to capture their time stamps within (a regex construct known as) a capturing group.
  - the format of the timestamp must remain the same throughout log entries.
- The size of any given log entry within a log file can be no greater than MaxBufLen.
- Log entries within log files are strictly in chronological order. If you work with log files where this isn't the case (as can be the case where log entries' time stamps reflect something other than time of logging such as time of request), then you could add some sort of "tolerance time" to the time constraints e.g. add or subtract a certain amount of time from your time constraints depending on your circumstances.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details
