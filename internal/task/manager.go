package task

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"
)

type TaskManager struct {
    tasks      map[string]*DownloadTask
    mu         sync.RWMutex
    storage    Storage
    downloader FileDownloader
    workQueue  chan *DownloadTask
    workers    int
    ctx        context.Context
    cancel     context.CancelFunc
    wg         sync.WaitGroup
}

type Storage interface {
    SaveTask(task *DownloadTask) error
    GetTask(taskID string) (*DownloadTask, error)
    GetAllTasks() (map[string]*DownloadTask, error)
}

type FileDownloader interface {
    DownloadFile(url string) (FileResult, error)
}

func NewTaskManager(storage Storage, downloader FileDownloader, workers int) *TaskManager {
    ctx, cancel := context.WithCancel(context.Background())
    
    tm := &TaskManager{
        tasks:      make(map[string]*DownloadTask),
        storage:    storage,
        downloader: downloader,
        workQueue:  make(chan *DownloadTask, 100),
        workers:    workers,
        ctx:        ctx,
        cancel:     cancel,
    }
    
    tm.startWorkers()
    return tm
}

func (tm *TaskManager) startWorkers() {
    for i := 0; i < tm.workers; i++ {
        tm.wg.Add(1)
        go tm.worker(i)
    }
}

func (tm *TaskManager) worker(id int) {
    defer tm.wg.Done()
    
    for {
        select {
        case task := <-tm.workQueue:
            log.Printf("Worker %d processing task %s", id, task.ID)
            tm.processTask(task)
        case <-tm.ctx.Done():
            log.Printf("Worker %d stopping", id)
            return
        }
    }
}

func (tm *TaskManager) processTask(task *DownloadTask) {
    tm.mu.Lock()
    task.UpdateStatus(StatusProcessing)
    tm.saveTask(task)
    tm.mu.Unlock()
    
    allSuccess := true
    
    for _, url := range task.URLs {
        select {
        case <-tm.ctx.Done():
            log.Printf("Task %s interrupted during processing", task.ID)
            return
        default:
            result, err := tm.downloader.DownloadFile(url)
            if err != nil {
                result.Error = err.Error()
                allSuccess = false
                log.Printf("Failed to download %s: %v", url, err)
            } else {
                log.Printf("Successfully downloaded %s to %s", url, result.FileName)
            }
            
            tm.mu.Lock()
            task.AddResult(result)
            tm.saveTask(task)
            tm.mu.Unlock()
        }
    }
    
    tm.mu.Lock()
    defer tm.mu.Unlock()
    
    if allSuccess {
        task.UpdateStatus(StatusCompleted)
    } else {
        task.UpdateStatus(StatusFailed)
    }
    tm.saveTask(task)
}

func (tm *TaskManager) CreateTask(urls []string) (*DownloadTask, error) {
    task, err := NewDownloadTask(urls)
    if err != nil {
        return nil, err
    }
    
    tm.mu.Lock()
    defer tm.mu.Unlock()
    
    tm.tasks[task.ID] = task
    if err := tm.storage.SaveTask(task); err != nil {
        return nil, err
    }
    
    select {
    case tm.workQueue <- task:
        log.Printf("Task %s queued for processing", task.ID)
    default:
        return nil, fmt.Errorf("work queue is full")
    }
    
    return task, nil
}

func (tm *TaskManager) GetTask(taskID string) (*DownloadTask, error) {
    tm.mu.RLock()
    defer tm.mu.RUnlock()
    
    task, exists := tm.tasks[taskID]
    if !exists {
        return nil, fmt.Errorf("task not found")
    }
    
    return task, nil
}

func (tm *TaskManager) GetAllTasks() map[string]*DownloadTask {
    tm.mu.RLock()
    defer tm.mu.RUnlock()
    
    tasks := make(map[string]*DownloadTask)
    for id, task := range tm.tasks {
        tasks[id] = task
    }
    
    return tasks
}

func (tm *TaskManager) saveTask(task *DownloadTask) {
    if err := tm.storage.SaveTask(task); err != nil {
        log.Printf("Error saving task %s: %v", task.ID, err)
    }
}

func (tm *TaskManager) RestoreTasks() error {
    tasks, err := tm.storage.GetAllTasks()
    if err != nil {
        return err
    }
    
    tm.mu.Lock()
    defer tm.mu.Unlock()
    
    for id, task := range tasks {
        tm.tasks[id] = task
        
        if !task.IsFinished() {
            task.UpdateStatus(StatusPending)
            select {
            case tm.workQueue <- task:
                log.Printf("Restored and requeued task %s", task.ID)
            default:
                log.Printf("Warning: could not requeue restored task %s (queue full)", task.ID)
            }
        }
    }
    
    return nil
}

func (tm *TaskManager) Shutdown(timeout time.Duration) {
    log.Println("Shutting down task manager...")
    
    tm.cancel()
    
    done := make(chan struct{})
    go func() {
        tm.wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        log.Println("All workers stopped gracefully")
    case <-time.After(timeout):
        log.Println("Warning: shutdown timeout, forcing stop")
    }
}