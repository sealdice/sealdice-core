package utils

import "os"

type ByModtime []os.FileInfo

func (s ByModtime) Len() int           { return len(s) }
func (s ByModtime) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByModtime) Less(i, j int) bool { return s[i].ModTime().Before(s[j].ModTime()) }
