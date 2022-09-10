package data

type Message struct {
	Value     string `json:"value" binding:"required"`
	Timestamp int64  `json:"timestamp,omitempty"`
}
