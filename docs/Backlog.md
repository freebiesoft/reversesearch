# Backlog

### Improve findLogEntries

Currently, findLogEntries scans the bytes buffer forwards for new line characters, placing each position within the buf that they're found at onto a stack along with their size (i.e. a '\r\n' character would be 2 bytes, '\n' would be 1 byte) as they're discovered. The lines are then processed backwards, inferring where lines begin and end via the newline positions stack. This means all bytes in the buffer will be scanned for new line characters even if there's a line in the buffer that matches leStartPattern and fails FromTime constraint. It would be more efficient to analyse each byte in the buffer backwards and process each line as they're discovered instead; this would be more efficient under such circumstances because it means no more bytes need to be scanned other than what is necessary since upon finding a line that fails FromTime, the search would be terminated immediately.

Although such a change may only introduce a relatively minor performance increase (i.e. compared with reading the bytes from disk in the first place), it would also simplify the logic of findLogEntries, as there'd be no need for the nlPosStack variable, and the edge case (i.e. in the event that bufOffset == 0) would be simpler to deal with.

### Determine the Best Default Value for StartBufLen

Based on the paragraph about StartBufLen in the performance analysis section of the technical documentation, investigate what the best default value for StartBufLen should be.

### Configuration Settings

Implement configuration settings that would give programmers more flexibility over how strictly log files must follow rules. Current ideas are:
- configuration setting that determines if "NoMoreLogEntries" error should be thrown when the first line in the file does not match LeStartPattern
- configuration setting that determines if "FileIsEmpty" error should be thrown

### Translate to C or C++

Further on, it would be great to create a C/C++ translation of the library as C's regexp library (PCRE) is a lot more performant than Go's, and this library's performance is highly dependent on the regular expression engine used. There may also be performance improvements in other areas such as file reading when written in C or C++.

If the performance improvement is good enough, then this library could instead effectively be transformed to bindings for the C/C++ library which would hold the main functionality.
