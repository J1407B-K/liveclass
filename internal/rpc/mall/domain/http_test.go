package domain

import (
	"errors"
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

func TestIsMySQLDeadlock(t *testing.T) {
	if !isMySQLDeadlock(&mysqlDriver.MySQLError{Number: 1213, Message: "deadlock"}) {
		t.Fatal("expected MySQL 1213 to be recognized as a deadlock")
	}
	if isMySQLDeadlock(errors.New("ordinary failure")) {
		t.Fatal("ordinary errors must not be retried as deadlocks")
	}
}

func TestBranchMaxAttempts(t *testing.T) {
	t.Setenv("LIVECLASS_MALL_BRANCH_MAX_ATTEMPTS", "1")
	if got := branchMaxAttempts(); got != 1 {
		t.Fatalf("branchMaxAttempts() = %d, want 1", got)
	}
	t.Setenv("LIVECLASS_MALL_BRANCH_MAX_ATTEMPTS", "invalid")
	if got := branchMaxAttempts(); got != 3 {
		t.Fatalf("invalid override should use default 3, got %d", got)
	}
}
