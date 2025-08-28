package utils

import (
	"encoding/base64"
	"encoding/json"
)

type Cursor struct {
	LastCreatedAt int64
	Limit         int
}

func (c *Cursor) Encode() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(data), nil
}

func DecodeCursor(encodedCursor string) (*Cursor, error) {
	if encodedCursor == "" {
		return nil, nil
	}

	data, err := base64.URLEncoding.DecodeString(encodedCursor)
	if err != nil {
		return nil, err
	}

	var c Cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
