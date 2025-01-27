// rest/server.go
package rest

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"
    "github.com/gorilla/mux"
    "github.com/asynkron/protoactor-go/actor"
    "reddit/proto"
)

type Server struct {
    router *mux.Router
    engine *actor.PID
    system *actor.ActorSystem
}

type Response struct {
    Success bool        `json:"success"`
    Message string      `json:"message,omitempty"`
    Data    interface{} `json:"data,omitempty"`
}

type RegisterUserRequest struct {
    Username string `json:"username"`
}

type CreateForumRequest struct {
    Name        string `json:"name"`
    Description string `json:"description"`
}

type CreatePostRequest struct {
    Username    string `json:"username"`
    Subreddit   string `json:"subreddit"`
    Title       string `json:"title"`
    Content     string `json:"content"`
    IsRepost    bool   `json:"isRepost"`
    OriginalId  string `json:"originalId"`
}

type CreateCommentRequest struct {
    Username string `json:"username"`
    Content  string `json:"content"`
    ParentId string `json:"parentId"`
}

type VoteRequest struct {
    Username string `json:"username"`
    IsUpvote bool   `json:"isUpvote"`
}

type SendMessageRequest struct {
    SenderUsername   string `json:"senderUsername"`
    ReceiverUsername string `json:"receiverUsername"`
    Content          string `json:"content"`
}

func NewServer(engine *actor.PID, system *actor.ActorSystem) *Server {
    s := &Server{
        router: mux.NewRouter(),
        engine: engine,
        system: system,
    }
    s.setupRoutes()
    return s
}

func (s *Server) setupRoutes() {
    // User routes
    s.router.HandleFunc("/api/users", s.registerUser).Methods("POST")
    s.router.HandleFunc("/api/users/{username}/status", s.updateUserStatus).Methods("PUT")

    // Forum routes
    s.router.HandleFunc("/api/forums", s.createForum).Methods("POST")
    s.router.HandleFunc("/api/forums/{forumName}/join", s.joinForum).Methods("POST")
    s.router.HandleFunc("/api/forums/{forumName}/leave", s.leaveForum).Methods("POST")
    s.router.HandleFunc("/api/forums/{forumName}", s.getForumDetails).Methods("GET")

    // Post routes
    s.router.HandleFunc("/api/posts", s.createPost).Methods("POST")
    s.router.HandleFunc("/api/posts/{postId}", s.getPost).Methods("GET")
    s.router.HandleFunc("/api/posts/{postId}/comments", s.createComment).Methods("POST")
    s.router.HandleFunc("/api/posts/{postId}/vote", s.vote).Methods("POST")

    // Feed routes
    s.router.HandleFunc("/api/feed", s.getFeed).Methods("GET")

    // Message routes
    s.router.HandleFunc("/api/messages", s.sendMessage).Methods("POST")
    s.router.HandleFunc("/api/messages/{username}", s.getMessages).Methods("GET")

    s.router.Use(loggingMiddleware)
    s.router.Use(corsMiddleware)
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        log.Printf("Started %s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
        log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
    })
}

func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

func sendResponse(w http.ResponseWriter, status int, resp Response) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(resp); err != nil {
        log.Printf("Error encoding response: %v", err)
    }
}

func sendError(w http.ResponseWriter, status int, message string) {
    log.Printf("Sending error response: %s", message)
    sendResponse(w, status, Response{
        Success: false,
        Message: message,
    })
}

func (s *Server) registerUser(w http.ResponseWriter, r *http.Request) {
    var req RegisterUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("Failed to decode request body: %v", err)
        sendError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    log.Printf("Processing registration for user: %s", req.Username)
    future := s.system.Root.RequestFuture(s.engine, &proto.OnboardUser{
        UserHandle: req.Username,
    }, 5*time.Second)

    result, err := future.Result()
    if err != nil {
        log.Printf("Failed to process registration: %v", err)
        sendError(w, http.StatusInternalServerError, "Failed to register user")
        return
    }

    response, ok := result.(*proto.OnboardUserResponse)
    if !ok || !response.Success {
        sendError(w, http.StatusBadRequest, response.Message)
        return
    }

    log.Printf("Successfully registered user: %s", req.Username)
    sendResponse(w, http.StatusCreated, Response{
        Success: true,
        Message: response.Message,
    })
}

func (s *Server) updateUserStatus(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    var req struct {
        IsOnline bool `json:"isOnline"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    future := s.system.Root.RequestFuture(s.engine, &proto.ActivityStatus{
        UserHandle: vars["username"],
        IsOnline:   req.IsOnline,
    }, 5*time.Second)

    result, err := future.Result()
    if err != nil {
        sendError(w, http.StatusInternalServerError, "Failed to update status")
        return
    }

    response, ok := result.(*proto.ActivityStatusResponse)
    if !ok || !response.Success {
        sendError(w, http.StatusInternalServerError, response.Message)
        return
    }

    sendResponse(w, http.StatusOK, Response{
        Success: true,
        Message: response.Message,
    })
}

func (s *Server) createForum(w http.ResponseWriter, r *http.Request) {
    var req CreateForumRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    future := s.system.Root.RequestFuture(s.engine, &proto.CreateForum{
        Name: req.Name,
    }, 5*time.Second)

    result, err := future.Result()
    if err != nil {
        sendError(w, http.StatusInternalServerError, "Failed to create forum")
        return
    }

    response, ok := result.(*proto.CreateForumResponse)
    if !ok || !response.Success {
        sendError(w, http.StatusInternalServerError, response.Message)
        return
    }

    sendResponse(w, http.StatusCreated, Response{
        Success: true,
        Message: response.Message,
    })
}

func (s *Server) joinForum(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    var req RegisterUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    future := s.system.Root.RequestFuture(s.engine, &proto.JoinForum{
        UserHandle: req.Username,
        Subreddit:  vars["forumName"],
    }, 5*time.Second)

    result, err := future.Result()
    if err != nil {
        sendError(w, http.StatusInternalServerError, "Failed to join forum")
        return
    }

    response, ok := result.(*proto.JoinForumResponse)
    if !ok || !response.Success {
        sendError(w, http.StatusInternalServerError, response.Message)
        return
    }

    sendResponse(w, http.StatusOK, Response{
        Success: true,
        Message: response.Message,
    })
}

func (s *Server) leaveForum(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    var req RegisterUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    future := s.system.Root.RequestFuture(s.engine, &proto.LeaveForum{
        UserHandle: req.Username,
        Subreddit:  vars["forumName"],
    }, 5*time.Second)

    result, err := future.Result()
    if err != nil {
        sendError(w, http.StatusInternalServerError, "Failed to leave forum")
        return
    }

    response, ok := result.(*proto.LeaveForumResponse)
    if !ok || !response.Success {
        sendError(w, http.StatusInternalServerError, response.Message)
        return
    }

    sendResponse(w, http.StatusOK, Response{
        Success: true,
        Message: response.Message,
    })
}

func (s *Server) getForumDetails(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    forumName := vars["forumName"]

    future := s.system.Root.RequestFuture(s.engine, &proto.GetForumDetails{
        ForumName: forumName,
    }, 5*time.Second)

    result, err := future.Result()
    if err != nil {
        sendError(w, http.StatusInternalServerError, "Failed to get forum details")
        return
    }

    response, ok := result.(*proto.ForumDetails)
    if !ok || !response.Success {
        sendError(w, http.StatusInternalServerError, response.Message)
        return
    }

    sendResponse(w, http.StatusOK, Response{
        Success: true,
        Message: response.Message,
        Data:    response,
    })
}

func (s *Server) createPost(w http.ResponseWriter, r *http.Request) {
    var req CreatePostRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    future := s.system.Root.RequestFuture(s.engine, &proto.CreateContent{
        UserHandle:        req.Username,
        Subreddit:        req.Subreddit,
        Heading:          req.Title,
        Body:             req.Content,
        IsShare:          req.IsRepost,
        OriginalContentId: req.OriginalId,
    }, 5*time.Second)

    result, err := future.Result()
    if err != nil {
        sendError(w, http.StatusInternalServerError, "Failed to create post")
        return
    }

    response, ok := result.(*proto.CreateContentResponse)
    if !ok || !response.Success {
        sendError(w, http.StatusInternalServerError, response.Message)
        return
    }

    sendResponse(w, http.StatusCreated, Response{
        Success: true,
        Message: response.Message,
        Data: map[string]string{
            "contentId": response.ContentId,
        },
    })
}

func (s *Server) getPost(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    postId := vars["postId"]

    future := s.system.Root.RequestFuture(s.engine, &proto.GetPost{
        ContentId: postId,
    }, 5*time.Second)

    result, err := future.Result()
    if err != nil {
        sendError(w, http.StatusInternalServerError, "Failed to get post")
        return
    }

    response, ok := result.(*proto.GetPostResponse)
    if !ok || !response.Success {
        sendError(w, http.StatusInternalServerError, response.Message)
        return
    }

    sendResponse(w, http.StatusOK, Response{
        Success: true,
        Message: response.Message,
        Data:    response.Content,
    })
}

func (s *Server) createComment(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    var req CreateCommentRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    future := s.system.Root.RequestFuture(s.engine, &proto.CreateFeedback{
        UserHandle: req.Username,
        ContentId:  vars["postId"],
        ParentId:   req.ParentId,
        Body:       req.Content,
    }, 5*time.Second)

    result, err := future.Result()
    if err != nil {
        sendError(w, http.StatusInternalServerError, "Failed to create comment")
        return
    }

    response, ok := result.(*proto.CreateFeedbackResponse)
    if !ok || !response.Success {
        sendError(w, http.StatusInternalServerError, response.Message)
        return
    }

    sendResponse(w, http.StatusCreated, Response{
        Success: true,
        Message: response.Message,
        Data: map[string]string{
            "feedbackId": response.FeedbackId,
        },
    })
}

func (s *Server) vote(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    var req VoteRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    future := s.system.Root.RequestFuture(s.engine, &proto.Reaction{
        UserHandle: req.Username,
        ItemId:     vars["postId"],
        IsPositive: req.IsUpvote,
        IsContent:  true,
    }, 5*time.Second)

    result, err := future.Result()
    if err != nil {
        sendError(w, http.StatusInternalServerError, "Failed to register vote")
        return
    }

    response, ok := result.(*proto.ReactionResponse)
    if !ok || !response.Success {
        sendError(w, http.StatusInternalServerError, response.Message)
        return
    }

    sendResponse(w, http.StatusOK, Response{
        Success: true,
        Message: response.Message,
    })
}

func (s *Server) getFeed(w http.ResponseWriter, r *http.Request) {
    username := r.URL.Query().Get("username")
    sortMethod := r.URL.Query().Get("sort")
    if sortMethod == "" {
        sortMethod = "hot"
    }

    future := s.system.Root.RequestFuture(s.engine, &proto.GetFeed{
        UserHandle: username,
        SortMethod: sortMethod,
        Limit:      50,
    }, 5*time.Second)

    result, err := future.Result()
    if err != nil {
        sendError(w, http.StatusInternalServerError, "Failed to get feed")
        return
    }

    response, ok := result.(*proto.FeedBundle)
    if !ok || !response.Success {
        sendError(w, http.StatusInternalServerError, response.Message)
        return
    }

    // Ensure ContentId is included for each post
    for _, content := range response.Contents {
        if content.ContentId == "" {
            content.ContentId = "Unknown" // Or generate a new ID if necessary
        }
    }

    sendResponse(w, http.StatusOK, Response{
        Success: true,
        Message: response.Message,
        Data:    response.Contents,
    })
}

func (s *Server) sendMessage(w http.ResponseWriter, r *http.Request) {
    var req SendMessageRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    future := s.system.Root.RequestFuture(s.engine, &proto.DirectChat{
        Sender:   req.SenderUsername,
        Receiver: req.ReceiverUsername,
        Content:  req.Content,
    }, 5*time.Second)

    result, err := future.Result()
    if err != nil {
        sendError(w, http.StatusInternalServerError, "Failed to send message")
        return
    }

    response, ok := result.(*proto.ChatResponse)
    if !ok || !response.Success {
        sendError(w, http.StatusInternalServerError, response.Message)
        return
    }

    sendResponse(w, http.StatusOK, Response{
        Success: true,
        Message: response.Message,
    })
}

func (s *Server) getMessages(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    username := vars["username"]

    future := s.system.Root.RequestFuture(s.engine, &proto.GetChats{
        UserHandle: username,
    }, 5*time.Second)

    result, err := future.Result()
    if err != nil {
        sendError(w, http.StatusInternalServerError, "Failed to get messages")
        return
    }

    response, ok := result.(*proto.ChatBundle)
    if !ok || !response.Success {
        sendError(w, http.StatusInternalServerError, response.Message)
        return
    }

    sendResponse(w, http.StatusOK, Response{
        Success: true,
        Message: response.Message,
        Data:    response.Messages,
    })
}

func (s *Server) Start(port int) error {
    addr := fmt.Sprintf(":%d", port)
    log.Printf("Starting REST server on %s", addr)
    return http.ListenAndServe(addr, s.router)
}

// Helper method to handle timeouts for futures
func (s *Server) waitForResponse(future *actor.Future, timeout time.Duration) (interface{}, error) {
    result, err := future.Result()
    if err != nil {
        return nil, fmt.Errorf("request failed: %v", err)
    }
    
    select {
    case <-time.After(timeout):
        return nil, fmt.Errorf("request timed out after %v", timeout)
    default:
        return result, nil
    }
}

// Health check endpoint
func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
    sendResponse(w, http.StatusOK, Response{
        Success: true,
        Message: "Service is healthy",
    })
}