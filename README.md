# Reversesearch

Reversesearch is a library that allows callers to search log files in reverse. Time constraints may be specified such as "FromTime" and "UntilTime"; when a log entry is discovered with a timestamp that is before the FromTime constraint, the process terminates and no further log entries are searched within the log file, thus saving time & resources. The library works seamlessly for single and multi line log entries alike.

The idea behind this is that most of the time I have wanted to search through large log files (on the scale of gigabytes), I am usually only interested in log entries that were logged within a relatively short amount of time ago from now. Under such scenarios it would be much more efficient to search log files in reverse, and terminate the search upon finding a log entry that was logged before a specified time.

One of the main reasons I started this project was to learn TDD and increase my Go programming skills, however, if enough people show interest in this project, or if I found any use for a high performance log searching library in the future. I will continue work as per the vision, the backlog, and any other community suggestions.

#### The Vision

The vision is to create a more general log file searching library that will offer a forward searching function (that would terminate searches after failing UntilTime conditions), a reverse searching function (already implemented of course!), and a function designed for time ranges (i.e. when FromTime and UntilTime are both specified) which would employ a binary search based algorithm that would find the point at which the time range starts within the log file, then use forward search mechanics until the UntilTime constraint fails.

Although lots of focus has already gone into making the ReverseSearch function as optimised as possible, there're ideas to make it further optimised that're awaiting implementation (see docs/ for more info on the backlog). Further on, it would be great to create a C translation of the library (along with provided bindings for Python and Go) as C's regexp library (PCRE) is a lot more performant than Go's, and this library's performance is highly dependent on the regular expression engine used.


## Getting Started

To get started, quickly run through the prerequisites below and then check the examples/main.go for a quick walkthrough.

#### Prerequisites

Before being able to run reversesearch there is a package dependency that you will need to download. Please open up a terminal and run the following command:

```
go get github.com/golang-collections/collections
```

After this you can download the reversesearch library with the following command:

```
go get github.com/freebiesoft/reversesearch
```

## Running the tests

As this project was developed using TDD and the nature of the nature of the code is very bug prone, there are lots of tests! To run them all, simply type the following command from the root project directory:

```
go test
```

This will run both unit tests and integration tests; i.e. the tests in reversesearch_unit_test.go and reversesearch_int_test.go. The unit tests focus on all the internal functions whilst the integration tests focus on the one exported function, ReverseSearch.

## Features

- reversesearch is 100% thread-safe.
- Can specify own match mechanics via a custom output handler (see example 4 in examples/main.go).
- Works seamlessly with log files that use single or multi line log entries.
- Works seamlessly with log files that use either windows style newlines (CRLF) or Unix style newlines (LF).
- Code has been commented with GoDoc in mind.

## Limitations

- Log files must be predictable in nature i.e. you must be able to define how a log entry "starts" via regular expressions, and must be able to capture their time stamps within (a regex construct known as) a capturing group.
- Log files must be standardised and predictable in nature i.e.:
  - must be able to define how a log entry "starts" via regular expressions and be able to capture their time stamps within (a regex construct known as) a capturing group.
  - time format in the timestamp must remain the same throughout log entries.
- The size of any given log entry within a log file can be no greater than MaxBufLen.
- Log entries within log files are strictly in chronological order. If you work with log files where this isn't the case (as can be the case with log files where log entries' time stamps reflect something other than time of logging such as time of request), then you could add some sort of "tolerance time" to the time constraints e.g. add or subtract a certain amount of time from your time constraints depending on your circumstances.
- This library has only been tested on Windows and Linux.
- This library has only been tested with UTF-8 encoded files (but there should certainly be problem with using ASCII encoded files too).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details
