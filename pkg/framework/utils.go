package framework

import "strings"

func TrimBlank(str string) string {
	const blankSet = " \t\n\r"
	return strings.Trim(str, blankSet)
}
