package task

import (
    "file-downloader/internal/downloader"
)

type downloaderAdapter struct {
    downloader *downloader.Downloader
}

func NewDownloaderAdapter(dl *downloader.Downloader) FileDownloader {
    return &downloaderAdapter{downloader: dl}
}

func (a *downloaderAdapter) DownloadFile(url string) (FileResult, error) {
    result, err := a.downloader.DownloadFile(url)
    if err != nil {
        return FileResult{}, err
    }
    
    return FileResult{
        URL:      result.URL,
        FileName: result.FileName,
        Error:    result.Error,
        Size:     result.Size,
    }, nil
}