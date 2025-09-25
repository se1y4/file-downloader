package downloader

import (
    "fmt"
	"log"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"
)

type Downloader struct {
    downloadDir string
    client      *http.Client
}

type DownloadResult struct {
    URL      string `json:"url"`
    FileName string `json:"file_name,omitempty"`
    Error    string `json:"error,omitempty"`
    Size     int64  `json:"size,omitempty"`
}

func NewDownloader(downloadDir string) *Downloader {
    if err := os.MkdirAll(downloadDir, 0755); err != nil {
        log.Printf("Warning: cannot create download directory: %v", err)
    }
    
    return &Downloader{
        downloadDir: downloadDir,
        client: &http.Client{
            Timeout: 30 * time.Minute,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxConnsPerHost:     10,
                IdleConnTimeout:     90 * time.Second,
                TLSHandshakeTimeout: 10 * time.Second,
            },
        },
    }
}

func (d *Downloader) DownloadFile(url string) (DownloadResult, error) {
    result := DownloadResult{URL: url}
    
    resp, err := d.client.Get(url)
    if err != nil {
        return result, fmt.Errorf("HTTP request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return result, fmt.Errorf("HTTP status %d: %s", resp.StatusCode, resp.Status)
    }
    
    fileName := d.getFileName(url, resp)
    filePath := filepath.Join(d.downloadDir, fileName)
    
    file, err := os.Create(filePath)
    if err != nil {
        return result, fmt.Errorf("failed to create file: %w", err)
    }
    defer file.Close()
    
    size, err := io.Copy(file, resp.Body)
    if err != nil {
        os.Remove(filePath)
        return result, fmt.Errorf("failed to write file: %w", err)
    }
    
    result.FileName = fileName
    result.Size = size
    return result, nil
}

func (d *Downloader) getFileName(url string, resp *http.Response) string {
    contentDisposition := resp.Header.Get("Content-Disposition")
    if contentDisposition != "" {
        if strings.Contains(contentDisposition, "filename=") {
            parts := strings.Split(contentDisposition, "filename=")
            if len(parts) > 1 {
                filename := strings.Trim(parts[1], `"`)
                if filename != "" {
                    return sanitizeFileName(filename)
                }
            }
        }
    }
    
    path := resp.Request.URL.Path
    if path != "" {
        baseName := filepath.Base(path)
        if baseName != "" && baseName != "." && baseName != "/" {
            return sanitizeFileName(baseName)
        }
    }
    
    return fmt.Sprintf("downloaded_file_%d", time.Now().UnixNano())
}

func sanitizeFileName(name string) string {
    name = strings.ReplaceAll(name, "/", "_")
    name = strings.ReplaceAll(name, "\\", "_")
    name = strings.ReplaceAll(name, ":", "_")
    name = strings.ReplaceAll(name, "*", "_")
    name = strings.ReplaceAll(name, "?", "_")
    name = strings.ReplaceAll(name, "\"", "_")
    name = strings.ReplaceAll(name, "<", "_")
    name = strings.ReplaceAll(name, ">", "_")
    name = strings.ReplaceAll(name, "|", "_")
    return name
}