package agent

import "testing"

func TestConfirmsMallExchange(t *testing.T) {
	for _, message := range []string{"确认兑换", "我确认下单这个商品", "confirm purchase"} {
		if !confirmsMallExchange(message) {
			t.Fatalf("expected explicit confirmation for %q", message)
		}
	}
	for _, message := range []string{"推荐一个商品", "看看商城", "这个不错"} {
		if confirmsMallExchange(message) {
			t.Fatalf("must not approve exchange for %q", message)
		}
	}
}
