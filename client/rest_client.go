package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
     "github.com/mitchellh/mapstructure"
     "reddit/proto"
)

type RestClient struct {
	baseURL    string
	httpClient *http.Client
}

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func NewRestClient(baseURL string) *RestClient {
	return &RestClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *RestClient) RegisterUser(username string) error {
	payload := map[string]string{"username": username}
	return c.post("/api/users", payload)
}

func (c *RestClient) CreateForum(name, description string) error {
	payload := map[string]string{"name": name, "description": description}
	return c.post("/api/forums", payload)
}

func (c *RestClient) JoinForum(username, forumName string) error {
	payload := map[string]string{"username": username}
	return c.post(fmt.Sprintf("/api/forums/%s/join", forumName), payload)
}

func (c *RestClient) CreatePost(username, forum, title, content string, isRepost bool, originalId string) (string, error) {
    payload := map[string]interface{}{
        "username":   username,
        "subreddit":  forum,
        "title":      title,
        "content":    content,
        "isRepost":   isRepost,
        "originalId": originalId,
    }
    var response Response
    err := c.doRequest("POST", "/api/posts", payload, &response)
    if err != nil {
        return "", err
    }
    if !response.Success {
        return "", fmt.Errorf(response.Message)
    }
    data, ok := response.Data.(map[string]interface{})
    if !ok {
        return "", fmt.Errorf("unexpected response format")
    }
    contentId, ok := data["contentId"].(string)
    if !ok {
        return "", fmt.Errorf("content ID not found in response")
    }
    return contentId, nil
}

func (c *RestClient) CreateComment(username, postId, parentId, content string) error {
	payload := map[string]interface{}{
		"username": username,
		"content":  content,
		"parentId": parentId,
	}
	return c.post(fmt.Sprintf("/api/posts/%s/comments", postId), payload)
}

func (c *RestClient) Vote(username, postId string, isUpvote bool) error {
	payload := map[string]interface{}{"username": username, "isUpvote": isUpvote}
	return c.post(fmt.Sprintf("/api/posts/%s/vote", postId), payload)
}

func (c *RestClient) SendMessage(from, to, content string) error {
	payload := map[string]interface{}{
		"senderUsername":   from,
		"receiverUsername": to,
		"content":          content,
	}
	return c.post("/api/messages", payload)
}

func (c *RestClient) GetMessages(username string) ([]interface{}, error) {
	var response Response
	err := c.doRequest("GET", fmt.Sprintf("/api/messages/%s", username), nil, &response)
	if err != nil {
		return nil, err
	}
	if messages, ok := response.Data.([]interface{}); ok {
		return messages, nil
}
return nil, fmt.Errorf("invalid messages format in response")
}

func (c *RestClient) GetFeed(username, sortMethod string) ([]*proto.Content, error) {
    var response Response
    err := c.doRequest("GET", fmt.Sprintf("/api/feed?username=%s&sort=%s", username, sortMethod), nil, &response)
    if err != nil {
        return nil, err
    }
    
    if !response.Success {
        return nil, fmt.Errorf(response.Message)
    }
    
    contents, ok := response.Data.([]interface{})
    if !ok {
        return nil, fmt.Errorf("unexpected response format")
    }
    
    var feed []*proto.Content
    for _, item := range contents {
        contentMap, ok := item.(map[string]interface{})
        if !ok {
            return nil, fmt.Errorf("unexpected content format")
        }
        
        content := &proto.Content{}
        if err := mapstructure.Decode(contentMap, content); err != nil {
            return nil, fmt.Errorf("failed to decode content: %v", err)
        }
        feed = append(feed, content)
    }
    
    return feed, nil
}

func (c *RestClient) post(endpoint string, payload interface{}) error {
	var response Response
	return c.doRequest("POST", endpoint, payload, &response)
}

func (c *RestClient) doRequest(method, endpoint string, payload interface{}, response interface{}) error {
	var req *http.Request
	var err error
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %v", err)
		}
		req, err = http.NewRequest(method, c.baseURL+endpoint, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, c.baseURL+endpoint, nil)
	}
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(response)
}

func main() {
	client := NewRestClient("http://localhost:8080")
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Interactive Reddit Client")
	fmt.Println("Available Commands:")
	fmt.Println("  register <username>")
	fmt.Println("  create_forum <forum_name> <description>")
	fmt.Println("  join_forum <username> <forum_name>")
	fmt.Println("  create_post <username> <forum> <title> <content> [isRepost] [originalId]")
	fmt.Println("  comment <username> <postId> <parentId> <content>")
	fmt.Println("  vote <username> <postId> <upvote/downvote>")
	fmt.Println("  send_message <from> <to> <content>")
	fmt.Println("  get_messages <username>")
	fmt.Println("  get_feed <username> <sortMethod>")
	fmt.Println("  exit")

	for {
		fmt.Print("> ")
		command, _ := reader.ReadString('\n')
		command = strings.TrimSpace(command)
		args := strings.Split(command, " ")

		switch args[0] {
		case "register":
			if len(args) != 2 {
				fmt.Println("Usage: register <username>")
				continue
			}
			err := client.RegisterUser(args[1])
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("User registered successfully.")
			}
		case "create_forum":
			if len(args) < 3 {
				fmt.Println("Usage: create_forum <forum_name> <description>")
				continue
			}
			err := client.CreateForum(args[1], strings.Join(args[2:], " "))
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Forum created successfully.")
			}
		case "join_forum":
			if len(args) != 3 {
				fmt.Println("Usage: join_forum <username> <forum_name>")
				continue
			}
			err := client.JoinForum(args[1], args[2])
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Joined forum successfully.")
			}
		case "create_post":
            if len(args) < 5 {
                fmt.Println("Usage: create_post <username> <forum> <title> <content>")
                continue
            }
            username := args[1]
            forum := args[2]
            title := args[3]
            content := strings.Join(args[4:], " ")
            isRepost := false
            originalId := ""
            contentId, err := client.CreatePost(username, forum, title, content, isRepost, originalId)
            if err != nil {
                fmt.Println("Error:", err)
            } else {
                fmt.Printf("Post created successfully. Content ID: %s\n", contentId)
            }
		case "comment":
			if len(args) < 5 {
				fmt.Println("Usage: comment <username> <postId> <parentId> <content>")
				continue
			}
			err := client.CreateComment(args[1], args[2], args[3], args[4])
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Comment added successfully.")
			}
		case "vote":
			if len(args) != 4 {
				fmt.Println("Usage: vote <username> <postId> <upvote/downvote>")
				continue
			}
			isUpvote := args[3] == "upvote"
			err := client.Vote(args[1], args[2], isUpvote)
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Vote recorded successfully.")
			}
		case "send_message":
			if len(args) < 4 {
				fmt.Println("Usage: send_message <from> <to> <content>")
				continue
			}
			err := client.SendMessage(args[1], args[2], strings.Join(args[3:], " "))
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Message sent successfully.")
			}
		case "get_messages":
			if len(args) != 2 {
				fmt.Println("Usage: get_messages <username>")
				continue
			}
			messages, err := client.GetMessages(args[1])
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Printf("Messages: %v\n", messages)
			}
		case "get_feed":
            if len(args) != 3 {
                fmt.Println("Usage: get_feed <username> <sortMethod>")
                continue
            }
            feed, err := client.GetFeed(args[1], args[2])
            if err != nil {
                fmt.Println("Error:", err)
            } else {
                fmt.Println("Feed retrieved successfully:")
                for i, post := range feed {
                    fmt.Printf("Post #%d:\n", i+1)
                    fmt.Printf("  Creator: %s\n", post.Creator)
                    fmt.Printf("  Subreddit: %s\n", post.Subreddit)
                    fmt.Printf("  Heading: %s\n", post.Heading)
                    fmt.Printf("  Body: %s\n", post.Body)
                    fmt.Printf("  Points: %d\n", post.Points)
                    fmt.Printf("  Is Share: %v\n", post.IsShare)
                    fmt.Printf("  Feedback Count: %d\n", len(post.Feedback))
                    fmt.Printf("  Reactions: %v\n", post.Reactions)
                    fmt.Println("---")
                }
            }
		case "exit":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Unknown command:", args[0])
		}
	}
}