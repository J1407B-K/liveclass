package webrtc

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pion/webrtc/v4"
)

func DecodeSDP(in string) (webrtc.SessionDescription, error) {
	raw, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	var sd webrtc.SessionDescription
	return sd, json.Unmarshal(raw, &sd)
}

func EncodeSDP(sd *webrtc.SessionDescription) (string, error) {
	raw, err := json.Marshal(sd)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}
