package server

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "file-downloader/internal/task"
)

type HTTPServer struct {
    server      *http.Server
    taskManager *task.TaskManager
}

func NewHTTPServer(addr string, taskManager *task.TaskManager) *HTTPServer {
    s := NewServer(taskManager)
    
    return &HTTPServer{
        server: &http.Server{
            Addr:         addr,
            Handler:      s,
            ReadTimeout:  15 * time.Second,
            WriteTimeout: 15 * time.Second,
            IdleTimeout:  60 * time.Second,
        },
        taskManager: taskManager,
    }
}

func (s *HTTPServer) Start() error {
    log.Printf("Starting HTTP server on %s", s.server.Addr)
    
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("HTTP server error: %v", err)
        }
    }()
    
    <-stop
    log.Println("Shutdown signal received")
    
    return s.Shutdown(30 * time.Second)
}

func (s *HTTPServer) Shutdown(timeout time.Duration) error {
    log.Println("Starting graceful shutdown...")
    
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    
    if err := s.server.Shutdown(ctx); err != nil {
        return err
    }
    
    s.taskManager.Shutdown(timeout)
    
    log.Println("Server shutdown completed")
    return nil
}