package server

import (
    "encoding/json"
    "net/http"
    "time"
    
    "github.com/gorilla/mux"
    
    "file-downloader/internal/task"
)

type CreateTaskRequest struct {
    URLs []string `json:"urls"`
}

type ErrorResponse struct {
    Error string `json:"error"`
}

type Server struct {
    taskManager *task.TaskManager
    router      *mux.Router
}

func NewServer(taskManager *task.TaskManager) *Server {
    s := &Server{
        taskManager: taskManager,
        router:      mux.NewRouter(),
    }
    
    s.setupRoutes()
    return s
}

func (s *Server) setupRoutes() {
    s.router.HandleFunc("/tasks", s.createTask).Methods("POST")
    s.router.HandleFunc("/tasks/{id}", s.getTask).Methods("GET")
    s.router.HandleFunc("/tasks", s.listTasks).Methods("GET")
    s.router.HandleFunc("/health", s.healthCheck).Methods("GET")
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
    var req CreateTaskRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        s.sendError(w, http.StatusBadRequest, "Invalid JSON payload")
        return
    }
    
    if len(req.URLs) == 0 {
        s.sendError(w, http.StatusBadRequest, "URLs list cannot be empty")
        return
    }
    
    task, err := s.taskManager.CreateTask(req.URLs)
    if err != nil {
        s.sendError(w, http.StatusInternalServerError, err.Error())
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(task)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    taskID := vars["id"]
    
    task, err := s.taskManager.GetTask(taskID)
    if err != nil {
        s.sendError(w, http.StatusNotFound, "Task not found")
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(task)
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
    tasks := s.taskManager.GetAllTasks()
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(tasks)
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
    response := map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now().UTC().Format(time.RFC3339),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (s *Server) sendError(w http.ResponseWriter, status int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    s.router.ServeHTTP(w, r)
}