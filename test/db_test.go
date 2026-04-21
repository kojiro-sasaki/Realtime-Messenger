package test

import (
	"database/sql"
	"fmt"
	"realtime-messenger/internal/chat"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestDBWorker_ShutdownDuringLoad(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")

	db.Exec(`CREATE TABLE messages (
        sender TEXT,
        text TEXT
    )`)

	h := chat.NewHub()
	h.StartDBWorkerTracked(db)

	total := 1000

	go func() {
		for i := 0; i < total; i++ {
			h.SaveMessage(chat.Message{
				Sender:  "user",
				Message: fmt.Sprintf(" %d", i),
			})
		}
	}()

	time.Sleep(50 * time.Millisecond)

	h.Shutdown()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count)

	if count == 0 {
		t.Fatal("no messages processed at all")
	}
}
