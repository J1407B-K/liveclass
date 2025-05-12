package cut

import "strings"

func CutLessons(l string) []string {
	return strings.Split(l, "$")
}

func CombineLessons(sl []string) string {
	return strings.Join(sl, "$")
}

func OutputLessons(l string) string {
	sl := CutLessons(l)
	return strings.Join(sl, "/")
}
