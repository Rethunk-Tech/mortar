package fsx

import "strings"

// FoldCase maps path to the form two spellings of one location share: Windows paths ignore case, so D:\SteamLibrary
// and d:\steamlibrary are one folder.
func FoldCase(path string) string { return strings.ToLower(path) }
