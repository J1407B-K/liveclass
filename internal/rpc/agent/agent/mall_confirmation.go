package agent

import "strings"

func confirmsMallExchange(message string) bool {
	message = strings.ToLower(strings.TrimSpace(message))
	phrases := []string{"确认兑换", "确认下单", "确定兑换", "confirm exchange", "confirm purchase"}
	for _, phrase := range phrases {
		if strings.Contains(message, phrase) {
			return true
		}
	}
	return false
}
