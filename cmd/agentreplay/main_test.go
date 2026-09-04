package main

import (
	"reflect"
	"testing"
)

func TestExtractCitations(t *testing.T) {
	answer := "事实一[课程.md#章节一]，重复[课程.md#章节一]；事实二[工程讲义.md#章节二]。"
	want := []string{"课程.md#章节一", "工程讲义.md#章节二"}
	if got := extractCitations(answer); !reflect.DeepEqual(got, want) {
		t.Fatalf("extractCitations()=%v want=%v", got, want)
	}
}

func TestNewEvalRequestIDFitsDatabaseColumn(t *testing.T) {
	id := newEvalRequestID("variant-name-that-is-intentionally-too-long")
	if len(id) > 64 {
		t.Fatalf("request id length=%d id=%q", len(id), id)
	}
}
