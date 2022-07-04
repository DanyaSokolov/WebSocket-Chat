package chat

import "goChat/utils"

type Message struct {
	ID     int64  `json:"id"`
	Body   string `json:"body"`
	Sender string `json:"sender"`
}

func NewMessage(body string, sender string) *Message {
	return &Message{
		ID:     utils.GetRandomI64(),
		Sender: sender,
		Body:   body,
	}
}
