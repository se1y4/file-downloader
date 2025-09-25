package storage

import (
    "encoding/json"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "sync"
    
    "file-downloader/internal/task"
)

type FileStorage struct {
    storageDir string
    mu         sync.RWMutex
}

func NewFileStorage(storageDir string) (*FileStorage, error) {
    if err := os.MkdirAll(storageDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create storage directory: %w", err)
    }
    
    return &FileStorage{
        storageDir: storageDir,
    }, nil
}

func (fs *FileStorage) getTaskPath(taskID string) string {
    return filepath.Join(fs.storageDir, fmt.Sprintf("task_%s.json", taskID))
}

func (fs *FileStorage) SaveTask(task *task.DownloadTask) error {
    fs.mu.Lock()
    defer fs.mu.Unlock()
    
    data, err := json.MarshalIndent(task, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal task: %w", err)
    }
    
    taskPath := fs.getTaskPath(task.ID)
    tmpPath := taskPath + ".tmp"
    
    if err := os.WriteFile(tmpPath, data, 0644); err != nil {
        return fmt.Errorf("failed to write task file: %w", err)
    }
    
    if err := os.Rename(tmpPath, taskPath); err != nil {
        return fmt.Errorf("failed to rename task file: %w", err)
    }
    
    return nil
}

func (fs *FileStorage) GetTask(taskID string) (*task.DownloadTask, error) {
    fs.mu.RLock()
    defer fs.mu.RUnlock()
    
    taskPath := fs.getTaskPath(taskID)
    data, err := os.ReadFile(taskPath)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, fmt.Errorf("task not found")
        }
        return nil, fmt.Errorf("failed to read task file: %w", err)
    }
    
    var task task.DownloadTask
    if err := json.Unmarshal(data, &task); err != nil {
        return nil, fmt.Errorf("failed to unmarshal task: %w", err)
    }
    
    return &task, nil
}

func (fs *FileStorage) GetAllTasks() (map[string]*task.DownloadTask, error) {
    fs.mu.RLock()
    defer fs.mu.RUnlock()
    
    entries, err := os.ReadDir(fs.storageDir)
    if err != nil {
        return nil, fmt.Errorf("failed to read storage directory: %w", err)
    }
    
    tasks := make(map[string]*task.DownloadTask)
    
    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }
        
        if matched, _ := filepath.Match("task_*.json", entry.Name()); matched {
            taskPath := filepath.Join(fs.storageDir, entry.Name())
            data, err := os.ReadFile(taskPath)
            if err != nil {
                log.Printf("Warning: failed to read task file %s: %v", taskPath, err)
                continue
            }
            
            var task task.DownloadTask
            if err := json.Unmarshal(data, &task); err != nil {
                log.Printf("Warning: failed to unmarshal task file %s: %v", taskPath, err)
                continue
            }
            
            tasks[task.ID] = &task
        }
    }
    
    return tasks, nil
}