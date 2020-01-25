# Technical Documentation

The purpose of this document is to contain information that I think is unnecessarily technical for the README.md file.

## Performance Analysis

### Memory Usage

The space complexity of the ReverseSearch function is O(m) where m is the size of the bytes buffer at the largest point. This will usually be equal to StartBufLen, but sometimes there may be a log entry that needs to be processed that is larger than this value (which would be rare if StartBufLen was sufficiently large, which the deffault value should be), in which case O(m) would be equal to anywhere between 1-2 times the size of the largest log entry to be processed (because the bytes buffer gets doubled in size every time it comes across a log entry that will not fully fit into it). The worst case is for O(m) to be equal to MaxBufLen, which can easily happen in cases of erroneous input, such as when the LeStartPattern doesn't match the beginning of any log entries in the log file, so it must be stressed to keep MaxBufLen to a value that the environment can withstand.

### Run Time

The worst case run time complexity is O(n) where n is the number of bytes in the log file, but the actual run time can vary a lot depending on the size of the file, and the overall % of bytes that get processed before the search terminates upon finding a log entry that fails the FromTime constraint. A related note on actual run time is how large the bytes buffer is and how many file reads are required, which for the most part depends on the value of StartBufLen.

The best value for StartBufLen depends on the overhead per file read operation (the assumption is that this is significantly larger than the next type of overhead), and the overhead per byte during read operations. To understand, consider this - when a line is found within the buffer that fails the FromTime constraint, no more lines need to be processed beyond that point in the buffer, so if the buffer is large enough, and the overhead per byte during read operations is great enough, it may turn out to be less efficient than having a smaller size for the buffer, even if that would mean more overall file reads. This also depends on a number of hard-to-predict factors, such as the number of total bytes that will be processed before a FromTime constraint fails, or IF a FromTime constraint will even fail at all. There is an item in the backlog to investigate what the best default value for StartBufLen should be.

Another important factor for overall run time of ReverseSearch is the performance of Go's "regexp" library. Some research clearly suggests that C's regexp library (PCRE) is a lot more performant than Go's (which makes sense as it has been streamlined by the community for decades). An option here is to translate the library to C or C++ and compare the performance to the Go version of it, then instead turn the Go version into bindings for the C/C++ version if the C/C++ version gave a significant performance boost.

I was particularly mindful around areas of code that could potentially be called millions, or even billions of times, such as not bothering to validate the parameters in any of the findLogEntries, processLine, processLogEntry functions, as the overhead of doing this through millions of iterations would start to mount up. Instead parameters are only validated in the ReverseSearch function. Also there is a point in the backlog about improving the findLogEntries function that would remove more of this type of overhead; in particular, getting rid of the nlPosStack and related code. Another consideration is how much overhead function calling gives, as processLine could potentially be called a large number of times, however, this will probably be very insignificant compared with the operations that actually take place within that function.

I have already mentioned that translating the code to C/C++ could improve the performance in regards to regular expressions, but it could also improve the file reads, and the areas of code that get iterated over a lot too, so there could be a lot to gain from re-writing this in C/C++.

## Glossary

This section is intended to help understand abbreviations used in code comments and variable/function names.

<b>le :</b> Log entry

<b>nl :</b> Newline character

<b>buf :</b> This refers to the bytes buffer that is used to read bytes from the log file

<b>re :</b> Regular expression

<b>pos :</b> Position; i.e. within an array/slice
