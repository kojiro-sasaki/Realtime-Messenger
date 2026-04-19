package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginAttempt struct {
	count     int
	blockedTo time.Time
}

type checkRequest struct {
	ip       string
	username string
	resp     chan bool
	success  bool
}

var loginChan = make(chan checkRequest, 100)

func startLoginLimiter() {
	attempts := make(map[string]*loginAttempt)

	for req := range loginChan {
		key := req.ip + ":" + req.username

		att := attempts[key]
		now := time.Now()

		if att != nil && now.Before(att.blockedTo) {
			req.resp <- false
			continue
		}

		if req.success {
			delete(attempts, key)
			req.resp <- true
			continue
		}

		if att == nil {
			att = &loginAttempt{}
			attempts[key] = att
		}

		att.count++

		if att.count >= 5 {
			att.blockedTo = now.Add(1 * time.Minute)
			att.count = 0
		}

		req.resp <- true
	}
}

func hashPass(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func checkPass(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func registerHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {
		http.ServeFile(w, r, "./static/register.html")
		return
	}

	if r.Method == http.MethodPost {
		var u User

		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		if u.Username == "" || u.Password == "" {
			http.Error(w, "empty fields", http.StatusBadRequest)
			return
		}

		if len(u.Password) < 6 {
			http.Error(w, "password must be at least 6 characters", http.StatusBadRequest)
			return
		}

		hash, err := hashPass(u.Password)
		if err != nil {
			http.Error(w, "error hashing password", http.StatusInternalServerError)
			return
		}

		_, err = db.Exec(
			"INSERT INTO users (username, password) VALUES (?, ?)",
			u.Username,
			hash,
		)
		if err != nil {
			http.Error(w, "username already exists", http.StatusBadRequest)
			return
		}

		w.Write([]byte("user created"))
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {
		http.ServeFile(w, r, "./static/login.html")
		return
	}

	if r.Method == http.MethodPost {
		var u User

		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		ip := strings.Split(r.RemoteAddr, ":")[0]

		resp := make(chan bool)
		loginChan <- checkRequest{
			ip:       ip,
			username: u.Username,
			resp:     resp,
			success:  false,
		}

		if !<-resp {
			http.Error(w, "Too many attempts. Try later.", 429)
			return
		}
		var hash string
		err := db.QueryRow(
			"SELECT password FROM users WHERE username=?",
			u.Username,
		).Scan(&hash)

		if err != nil || !checkPass(hash, u.Password) {

			resp := make(chan bool)
			loginChan <- checkRequest{
				ip:       ip,
				username: u.Username,
				resp:     resp,
				success:  false,
			}
			<-resp

			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		resp = make(chan bool)
		loginChan <- checkRequest{
			ip:       ip,
			username: u.Username,
			resp:     resp,
			success:  true,
		}
		<-resp

		http.SetCookie(w, &http.Cookie{
			Name:     "user",
			Value:    u.Username,
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteStrictMode,
			Path:     "/",
		})

		w.Write([]byte("login success"))
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func isAuthenticated(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("user")
	if err != nil {
		return "", false
	}
	return cookie.Value, true
}
