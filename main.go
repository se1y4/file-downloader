package main

import (
    "flag"
    "log"
    "path/filepath"
    
    "file-downloader/internal/downloader"
    "file-downloader/internal/server"
    "file-downloader/internal/storage"
    "file-downloader/internal/task"
)

func main() {
    var (
        addr         = flag.String("addr", ":8080", "HTTP server address")
        storageDir   = flag.String("storage-dir", "./storage", "Directory for task storage")
        downloadDir  = flag.String("download-dir", "./downloads", "Directory for downloaded files")
        workers      = flag.Int("workers", 3, "Number of download workers")
    )
    flag.Parse()
    
    absStorageDir, err := filepath.Abs(*storageDir)
    if err != nil {
        log.Fatalf("Failed to get absolute storage path: %v", err)
    }
    
    absDownloadDir, err := filepath.Abs(*downloadDir)
    if err != nil {
        log.Fatalf("Failed to get absolute download path: %v", err)
    }
    
    fileStorage, err := storage.NewFileStorage(absStorageDir)
    if err != nil {
        log.Fatalf("Failed to create file storage: %v", err)
    }
    
    fileDownloader := downloader.NewDownloader(absDownloadDir)
    downloaderAdapter := task.NewDownloaderAdapter(fileDownloader)
    
    taskManager := task.NewTaskManager(fileStorage, downloaderAdapter, *workers)
    
    if err := taskManager.RestoreTasks(); err != nil {
        log.Printf("Warning: failed to restore tasks: %v", err)
    }
    
    httpServer := server.NewHTTPServer(*addr, taskManager)
    
    if err := httpServer.Start(); err != nil {
        log.Fatalf("Server error: %v", err)
    }
}