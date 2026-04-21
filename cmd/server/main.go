package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"realtime-messenger/internal/auth"
	"realtime-messenger/internal/chat"
	"realtime-messenger/internal/db"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	db.InitDB()
	db.CreateTables()

	if err := auth.InitSecret(); err != nil {
		log.Fatal(err)
	}

	go auth.StartLoginLimiter()
	hub := chat.NewHub()
	go hub.Run()

	workers := 4
	for i := 0; i < workers; i++ {
		hub.StartDBWorkerTracked(db.DB)
	}

	http.HandleFunc("/ws", chat.WsHandler(hub))
	http.HandleFunc("/register", auth.RegisterHandler)
	http.HandleFunc("/login", auth.LoginHandler)
	http.HandleFunc("/api/me", auth.MeHandler)
	http.HandleFunc("/logout", auth.LogoutHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, ok := auth.IsAuthenticated(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		http.ServeFile(w, r, "./static/index.html")
	})

	http.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("./static")),
		),
	)

	server := &http.Server{
		Addr: ":8080",
	}

	go func() {
		log.Println("server started on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println("server error:", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("shutting down...")

	hub.CloseClients()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Println("server shutdown error:", err)
	}
	hub.Shutdown()

	log.Println("server stopped")
}
