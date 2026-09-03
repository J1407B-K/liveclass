package session

import "testing"

func TestParseSummaryExtractsJSON(t *testing.T) {
	got, err := parseSummary("```json\n{\"summary\":\"保留事实\",\"important_facts\":[\"A\"],\"decisions\":[],\"unfinished_tasks\":[]}\n```")
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary != "保留事实" || len(got.ImportantFacts) != 1 {
		t.Fatalf("unexpected summary: %#v", got)
	}
}

func TestParseSummaryRejectsEmpty(t *testing.T) {
	if _, err := parseSummary(`{"summary":""}`); err == nil {
		t.Fatal("expected validation error")
	}
}
