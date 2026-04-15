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

var users = make(map[string]string)

func hashPass(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func checkPass(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if u.Username == "" || u.Password == "" {
		http.Error(w, "empty fields", http.StatusBadRequest)
		return
	}
	if len(u.Password) < 5 {
		http.Error(w, "Password must be at least 6 symbols", http.StatusBadRequest)
		return
	}
	if _, exists := users[u.Username]; exists {
		http.Error(w, "User already exists", http.StatusBadRequest)
		return
	}
	hash, err := hashPass(u.Password)
	if err != nil {
		http.Error(w, "error hashing pass", http.StatusBadRequest)
		return
	}
	users[u.Username] = hash
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User created"))
}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	hash, exists := users[u.Username]
	if !exists || !checkPass(hash, u.Password) {
		http.Error(w, "invalid some", http.StatusUnauthorized)
		return
	}
	w.Write([]byte("login success"))
}
