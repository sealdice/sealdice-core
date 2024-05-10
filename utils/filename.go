package utils

import "regexp"

// FilenameClean makes a name legal for file name by removing every
// '/', ':', '*', '?', '"', '<', '>', '|', '\' from the name.
func FilenameClean(name string) string {
	re := regexp.MustCompile(`[/:\*\?"<>\|\\]`)
	return re.ReplaceAllString(name, "")
}
