# Reddit Clone

A Go-based Reddit clone implementing core Reddit functionality using REST API architecture and Actor Model for concurrent operations.

## Features
- User registration and management
- Forum creation and management
- Post/comment system with voting
- Direct messaging between users
- Real-time updates using Actor Model
- Thread-safe operations with mutex locks

## Tech Stack
- Go (Golang)
- Gorilla Mux (Router)
- Proto Actor (Concurrency)
- Protocol Buffers

## Quick Start
1. Start the server:
```bash
go run cmd/server/main.go -port 8080 -actor-port 8085
```

2. Start a client:
```bash
go run client/rest_client.go
```

## Available Commands
- `register <username>` - Register new user
- `create_forum <forum_name> <description>` - Create new forum
- `join_forum <username> <forum_name>` - Join existing forum
- `create_post <username> <forum> <title> <content>` - Create new post
- `comment <username> <postId> <parentId> <content>` - Add comment
- `vote <username> <postId> <upvote/downvote>` - Vote on post
- `get_feed <username> <sortMethod>` - Get content feed
- `send_message <from> <to> <content>` - Send direct message
- `get_messages <username>` - Get user messages

## Demo
Watch the demo video: [YouTube Demo](https://www.youtube.com/watch?v=RSbL_fuPvZ8&feature=youtu.be)

## Contributors
- Arpita Patnaik
- Bala Tripura Kumari Bodapati
