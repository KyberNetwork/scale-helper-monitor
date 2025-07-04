package slack

// Message represents a Slack message
type Message struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment represents a Slack message attachment
type Attachment struct {
	Color  string  `json:"color"`
	Title  string  `json:"title"`
	Fields []Field `json:"fields"`
}

// Field represents a Slack message field
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
} 