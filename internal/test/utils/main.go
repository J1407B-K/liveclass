package main

import (
	"fmt"
	"liveclass/internal/utils/cut"
)

func main() {
	s := "ls_kq"

	lesson, teacher := cut.CutLesson(s)
	fmt.Println(lesson)
	fmt.Println(teacher)
}
