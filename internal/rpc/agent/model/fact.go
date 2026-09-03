package model

type FactExtractInput struct {
	UserID  int64  `json:"user_id"`
	ConvID  string `json:"conv_id"`
	Message string `json:"message"`
}

type FactCandidate struct {
	FactType    string  `json:"fact_type"`
	ConflictKey string  `json:"conflict_key,omitempty"`
	Content     string  `json:"content"`
	Confidence  float64 `json:"confidence"`
}
