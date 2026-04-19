package main

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
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

		var hash string

		err := db.QueryRow(
			"SELECT password FROM users WHERE username=?",
			u.Username,
		).Scan(&hash)

		if err != nil || !checkPass(hash, u.Password) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "user",
			Value:    u.Username,
			HttpOnly: true,
			Secure:   true,
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
