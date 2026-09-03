package eval

import "testing"

func TestEvaluateRetrievalAndProcessMetrics(t *testing.T) {
	cases := []Case{{ID: "1", ExpectedSkill: "student_qa", ExpectedTools: []string{}, ForbiddenTools: []string{"query_on_internet"}, GoldDocs: []string{"doc-a"}, ExpectedResult: ExpectedResult{Contains: []string{"蓝山"}}}}
	predictions := []Prediction{{CaseID: "1", Skill: "student_qa", RetrievedDocs: []string{"doc-a"}, CitedDocs: []string{"doc-a"}, Answer: "由蓝山工作室开发", Tokens: 100, Steps: 2}}
	r := Evaluate("v2", cases, predictions)
	if r.TaskSuccess != 1 || r.HitAt1 != 1 || r.MRR != 1 || r.Faithfulness != 1 {
		t.Fatalf("unexpected report: %#v", r)
	}
}

func TestEvaluateDetectsForbiddenAndDuplicateTools(t *testing.T) {
	cases := []Case{{ID: "1", ForbiddenTools: []string{"danger"}}}
	predictions := []Prediction{{CaseID: "1", Tools: []string{"danger", "danger"}}}
	r := Evaluate("v", cases, predictions)
	if r.ConstraintViolationRate != 1 || r.DuplicateToolCalls != 1 || r.TaskSuccess != 0 {
		t.Fatalf("unexpected report: %#v", r)
	}
}
