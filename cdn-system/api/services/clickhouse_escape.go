package services

import "strings"

var clickhouseStringReplacer = strings.NewReplacer(
	"\\", "\\\\",
	"'", "\\'",
	"\n", "\\n",
	"\r", "\\r",
	"\t", "\\t",
	"\x00", "\\0",
)

func escapeClickHouseString(value string) string {
	return clickhouseStringReplacer.Replace(value)
}

func quoteClickHouseString(value string) string {
	return "'" + escapeClickHouseString(value) + "'"
}
