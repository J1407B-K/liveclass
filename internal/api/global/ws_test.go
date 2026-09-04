package global

import (
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
)

func originContext(origin, host string) *app.RequestContext {
	ctx := app.NewContext(0)
	ctx.Request.SetHost(host)
	if origin != "" {
		ctx.Request.Header.Set("Origin", origin)
	}
	return ctx
}

func TestOriginChecker(t *testing.T) {
	check := originChecker([]string{"https://class.example.com"})
	for _, tc := range []struct {
		origin, host string
		want         bool
	}{
		{origin: "", host: "api.example.com", want: true},
		{origin: "https://api.example.com", host: "api.example.com", want: true},
		{origin: "https://class.example.com", host: "api.example.com", want: true},
		{origin: "https://evil.example.com", host: "api.example.com", want: false},
		{origin: "ftp://api.example.com", host: "api.example.com", want: false},
		{origin: "https://api.example.com/path", host: "api.example.com", want: false},
		{origin: "://bad", host: "api.example.com", want: false},
	} {
		if got := check(originContext(tc.origin, tc.host)); got != tc.want {
			t.Errorf("origin=%q host=%q got %v want %v", tc.origin, tc.host, got, tc.want)
		}
	}
}
