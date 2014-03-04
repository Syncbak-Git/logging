/*
Package logging provides basic logging with machine-readable output.

Log Format

The log format is designed to be easily machine readable. Log entries consist of a
timestamp field, severity field, a free-format message field and a JSON-encoded set of key-value pairs.
The fields are tab-separated, and '\t', '{' and '}' in the message field are replaced with ' ', '[' and ']' respectively.
The JSON-encoded key-value pairs include an optional set of user-supplied key-values, and the following values:
		timestamp RFC3339 Nano UTC timestamp
		severity  DEBUG, INFO, WARNING, ERROR, CRITICAL, FATAL
		pid       process ID
		pid       process ID
		app       application name
		host      host name or IP address
		line      line number where the Logger was called
		file      file name where the logger was called
		function  function name where the logger was called
*/
package logging
