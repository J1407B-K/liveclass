package model

type FactCandidate struct {
	FactType   string  `json:"fact_type"`
	Content    string  `json:"content"`
	Confidence float64 `json:"confidence"`
}
