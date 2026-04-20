# Realtime-Messenger

Realtime chat server written in Go with WebSocket support, JWT authentication, rooms, and role system.

## Stack

- **Go** — backend
- **WebSocket** — gorilla/websocket
- **SQLite** — modernc.org/sqlite
- **JWT** — golang-jwt/jwt/v5
- **bcrypt** — golang.org/x/crypto

## Features

- Registration and login with bcrypt password hashing
- JWT authentication via HttpOnly cookie
- Rate limiting on login (5 attempts → 1 minute block)
- Chat rooms: `general`, `dev`, `gaming`, `sport`
- Role system: `user`, `mod`, `admin`
- Private messages (DM)
- Graceful shutdown

## Getting Started

### Requirements

- Go 1.21+

### Setup

```bash
git clone https://github.com/kojiro-sasaki/realtime-messenger
cd realtime-messenger
go mod tidy
```

### Environment

bash
`export JWT_SECRET=your_secret_key_here`

powershell
`$env:JWT_SECRET="your_secret_key_here"`

### Run

```bash
go run ./cmd/main.go
```

Server starts on `http://localhost:8080`

## Project Structure

```
realtime-messenger/
├── cmd/
│   └── main.go
├── internal/
│   ├── auth/
│   │   ├── auth.go       # register, login, logout, me handlers, rate limiter
│   │   └── jwt.go        # token generation and parsing
│   ├── chat/
│   │   ├── client.go     # client struct
│   │   ├── hub.go        # hub, commands, db worker
│   │   └── ws.go         # websocket handler
│   └── db/
│       └── db.go         # sqlite init, queries
└── static/
    ├── index.html
    ├── login.html
    └── register.html
```

## API

| Method | Endpoint | Description |
|---|---|---|
| GET | `/` | Chat page (requires auth) |
| GET | `/login` | Login page |
| POST | `/login` | Login, sets session cookie |
| GET | `/register` | Register page |
| POST | `/register` | Create account |
| GET | `/logout` | Clear session, redirect to login |
| GET | `/api/me` | Get current user info |
| WS | `/ws` | WebSocket connection |

## Chat Commands

| Command | Permission | Description |
|---|---|---|
| `/help` | everyone | list commands |
| `/users` | everyone | list online users |
| `/rooms` | everyone | list available rooms |
| `/rusers` | everyone | list users in current room |
| `/join <room>` | everyone | join a room |
| `/leave` | everyone | return to general |
| `/msg <user> <text>` | everyone | send private message |
| `/name <newname>` | everyone | change display name |
| `/whois <user>` | mod+ | show user info |
| `/kick <user>` | mod+ | kick user |
| `/role <user> <role>` | admin | change user role |

## Roles

- `user` — default role
- `mod` — can kick users, use `/whois`
- `admin` — full access, can change roles

To set the first admin manually:

```bash
sqlite3 chat.db "UPDATE users SET role = 'admin' WHERE username = 'yourname';"
```
