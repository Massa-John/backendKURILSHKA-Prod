package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "strconv"
    "time"

    "github.com/gorilla/mux"
    _ "github.com/lib/pq"
    redis "github.com/redis/go-redis/v9"
)

type User struct {
    ID     int    `json:"id"`
    Name   string `json:"name"`
    Email  string `json:"email"`
    Avatar string `json:"avatar,omitempty"`
    Status string `json:"status"`
    Online bool   `json:"online"`
}

type Message struct {
    ID         int    `json:"id"`
    SenderID   int    `json:"senderId"`
    ReceiverID int    `json:"receiverId"`
    Text       string `json:"text"`
    Time       string `json:"time"`
}

type Contact struct {
    User
    LastMessage string `json:"lastMessage"`
}

var (
    dbConn      *sql.DB
    redisClient *redis.Client
)

func getEnv(key, fallback string) string {
    if value, ok := os.LookupEnv(key); ok && value != "" {
        return value
    }
    return fallback
}

func initDB() error {
    host := getEnv("POSTGRES_HOST", "localhost")
    port := getEnv("POSTGRES_PORT", "5432")
    user := getEnv("POSTGRES_USER", "chatuser")
    password := getEnv("POSTGRES_PASSWORD", "chatpass")
    dbname := getEnv("POSTGRES_DB", "chatdb")

    dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

    var err error
    dbConn, err = sql.Open("postgres", dsn)
    if err != nil {
        return err
    }

    dbConn.SetMaxOpenConns(25)
    dbConn.SetMaxIdleConns(5)
    dbConn.SetConnMaxLifetime(5 * time.Minute)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err = dbConn.PingContext(ctx); err != nil {
        return err
    }

    return nil
}

func initRedis() error {
    redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
    redisPassword := getEnv("REDIS_PASSWORD", "redispass")

    redisClient = redis.NewClient(&redis.Options{
        Addr:     redisAddr,
        Password: redisPassword,
        DB:       0,
    })

    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    if err := redisClient.Ping(ctx).Err(); err != nil {
        return err
    }

    return nil
}

func enableCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(payload)
}

func getUsers(w http.ResponseWriter, r *http.Request) {
    rows, err := dbConn.Query(`SELECT id, name, email, status, avatar FROM users ORDER BY id`)
    if err != nil {
        respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }
    defer rows.Close()

    users := []User{}
    for rows.Next() {
        var u User
        if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Status, &u.Avatar); err != nil {
            respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
            return
        }
        u.Online = (u.Status == "online")
        users = append(users, u)
    }

    respondJSON(w, http.StatusOK, users)
}

func getContacts(w http.ResponseWriter, r *http.Request) {
    cacheKey := "contacts"

    if val, err := redisClient.Get(context.Background(), cacheKey).Result(); err == nil {
        var contacts []Contact
        if err := json.Unmarshal([]byte(val), &contacts); err == nil {
            respondJSON(w, http.StatusOK, contacts)
            return
        }
    }

    rows, err := dbConn.Query(`
        SELECT u.id, u.name, u.email, u.status, u.avatar,
               COALESCE((SELECT text FROM messages m
                         WHERE m.sender_id = u.id OR m.receiver_id = u.id
                         ORDER BY sent_at DESC LIMIT 1), 'Нет сообщений')
        FROM users u
        ORDER BY u.id
    `)
    if err != nil {
        respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }
    defer rows.Close()

    contacts := []Contact{}
    for rows.Next() {
        var c Contact
        if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.Status, &c.Avatar, &c.LastMessage); err != nil {
            respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
            return
        }
        c.Online = (c.Status == "online")
        contacts = append(contacts, c)
    }

    payload, _ := json.Marshal(contacts)
    _ = redisClient.Set(context.Background(), cacheKey, payload, 5*time.Minute).Err()

    respondJSON(w, http.StatusOK, contacts)
}

func getMessages(w http.ResponseWriter, r *http.Request) {
    rows, err := dbConn.Query(`
        SELECT id, sender_id, receiver_id, text, sent_at
        FROM messages
        ORDER BY sent_at ASC
    `)
    if err != nil {
        respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }
    defer rows.Close()

    messages := []Message{}
    for rows.Next() {
        var m Message
        var sentAt time.Time
        if err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Text, &sentAt); err != nil {
            respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
            return
        }
        m.Time = sentAt.Format("15:04")
        messages = append(messages, m)
    }

    respondJSON(w, http.StatusOK, messages)
}

func createMessage(w http.ResponseWriter, r *http.Request) {
    var msg Message
    if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
        respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
        return
    }

    if msg.SenderID == 0 || msg.ReceiverID == 0 || msg.Text == "" {
        respondJSON(w, http.StatusBadRequest, map[string]string{"error": "senderId, receiverId and text are required"})
        return
    }

    query := `
        INSERT INTO messages (sender_id, receiver_id, text, sent_at)
        VALUES ($1, $2, $3, NOW())
        RETURNING id, sent_at
    `

    var id int
    var sentAt time.Time
    if err := dbConn.QueryRow(query, msg.SenderID, msg.ReceiverID, msg.Text).Scan(&id, &sentAt); err != nil {
        respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }

    msg.ID = id
    msg.Time = sentAt.Format("15:04")

    _ = redisClient.Del(context.Background(), "contacts").Err()

    respondJSON(w, http.StatusCreated, msg)
}

func health(w http.ResponseWriter, r *http.Request) {
    if err := dbConn.Ping(); err != nil {
        respondJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "db unavailable"})
        return
    }
    respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func main() {
    if err := initDB(); err != nil {
        log.Fatalf("failed to initialize DB: %v", err)
    }
    if err := initRedis(); err != nil {
        log.Fatalf("failed to initialize Redis: %v", err)
    }

    r := mux.NewRouter()
    r.Use(enableCORS)

    r.HandleFunc("/health", health).Methods("GET")
    r.HandleFunc("/api/users", getUsers).Methods("GET")
    r.HandleFunc("/api/contacts", getContacts).Methods("GET")
    r.HandleFunc("/api/messages", getMessages).Methods("GET")
    r.HandleFunc("/api/messages", createMessage).Methods("POST")

    port := getEnv("PORT", "8080")
    log.Printf("Server started on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, r))
}
