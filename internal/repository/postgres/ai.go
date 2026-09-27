package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sfedu-crm/internal/domain"
)

type aiRepository struct{ pool *pgxpool.Pool }

func NewAIRepository(pool *pgxpool.Pool) *aiRepository { return &aiRepository{pool: pool} }

func (r *aiRepository) CreateConversation(ctx context.Context, userID int64, title string) (*domain.AIConversation, error) {
	var c domain.AIConversation
	err := queryer(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO ai_conversations(user_id,title) VALUES($1,$2)
		RETURNING id,user_id,title,created_at,updated_at`, userID, title).
		Scan(&c.ID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, mapDBError(err)
	}
	return &c, nil
}

func (r *aiRepository) GetConversation(ctx context.Context, conversationID, userID int64) (*domain.AIConversation, error) {
	var c domain.AIConversation
	err := queryer(ctx, r.pool).QueryRow(ctx, `
		SELECT id,user_id,title,created_at,updated_at
		FROM ai_conversations WHERE id=$1 AND user_id=$2`, conversationID, userID).
		Scan(&c.ID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, mapDBError(err)
	}
	return &c, nil
}

func (r *aiRepository) ListConversations(ctx context.Context, userID int64, limit int) ([]*domain.AIConversation, error) {
	rows, err := queryer(ctx, r.pool).Query(ctx, `
		SELECT id,user_id,title,created_at,updated_at
		FROM ai_conversations WHERE user_id=$1
		ORDER BY updated_at DESC,id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	result := make([]*domain.AIConversation, 0)
	for rows.Next() {
		var c domain.AIConversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &c)
	}
	return result, rows.Err()
}

func (r *aiRepository) DeleteConversation(ctx context.Context, conversationID, userID int64) error {
	tag, err := queryer(ctx, r.pool).Exec(ctx, `DELETE FROM ai_conversations WHERE id=$1 AND user_id=$2`, conversationID, userID)
	if err != nil {
		return mapDBError(err)
	}
	return requireAffected(tag)
}

func (r *aiRepository) AddMessage(ctx context.Context, conversationID int64, role, content string, sources []domain.AISource) (*domain.AIMessage, error) {
	if sources == nil {
		sources = []domain.AISource{}
	}
	sourcesJSON, err := json.Marshal(sources)
	if err != nil {
		return nil, fmt.Errorf("aiRepository.AddMessage marshal sources: %w", err)
	}
	var m domain.AIMessage
	var storedSources []byte
	err = queryer(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO ai_messages(conversation_id,role,content,sources) VALUES($1,$2,$3,$4::jsonb)
		RETURNING id,conversation_id,role,content,sources,created_at`, conversationID, role, content, string(sourcesJSON)).
		Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &storedSources, &m.CreatedAt)
	if err != nil {
		return nil, mapDBError(err)
	}
	if err := json.Unmarshal(storedSources, &m.Sources); err != nil {
		return nil, fmt.Errorf("aiRepository.AddMessage decode sources: %w", err)
	}
	if _, err := queryer(ctx, r.pool).Exec(ctx, `UPDATE ai_conversations SET updated_at=NOW() WHERE id=$1`, conversationID); err != nil {
		return nil, mapDBError(err)
	}
	return &m, nil
}

func (r *aiRepository) ListMessages(ctx context.Context, conversationID, userID int64, limit int) ([]*domain.AIMessage, error) {
	rows, err := queryer(ctx, r.pool).Query(ctx, `
		SELECT m.id,m.conversation_id,m.role,m.content,m.sources,m.created_at
		FROM ai_messages m
		JOIN ai_conversations c ON c.id=m.conversation_id
		WHERE m.conversation_id=$1 AND c.user_id=$2
		ORDER BY m.id DESC LIMIT $3`, conversationID, userID, limit)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	result := make([]*domain.AIMessage, 0)
	for rows.Next() {
		var m domain.AIMessage
		var sourcesJSON []byte
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &sourcesJSON, &m.CreatedAt); err != nil {
			return nil, err
		}
		if len(sourcesJSON) > 0 {
			if err := json.Unmarshal(sourcesJSON, &m.Sources); err != nil {
				return nil, fmt.Errorf("aiRepository.ListMessages decode sources: %w", err)
			}
		}
		result = append(result, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result, nil
}

func (r *aiRepository) UpsertDocument(ctx context.Context, title, source, checksum string) (int64, bool, error) {
	var id int64
	var oldChecksum string
	err := queryer(ctx, r.pool).QueryRow(ctx, `SELECT id,checksum FROM ai_documents WHERE source=$1`, source).Scan(&id, &oldChecksum)
	if err == nil {
		if oldChecksum == checksum {
			return id, false, nil
		}
		_, err = queryer(ctx, r.pool).Exec(ctx, `UPDATE ai_documents SET title=$1,checksum=$2,updated_at=NOW() WHERE id=$3`, title, checksum, id)
		return id, true, mapDBError(err)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, false, mapDBError(err)
	}
	err = queryer(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO ai_documents(title,source,checksum) VALUES($1,$2,$3) RETURNING id`, title, source, checksum).Scan(&id)
	if err != nil {
		return 0, false, mapDBError(err)
	}
	return id, true, nil
}

func (r *aiRepository) ReplaceDocumentChunks(ctx context.Context, documentID int64, chunks []domain.AIChunkInput) error {
	q := queryer(ctx, r.pool)
	if _, err := q.Exec(ctx, `DELETE FROM ai_chunks WHERE document_id=$1`, documentID); err != nil {
		return mapDBError(err)
	}
	for _, chunk := range chunks {
		vector := vectorLiteral(chunk.Embedding)
		if _, err := q.Exec(ctx, `
			INSERT INTO ai_chunks(document_id,position,content,embedding)
			VALUES($1,$2,$3,$4::vector)`, documentID, chunk.Position, chunk.Content, vector); err != nil {
			return mapDBError(err)
		}
	}
	return nil
}

func (r *aiRepository) SearchChunks(ctx context.Context, embedding []float64, limit int) ([]domain.AIRetrievedChunk, error) {
	vector := vectorLiteral(embedding)
	rows, err := queryer(ctx, r.pool).Query(ctx, `
		SELECT d.id,c.id,d.title,d.source,c.content,
		       1 - (c.embedding <=> $1::vector) AS similarity
		FROM ai_chunks c
		JOIN ai_documents d ON d.id=c.document_id
		ORDER BY c.embedding <=> $1::vector
		LIMIT $2`, vector, limit)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	result := make([]domain.AIRetrievedChunk, 0, limit)
	for rows.Next() {
		var item domain.AIRetrievedChunk
		if err := rows.Scan(&item.DocumentID, &item.ChunkID, &item.Title, &item.Source, &item.Content, &item.Similarity); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *aiRepository) ListDocuments(ctx context.Context) ([]*domain.AIDocument, error) {
	rows, err := queryer(ctx, r.pool).Query(ctx, `
		SELECT d.id,d.title,d.source,d.checksum,COUNT(c.id),d.updated_at
		FROM ai_documents d LEFT JOIN ai_chunks c ON c.document_id=d.id
		GROUP BY d.id ORDER BY d.title`)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	result := make([]*domain.AIDocument, 0)
	for rows.Next() {
		var d domain.AIDocument
		if err := rows.Scan(&d.ID, &d.Title, &d.Source, &d.Checksum, &d.ChunkCount, &d.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &d)
	}
	return result, rows.Err()
}

func (r *aiRepository) DeleteDocument(ctx context.Context, documentID int64) error {
	tag, err := queryer(ctx, r.pool).Exec(ctx, `DELETE FROM ai_documents WHERE id=$1`, documentID)
	if err != nil {
		return mapDBError(err)
	}
	return requireAffected(tag)
}

func (r *aiRepository) GetPersonalContext(ctx context.Context, userID int64, _ domain.Role) (*domain.AIPersonalContext, error) {
	var result domain.AIPersonalContext
	err := queryer(ctx, r.pool).QueryRow(ctx, `
		SELECT full_name,role,balance::float8,visits FROM users WHERE id=$1 AND is_active=true`, userID).
		Scan(&result.FullName, &result.Role, &result.Balance, &result.Visits)
	if err != nil {
		return nil, mapDBError(err)
	}
	if result.Role != domain.RoleClient {
		return &result, nil
	}

	var sessionsLeft sql.NullInt64
	var until time.Time
	var name string
	err = queryer(ctx, r.pool).QueryRow(ctx, `
		SELECT cs.product_name,cs.sessions_left,cs.end_date
		FROM client_subscriptions cs
		WHERE cs.client_id=$1 AND cs.is_active=true AND cs.frozen_at IS NULL AND cs.start_date<=NOW() AND cs.end_date>NOW()
		  AND (cs.sessions_left IS NULL OR cs.sessions_left>0)
		ORDER BY cs.end_date ASC,cs.id ASC LIMIT 1`, userID).Scan(&name, &sessionsLeft, &until)
	if err == nil {
		result.SubscriptionName = name
		if sessionsLeft.Valid {
			value := int(sessionsLeft.Int64)
			result.SessionsLeft = &value
		}
		result.SubscriptionUntil = &until
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, mapDBError(err)
	}

	rows, err := queryer(ctx, r.pool).Query(ctx, `
		SELECT t.id,t.start_time,t.end_time,COALESCE(u.full_name,'Без тренера')
		FROM trainings t
		LEFT JOIN trainers tr ON tr.id=t.trainer_id
		LEFT JOIN users u ON u.id=tr.user_id
		WHERE t.client_id=$1 AND t.status='scheduled' AND t.start_time>=NOW()
		ORDER BY t.start_time LIMIT 5`, userID)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var item domain.AIUpcomingTraining
		if err := rows.Scan(&item.TrainingID, &item.StartsAt, &item.EndsAt, &item.Trainer); err != nil {
			return nil, err
		}
		result.UpcomingTrainings = append(result.UpcomingTrainings, item)
	}
	return &result, rows.Err()
}

func vectorLiteral(values []float64) string {
	var b strings.Builder
	b.Grow(len(values) * 10)
	b.WriteByte('[')
	for i, value := range values {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(value, 'g', -1, 64))
	}
	b.WriteByte(']')
	return b.String()
}
