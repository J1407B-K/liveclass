package wsauth

import (
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
)

func TestTokenTransportPriority(t *testing.T) {
	ctx := app.NewContext(0)
	ctx.Request.Header.Set("Authorization", "Bearer header-token")
	ctx.Request.Header.SetCookie("access_token", "cookie-token")
	ctx.Request.SetRequestURI("/ws?token=query-token")
	if got := Token(ctx, true); got != "header-token" {
		t.Fatalf("Token() = %q", got)
	}

	ctx.Request.Header.Del("Authorization")
	if got := Token(ctx, true); got != "cookie-token" {
		t.Fatalf("cookie Token() = %q", got)
	}
}

func TestTokenQueryRequiresExplicitOptIn(t *testing.T) {
	ctx := app.NewContext(0)
	ctx.Request.SetRequestURI("/ws?token=query-token")
	if got := Token(ctx, false); got != "" {
		t.Fatalf("query token accepted while disabled: %q", got)
	}
	if got := Token(ctx, true); got != "query-token" {
		t.Fatalf("query token opt-in = %q", got)
	}
}
