package cut

import "strings"

func CombineLesson(name, teacher string) string {
	return name + "_" + teacher
}

func CutLesson(key string) (string, string) {
	slice := strings.Split(key, "_")
	lesson := slice[0]
	teacher := slice[1]

	return lesson, teacher
}

func SplitInfo(info string) (string, string) {
	slice := strings.Split(info, "/")
	username := slice[0]
	teacher := slice[1]
	return username, teacher
}
