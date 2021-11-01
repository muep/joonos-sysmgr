package main

func sliceContains(ss []string, s string) bool {
	for _, t := range ss {
		if s == t {
			return true
		}
	}
	return false
}
