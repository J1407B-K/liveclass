package cut

import (
	"strings"
)

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
	return slice[0], slice[1]
}

func SplitToLessonID(info string) []string {
	slice := strings.Split(info, "$")
	return slice
}

func CombineAddr(rtmp, flv, hls, key string) map[string]string {
	addr := make(map[string]string)

	addr["rtmp"] = rtmp + key
	addr["flv"] = flv + key + ".flv"
	addr["hls"] = hls + key + ".m3u8"

	return addr
}

func ShowAddr(addr map[string]string) string {
	return "rtmp play:" + addr["rtmp"] + "$" + "flv play:" + addr["flv"] + "$" + "hls play:" + addr["hls"]
}

func ShowOptions(so []string) string {
	switch len(so) {
	case 2:
		return "A:" + so[0] + "|" + "B:" + so[1]
	case 3:
		return "A:" + so[0] + "|" + "B:" + so[1] + "|" + "C:" + so[2]
	case 4:
		return "A:" + so[0] + "|" + "B:" + so[1] + "|" + "C:" + so[2] + "|" + "D:" + so[3]
	}
	return ""
}

func SplitAnsResp(str string) (string, string) {
	slice := strings.Split(str, "$")
	return slice[0], slice[1]
}
