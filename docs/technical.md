# Technical Documentation

The information in this document is brief and intended to guide understanding of the code and development ideas. More detailed information about the code can be found in the code comments. It is also worth mentioning that code comments have been written with godoc in mind.

## Performance Analysis

The space complexity of the ReverseSearch function is O(m) where m is the number of bytes in the largest log entry that will get processed. The worst case runtime is O(n) where n is the number of bytes, but this can vary a lot depending on the size of the file, and the overall % of bytes that get processed before the search terminates upon finding a log entry that fails the FromTime constraint.

<TODO: Explain about the way findLogEntries currently operates, and how StartBufLen affects it; explain the issue of over reading of bytes vs too many file access times, and maybe how we could go about deriving the best default value for StartBufLen - reference the proposed changes to findLogEntries also>

<TODO: explain how performance is tied to regular expressions engine.>

<TODO: explain how validation of search params will be done once only, i.e. no point in going over the top, validating params in every single function as it would waste resources>

## Glossary

This section is intended to help understand abbreviations used in code comments and variable/function names.

TODO
