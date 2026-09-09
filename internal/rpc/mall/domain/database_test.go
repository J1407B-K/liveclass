package domain

import (
	"strings"
	"testing"
)

func TestDSNForRoleUsesIndependentDefaults(t *testing.T) {
	for _, key := range []string{
		"LIVECLASS_MALL_MYSQL_DSN",
		"LIVECLASS_MALL_ORDER_MYSQL_DSN",
		"LIVECLASS_MALL_INVENTORY_MYSQL_DSN",
		"LIVECLASS_MALL_POINTS_MYSQL_DSN",
	} {
		t.Setenv(key, "")
	}
	wantDatabase := map[DatabaseRole]string{
		MallDatabase:      "/liveclass_mall?",
		OrderDatabase:     "/liveclass_mall_order?",
		InventoryDatabase: "/liveclass_mall_inventory?",
		PointsDatabase:    "/liveclass_mall_points?",
	}
	seen := map[string]bool{}
	for role, want := range wantDatabase {
		dsn, err := DSNForRole(role)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(dsn, want) {
			t.Fatalf("DSNForRole(%q) = %q, want database marker %q", role, dsn, want)
		}
		if seen[dsn] {
			t.Fatalf("role %q reused another resource's DSN", role)
		}
		seen[dsn] = true
	}
}

func TestDSNForRoleHonorsRoleSpecificEnvironment(t *testing.T) {
	want := "mall_user:secret@tcp(db.internal:3306)/inventory_prod?parseTime=true"
	t.Setenv("LIVECLASS_MALL_INVENTORY_MYSQL_DSN", want)
	got, err := DSNForRole(InventoryDatabase)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("DSNForRole(inventory) = %q, want %q", got, want)
	}
}

func TestDSNForRoleRejectsUnknownRole(t *testing.T) {
	if _, err := DSNForRole("unknown"); err == nil {
		t.Fatal("expected unknown role to fail")
	}
}
