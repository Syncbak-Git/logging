/*
Package logging provides basic logging with machine-readable output.

Log Format

The log format is designed to be easily machine and human readable.
There are normally two log files, one for text and one for json, but
both text and json entries can be written to the same file on separate
lines (see SetLogFile discussion).
Text log entries consist of tab separated timestamp, severity and message fields. JSON log
entries are JSON-encoded key-value pairs. In both text and json entries, '\t', '{' and '}' in
the message field are replaced with ' ', '[' and ']' respectively.
The JSON-encoded key-value pairs include an optional set of user-supplied key-values, and the following values:
		timestamp RFC3339 Nano UTC timestamp
		severity  DEBUG, INFO, WARNING, ERROR, CRITICAL, FATAL
        message   free form text field
		pid       process ID
		app       application name
		host      host name or IP address
		line      line number where the Logger was called
		file      file name where the logger was called
		function  function name where the logger was called
*/
package logging
