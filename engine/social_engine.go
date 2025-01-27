// engine/social_engine.go
package engine

import (
    "log"
    "sort"
    "sync"
    "time"
    "github.com/asynkron/protoactor-go/actor"
    "reddit/proto"
    "reddit/utils"
)

type SocialEngine struct {
    users       map[string]*UserData
    forums      map[string]*ForumData
    contents    map[string]*proto.Content
    feedbacks   map[string]*proto.Feedback
    chats       map[string][]*proto.DirectChat
    mutex       sync.RWMutex
}

type UserData struct {
    Handle     string
    Points     int
    Forums     map[string]bool
    IsOnline   bool
    LastSeen   time.Time
}

type ForumData struct {
    Name       string
    Members    map[string]bool
    Contents   []*proto.Content
    Created    time.Time
}

func NewSocialEngine() *SocialEngine {
    return &SocialEngine{
        users:      make(map[string]*UserData),
        forums:     make(map[string]*ForumData),
        contents:   make(map[string]*proto.Content),
        feedbacks:  make(map[string]*proto.Feedback),
        chats:      make(map[string][]*proto.DirectChat),
    }
}

func (s *SocialEngine) Receive(context actor.Context) {
    switch msg := context.Message().(type) {
    case *actor.Started:
        log.Println("Social engine started")
    case *proto.OnboardUser:
        s.handleOnboarding(context, msg)
    case *proto.CreateForum:
        s.handleForumCreation(context, msg)
    case *proto.JoinForum:
        s.handleForumJoin(context, msg)
    case *proto.LeaveForum:
        s.handleForumLeave(context, msg)
    case *proto.CreateContent:
        s.handleContentCreation(context, msg)
    case *proto.CreateFeedback:
        s.handleFeedbackCreation(context, msg)
    case *proto.Reaction:
        s.handleReaction(context, msg)
    case *proto.GetFeed:
        s.handleFeedRequest(context, msg)
    case *proto.GetPost:
        s.handleGetPost(context, msg)
    case *proto.GetForumDetails:
        s.handleGetForumDetails(context, msg)
    case *proto.DirectChat:
        s.handleChatDelivery(context, msg)
    case *proto.GetChats:
        s.handleChatRetrieval(context, msg)
    case *proto.ActivityStatus:
        s.handleActivityUpdate(context, msg)
    }
}

func (s *SocialEngine) handleOnboarding(context actor.Context, msg *proto.OnboardUser) {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    if msg.UserHandle == "" {
        context.Respond(&proto.OnboardUserResponse{
            Success: false,
            Message: "Username cannot be empty",
        })
        return
    }

    if _, exists := s.users[msg.UserHandle]; exists {
        context.Respond(&proto.OnboardUserResponse{
            Success: false,
            Message: "Username already exists",
        })
        return
    }

    s.users[msg.UserHandle] = &UserData{
        Handle:    msg.UserHandle,
        Points:    0,
        Forums:    make(map[string]bool),
        IsOnline:  true,
        LastSeen:  time.Now(),
    }

    log.Printf("New user onboarded: %s", msg.UserHandle)
    context.Respond(&proto.OnboardUserResponse{
        Success: true,
        Message: "User registered successfully",
    })
}

func (s *SocialEngine) handleForumCreation(context actor.Context, msg *proto.CreateForum) {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    if msg.Name == "" {
        context.Respond(&proto.CreateForumResponse{
            Success: false,
            Message: "Forum name cannot be empty",
        })
        return
    }

    if _, exists := s.forums[msg.Name]; exists {
        context.Respond(&proto.CreateForumResponse{
            Success: false,
            Message: "Forum already exists",
        })
        return
    }

    s.forums[msg.Name] = &ForumData{
        Name:     msg.Name,
        Members:  make(map[string]bool),
        Contents: make([]*proto.Content, 0),
        Created:  time.Now(),
    }

    log.Printf("New forum created: %s", msg.Name)
    context.Respond(&proto.CreateForumResponse{
        Success: true,
        Message: "Forum created successfully",
    })
}

func (s *SocialEngine) handleForumJoin(context actor.Context, msg *proto.JoinForum) {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    user, userExists := s.users[msg.UserHandle]
    forum, forumExists := s.forums[msg.Subreddit]

    if !userExists {
        context.Respond(&proto.JoinForumResponse{
            Success: false,
            Message: "User not found",
        })
        return
    }

    if !forumExists {
        context.Respond(&proto.JoinForumResponse{
            Success: false,
            Message: "Forum not found",
        })
        return
    }

    if forum.Members[msg.UserHandle] {
        context.Respond(&proto.JoinForumResponse{
            Success: false,
            Message: "User already a member",
        })
        return
    }

    forum.Members[msg.UserHandle] = true
    user.Forums[msg.Subreddit] = true

    log.Printf("User %s joined forum %s", msg.UserHandle, msg.Subreddit)
    context.Respond(&proto.JoinForumResponse{
        Success: true,
        Message: "Joined forum successfully",
    })
}

func (s *SocialEngine) handleForumLeave(context actor.Context, msg *proto.LeaveForum) {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    user, userExists := s.users[msg.UserHandle]
    forum, forumExists := s.forums[msg.Subreddit]

    if !userExists || !forumExists {
        context.Respond(&proto.LeaveForumResponse{
            Success: false,
            Message: "User or forum not found",
        })
        return
    }

    if !forum.Members[msg.UserHandle] {
        context.Respond(&proto.LeaveForumResponse{
            Success: false,
            Message: "User is not a member",
        })
        return
    }

    delete(forum.Members, msg.UserHandle)
    delete(user.Forums, msg.Subreddit)

    log.Printf("User %s left forum %s", msg.UserHandle, msg.Subreddit)
    context.Respond(&proto.LeaveForumResponse{
        Success: true,
        Message: "Left forum successfully",
    })
}

func (s *SocialEngine) handleGetForumDetails(context actor.Context, msg *proto.GetForumDetails) {
    s.mutex.RLock()
    defer s.mutex.RUnlock()

    forum, exists := s.forums[msg.ForumName]
    if !exists {
        context.Respond(&proto.ForumDetails{
            Success: false,
            Message: "Forum not found",
        })
        return
    }

    response := &proto.ForumDetails{
        Success:      true,
        Name:         forum.Name,
        MemberCount:  int32(len(forum.Members)),
        Contents:     forum.Contents,
        Message:      "Forum details retrieved successfully",
    }

    context.Respond(response)
}

func (s *SocialEngine) handleContentCreation(context actor.Context, msg *proto.CreateContent) {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    if _, exists := s.users[msg.UserHandle]; !exists {
        context.Respond(&proto.CreateContentResponse{
            Success: false,
            Message: "User not found",
        })
        return
    }

    forum, exists := s.forums[msg.Subreddit]
    if !exists {
        context.Respond(&proto.CreateContentResponse{
            Success: false,
            Message: "Forum not found",
        })
        return
    }

    contentId := utils.GenerateID("cnt")
    content := &proto.Content{
        ContentId:         contentId,
        Creator:          msg.UserHandle,
        Subreddit:        msg.Subreddit,
        Heading:          msg.Heading,
        Body:             msg.Body,
        Timestamp:        time.Now().Unix(),
        Feedback:         make([]*proto.Feedback, 0),
        Reactions:        make(map[string]int32),
        Points:           0,
        IsShare:          msg.IsShare,
        OriginalContentId: msg.OriginalContentId,
    }

    s.contents[contentId] = content
    forum.Contents = append(forum.Contents, content)

    log.Printf("New content created in %s by %s", msg.Subreddit, msg.UserHandle)
    context.Respond(&proto.CreateContentResponse{
        Success:   true,
        Message:   "Content created successfully",
        ContentId: contentId,
    })
}

func (s *SocialEngine) handleFeedbackCreation(context actor.Context, msg *proto.CreateFeedback) {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    if _, exists := s.users[msg.UserHandle]; !exists {
        context.Respond(&proto.CreateFeedbackResponse{
            Success: false,
            Message: "User not found",
        })
        return
    }

    content, exists := s.contents[msg.ContentId]
    if !exists {
        context.Respond(&proto.CreateFeedbackResponse{
            Success: false,
            Message: "Content not found",
        })
        return
    }

    feedbackId := utils.GenerateID("fdb")
    feedback := &proto.Feedback{
        FeedbackId:  feedbackId,
        ContentId:   msg.ContentId,
        Creator:     msg.UserHandle,
        Body:        msg.Body,
        Timestamp:   time.Now().Unix(),
        ParentId:    msg.ParentId,
        Replies:     make([]*proto.Feedback, 0),
        Reactions:   make(map[string]int32),
        Points:      0,
    }

    s.feedbacks[feedbackId] = feedback

    if msg.ParentId == "" {
        content.Feedback = append(content.Feedback, feedback)
    } else {
        if parent, exists := s.feedbacks[msg.ParentId]; exists {
            parent.Replies = append(parent.Replies, feedback)
        } else {
            context.Respond(&proto.CreateFeedbackResponse{
                Success: false,
                Message: "Parent feedback not found",
            })
            return
        }
    }

    context.Respond(&proto.CreateFeedbackResponse{
        Success:    true,
        Message:    "Feedback created successfully",
        FeedbackId: feedbackId,
    })
}

func (s *SocialEngine) handleReaction(context actor.Context, msg *proto.Reaction) {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    if _, exists := s.users[msg.UserHandle]; !exists {
        context.Respond(&proto.ReactionResponse{
            Success: false,
            Message: "User not found",
        })
        return
    }

    value := int32(1)
    if !msg.IsPositive {
        value = -1
    }

    var success bool

    if msg.IsContent {
        if content, exists := s.contents[msg.ItemId]; exists {
            previousValue := content.Reactions[msg.UserHandle]
            content.Reactions[msg.UserHandle] = value
            content.Points += value - previousValue
            success = true
        }
    } else {
        if feedback, exists := s.feedbacks[msg.ItemId]; exists {
            previousValue := feedback.Reactions[msg.UserHandle]
            feedback.Reactions[msg.UserHandle] = value
            feedback.Points += value - previousValue
            success = true
        }
    }

    if !success {
        context.Respond(&proto.ReactionResponse{
            Success: false,
            Message: "Item not found",
        })
        return
    }

    context.Respond(&proto.ReactionResponse{
        Success: true,
        Message: "Reaction recorded successfully",
    })
}

func (s *SocialEngine) handleGetPost(context actor.Context, msg *proto.GetPost) {
    s.mutex.RLock()
    defer s.mutex.RUnlock()

    content, exists := s.contents[msg.ContentId]
    if !exists {
        context.Respond(&proto.GetPostResponse{
            Success: false,
            Message: "Post not found",
            Content: nil,
        })
        return
    }

    context.Respond(&proto.GetPostResponse{
        Success: true,
        Message: "Post retrieved successfully",
        Content: content,
    })
}

func (s *SocialEngine) handleFeedRequest(context actor.Context, msg *proto.GetFeed) {
    s.mutex.RLock()
    defer s.mutex.RUnlock()

    user, exists := s.users[msg.UserHandle]
    if !exists {
        context.Respond(&proto.FeedBundle{
            Success: false,
            Message: "User not found",
            Contents: nil,
        })
        return
    }

    var contents []*proto.Content
    for forumName := range user.Forums {
        if forum, exists := s.forums[forumName]; exists {
            contents = append(contents, forum.Contents...)
        }
    }

    switch msg.SortMethod {
    case "hot":
        sort.Slice(contents, func(i, j int) bool {
            // Calculate ups and downs from reactions
            iUps := 0
            iDowns := 0
            for _, value := range contents[i].Reactions {
                if value > 0 {
                    iUps++
                } else {
                    iDowns++
                }
            }
            
            jUps := 0
            jDowns := 0
            for _, value := range contents[j].Reactions {
                if value > 0 {
                    jUps++
                } else {
                    jDowns++
                }
            }
            
            scoreI := utils.CalculateHotScore(iUps, iDowns, contents[i].Timestamp)
            scoreJ := utils.CalculateHotScore(jUps, jDowns, contents[j].Timestamp)
            return scoreI > scoreJ
        })
    case "new":
        sort.Slice(contents, func(i, j int) bool {
            return contents[i].Timestamp > contents[j].Timestamp
        })
    case "top":
        sort.Slice(contents, func(i, j int) bool {
            return contents[i].Points > contents[j].Points
        })
    }

    if msg.Limit > 0 && len(contents) > int(msg.Limit) {
        contents = contents[:msg.Limit]
    }

    context.Respond(&proto.FeedBundle{
        Success: true,
        Message: "Feed retrieved successfully",
        Contents: contents,
    })
}

func (s *SocialEngine) handleChatDelivery(context actor.Context, msg *proto.DirectChat) {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    if _, exists := s.users[msg.Sender]; !exists {
        context.Respond(&proto.ChatResponse{
            Success: false,
            Message: "Sender not found",
        })
        return
    }

    if _, exists := s.users[msg.Receiver]; !exists {
        context.Respond(&proto.ChatResponse{
            Success: false,
            Message: "Receiver not found",
        })
        return
    }

    msg.MessageId = utils.GenerateID("msg")
    msg.Timestamp = time.Now().Unix()
    msg.Seen = false

    if _, exists := s.chats[msg.Receiver]; !exists {
        s.chats[msg.Receiver] = make([]*proto.DirectChat, 0)
    }
    s.chats[msg.Receiver] = append(s.chats[msg.Receiver], msg)

    log.Printf("Message delivered from %s to %s", msg.Sender, msg.Receiver)
    context.Respond(&proto.ChatResponse{
        Success: true,
        Message: "Message delivered successfully",
    })
}

func (s *SocialEngine) handleChatRetrieval(context actor.Context, msg *proto.GetChats) {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    if _, exists := s.users[msg.UserHandle]; !exists {
        context.Respond(&proto.ChatBundle{
            Success: false,
            Message: "User not found",
            Messages: nil,
        })
        return
    }

    messages := s.chats[msg.UserHandle]
    // Mark all messages as seen when retrieved
    for _, message := range messages {
        message.Seen = true
    }

    log.Printf("Retrieved %d messages for user %s", len(messages), msg.UserHandle)
    context.Respond(&proto.ChatBundle{
        Success: true,
        Message: "Messages retrieved successfully",
        Messages: messages,
    })
}

func (s *SocialEngine) handleActivityUpdate(context actor.Context, msg *proto.ActivityStatus) {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    user, exists := s.users[msg.UserHandle]
    if !exists {
        context.Respond(&proto.ActivityStatusResponse{
            Success: false,
            Message: "User not found",
        })
        return
    }

    user.IsOnline = msg.IsOnline
    user.LastSeen = time.Now()

    log.Printf("Updated activity status for user %s: online=%v", msg.UserHandle, msg.IsOnline)
    context.Respond(&proto.ActivityStatusResponse{
        Success: true,
        Message: "Activity status updated successfully",
    })
}

// Helper methods

func (s *SocialEngine) cleanup() {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    // Remove old messages (older than 30 days)
    thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Unix()
    for username, messages := range s.chats {
        filtered := make([]*proto.DirectChat, 0)
        for _, msg := range messages {
            if msg.Timestamp > thirtyDaysAgo {
                filtered = append(filtered, msg)
            }
        }
        s.chats[username] = filtered
    }

    // Mark users as offline if they haven't been seen in 5 minutes
    fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
    for _, user := range s.users {
        if user.IsOnline && user.LastSeen.Before(fiveMinutesAgo) {
            user.IsOnline = false
        }
    }
}

func (s *SocialEngine) getStats() map[string]interface{} {
    s.mutex.RLock()
    defer s.mutex.RUnlock()

    return map[string]interface{}{
        "total_users":    len(s.users),
        "total_forums":   len(s.forums),
        "total_posts":    len(s.contents),
        "total_comments": len(s.feedbacks),
        "online_users":   s.getOnlineUserCount(),
    }
}

func (s *SocialEngine) getOnlineUserCount() int {
    count := 0
    for _, user := range s.users {
        if user.IsOnline {
            count++
        }
    }
    return count
}