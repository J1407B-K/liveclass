package model

type File struct {
	MimeType      string `json:"mimeType"`
	ID            string `json:"id"`
	DataURL       string `json:"dataURL"`
	Created       int64  `json:"created"`
	LastRetrieved int64  `json:"lastRetrieved"`
}
