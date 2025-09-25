package task

import (
    "encoding/json"
    "fmt"
    "net/url"
    "time"
    
    "github.com/google/uuid"
)

type TaskStatus string

const (
    StatusPending    TaskStatus = "pending"
    StatusProcessing TaskStatus = "processing"
    StatusCompleted  TaskStatus = "completed"
    StatusFailed     TaskStatus = "failed"
)

type DownloadTask struct {
    ID        string       `json:"id"`
    URLs      []string     `json:"urls"`
    Status    TaskStatus   `json:"status"`
    CreatedAt time.Time    `json:"created_at"`
    UpdatedAt time.Time    `json:"updated_at"`
    Results   []FileResult `json:"results,omitempty"`
}

type FileResult struct {
    URL      string `json:"url"`
    FileName string `json:"file_name,omitempty"`
    Error    string `json:"error,omitempty"`
    Size     int64  `json:"size,omitempty"`
}

func NewDownloadTask(urls []string) (*DownloadTask, error) {
    if len(urls) == 0 {
        return nil, fmt.Errorf("urls list cannot be empty")
    }
    
    for _, u := range urls {
        if _, err := url.ParseRequestURI(u); err != nil {
            return nil, fmt.Errorf("invalid URL: %s", u)
        }
    }
    
    return &DownloadTask{
        ID:        uuid.New().String(),
        URLs:      urls,
        Status:    StatusPending,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        Results:   make([]FileResult, 0, len(urls)),
    }, nil
}

func (t *DownloadTask) UpdateStatus(status TaskStatus) {
    t.Status = status
    t.UpdatedAt = time.Now()
}

func (t *DownloadTask) AddResult(result FileResult) {
    t.Results = append(t.Results, result)
}

func (t *DownloadTask) IsFinished() bool {
    return t.Status == StatusCompleted || t.Status == StatusFailed
}

func (t *DownloadTask) MarshalJSON() ([]byte, error) {
    type alias DownloadTask
    return json.Marshal(&struct {
        *alias
        CreatedAt string `json:"created_at"`
        UpdatedAt string `json:"updated_at"`
    }{
        alias:     (*alias)(t),
        CreatedAt: t.CreatedAt.Format(time.RFC3339),
        UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
    })
}