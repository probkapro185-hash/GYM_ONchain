package domain

import "time"

type AIChatRequest struct {
	ConversationID *int64 `json:"conversation_id,omitempty"`
	Message        string `json:"message"`
	UserID         int64  `json:"-"`
	Role           Role   `json:"-"`
}

type AIChatResponse struct {
	ConversationID int64      `json:"conversation_id"`
	Reply          string     `json:"reply"`
	Sources        []AISource `json:"sources,omitempty"`
}

type AISource struct {
	DocumentID int64   `json:"document_id"`
	ChunkID    int64   `json:"chunk_id"`
	Title      string  `json:"title"`
	Source     string  `json:"source"`
	Similarity float64 `json:"similarity"`
}

type AIDocument struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Source     string    `json:"source"`
	Checksum   string    `json:"checksum"`
	ChunkCount int       `json:"chunk_count"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type AIConversation struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"-"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AIMessage struct {
	ID             int64      `json:"id"`
	ConversationID int64      `json:"conversation_id"`
	Role           string     `json:"role"`
	Content        string     `json:"content"`
	Sources        []AISource `json:"sources,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type AIChunkInput struct {
	Position  int
	Content   string
	Embedding []float64
}

type AIRetrievedChunk struct {
	DocumentID int64
	ChunkID    int64
	Title      string
	Source     string
	Content    string
	Similarity float64
}

type AIPersonalContext struct {
	FullName          string
	Role              Role
	Balance           float64
	Visits            int
	SubscriptionName  string
	SessionsLeft      *int
	SubscriptionUntil *time.Time
	UpcomingTrainings []AIUpcomingTraining
}

type AIUpcomingTraining struct {
	StartsAt   time.Time
	EndsAt     time.Time
	Trainer    string
	TrainingID int64
}
