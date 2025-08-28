package utils

import (
	"fmt"
	"time"
)

const (
	LikersTTL      = 30 * time.Second
	NewLikersTTL   = 20 * time.Second
	LikersCountTTL = 15 * time.Second
)

func LikersKey(recipient string, token string) string {
	return fmt.Sprintf("likers:%s:%s", recipient, token)
}
func NewLikersKey(recipient string, token string) string {
	return fmt.Sprintf("newlikers:%s:%s", recipient, token)
}
func LikersCountKey(recipient string) string {
	return fmt.Sprintf("likerscount:%s", recipient)
}
