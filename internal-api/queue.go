package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const jobsKey = "jobs"

type Job struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

func connectRedis(addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  0,
		WriteTimeout: 5 * time.Second,
	})

	var lastErr error
	for i := 0; i < 30; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := client.Ping(ctx).Err()
		cancel()
		if err == nil {
			return client, nil
		}
		lastErr = err
		time.Sleep(time.Second)
	}
	return nil, lastErr
}

func enqueueJob(ctx context.Context, client *redis.Client, jobType string, payload map[string]any) (Job, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	job := Job{
		ID:        fmt.Sprintf("job-%d", time.Now().UnixNano()),
		Type:      jobType,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	}
	raw, err := json.Marshal(job)
	if err != nil {
		return Job{}, err
	}
	if err := client.RPush(ctx, jobsKey, raw).Err(); err != nil {
		return Job{}, err
	}
	return job, nil
}

func queueLength(ctx context.Context, client *redis.Client) (int64, error) {
	return client.LLen(ctx, jobsKey).Result()
}

func dequeueJob(ctx context.Context, client *redis.Client, block time.Duration) (Job, bool, error) {
	result, err := client.BLPop(ctx, block, jobsKey).Result()
	if err == redis.Nil {
		return Job{}, false, nil
	}
	if err != nil {
		return Job{}, false, err
	}
	if len(result) < 2 {
		return Job{}, false, nil
	}

	var job Job
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return Job{}, false, err
	}
	if job.Payload == nil {
		job.Payload = map[string]any{}
	}
	return job, true, nil
}
