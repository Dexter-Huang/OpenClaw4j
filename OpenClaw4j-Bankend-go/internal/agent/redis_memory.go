package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/seaskyland/openclaw4j-backend-go/internal/chat"
)

const defaultMemoryTTL = 30 * 24 * time.Hour

type RedisStore struct {
	client redis.UniversalClient
	ttl    time.Duration
}

func NewRedisStore(client redis.UniversalClient, ttl time.Duration) *RedisStore {
	if ttl <= 0 {
		ttl = defaultMemoryTTL
	}
	return &RedisStore{client: client, ttl: ttl}
}

func (s *RedisStore) Load(ctx context.Context, workspaceID, conversationID string, limit int) ([]chat.Message, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("agent memory redis client is unavailable")
	}
	values, err := s.client.LRange(ctx, memoryKey(workspaceID, conversationID), int64(-limit), -1).Result()
	if err != nil {
		return nil, fmt.Errorf("load agent memory: %w", err)
	}
	result := make([]chat.Message, 0, len(values))
	for _, value := range values {
		var message chat.Message
		if err := json.Unmarshal([]byte(value), &message); err != nil {
			return nil, fmt.Errorf("decode agent memory: %w", err)
		}
		result = append(result, message)
	}
	return result, nil
}
func (s *RedisStore) Append(ctx context.Context, workspaceID, conversationID string, messages ...chat.Message) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("agent memory redis client is unavailable")
	}
	values := make([]interface{}, 0, len(messages))
	for _, message := range messages {
		value, err := json.Marshal(message)
		if err != nil {
			return err
		}
		values = append(values, string(value))
	}
	key := memoryKey(workspaceID, conversationID)
	pipe := s.client.TxPipeline()
	if len(values) > 0 {
		pipe.RPush(ctx, key, values...)
	}
	pipe.Expire(ctx, key, s.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("append agent memory: %w", err)
	}
	return nil
}
func memoryKey(workspaceID, conversationID string) string {
	return "agent:memory:" + workspaceID + ":" + conversationID
}

var _ MemoryStore = (*RedisStore)(nil)
