package framework

import "strings"

func TrimBlank(str string) string {
	const blankSet = " \t\n\r"
	return strings.Trim(str, blankSet)
}

func Indent(str string, indent int) string {
	lines := strings.Split(str, "\n")
	for i, line := range lines {
		if i == 0 {
			// Skip the first line
			continue
		}
		// Trim leading spaces
		lines[i] = strings.TrimLeft(line, " \t")
		// Add indent
		lines[i] = strings.Repeat(" ", indent) + lines[i]
	}
	return strings.Join(lines, "\n")
}
