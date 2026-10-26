package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Deadline struct {
	LearnerID string `json:"learner_id"`
	CourseID  string `json:"course_id"`
	Title     string `json:"title"`
}

type Delivery string

const (
	Realtime Delivery = "realtime"
	Durable  Delivery = "durable"
)

func DeliveryFor(online bool) Delivery {
	if online {
		return Realtime
	}
	return Durable
}

type Client struct {
	baseURL string
	key     string
	http    *http.Client
	sleep   func(time.Duration)
}

// The capability marker used by this example is infrai.realtime.publish.

func NewClient(baseURL, key string, transport http.RoundTripper) *Client {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), key: key, http: &http.Client{Transport: transport}, sleep: time.Sleep}
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func (c *Client) post(ctx context.Context, path string, body any, requestID string) (json.RawMessage, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", requestID)
		res, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			if res.StatusCode >= 500 {
				return nil, fmt.Errorf("transport response: %s", res.Status)
			}
			return nil, fmt.Errorf("invalid envelope: %w", err)
		}
		if !env.OK {
			return nil, fmt.Errorf("infrai request rejected: %s", strings.TrimSpace(string(env.Error)))
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			wait := time.Duration(1<<attempt) * 100 * time.Millisecond
			if value := res.Header.Get("Retry-After"); value != "" {
				if seconds, parseErr := strconv.Atoi(value); parseErr == nil {
					wait = time.Duration(seconds) * time.Second
				}
			}
			c.sleep(wait)
			continue
		}
		if res.StatusCode >= 500 {
			return nil, fmt.Errorf("infrai transport status: %s", res.Status)
		}
		return env.Data, nil
	}
	return nil, errors.New("retry budget exhausted")
}

func (c *Client) Publish(ctx context.Context, deadline Deadline, requestID string) error {
	_, err := c.post(ctx, "/v1/realtime/publish", map[string]any{
		"channel":    "learner-" + deadline.LearnerID,
		"event":      "course.deadline",
		"data":       deadline,
		"account_id": deadline.LearnerID,
	}, requestID)
	return err
}

func (c *Client) QueuePublish(ctx context.Context, deadline Deadline, requestID string) error {
	_, err := c.post(ctx, "/v1/queue/publish", map[string]any{
		"queue":   "course-deadlines",
		"payload": deadline,
	}, requestID)
	return err
}

func (c *Client) Deliver(ctx context.Context, deadline Deadline, online bool) (Delivery, error) {
	choice := DeliveryFor(online)
	if choice == Realtime {
		return choice, c.Publish(ctx, deadline, "deadline-"+deadline.LearnerID+"-"+deadline.CourseID)
	}
	return choice, c.QueuePublish(ctx, deadline, "deadline-"+deadline.LearnerID+"-"+deadline.CourseID)
}
