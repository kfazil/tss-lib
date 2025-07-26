package main

import (
	"sync"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type User struct {
	ID        string   `json:"id"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Devices   []string `json:"devices"`
	Threshold int      `json:"threshold"`
}

var (
	userStore = make(map[string]*User)
	userMutex sync.Mutex
)

func RegisterUser(r *ghttp.Request) {
	type Req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var req Req
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(g.Map{"error": "Invalid request"})
		return
	}
	userMutex.Lock()
	defer userMutex.Unlock()
	for _, u := range userStore {
		if u.Username == req.Username {
			r.Response.WriteJson(g.Map{"error": "Username already exists"})
			return
		}
	}
	id := uuid.NewString()
	userStore[id] = &User{
		ID:        id,
		Username:  req.Username,
		Password:  req.Password, // In production, hash this!
		Devices:   []string{},
		Threshold: 0,
	}
	r.Response.WriteJson(g.Map{"message": "User registered", "id": id})
}

// Login handler for existing users
func LoginUser(r *ghttp.Request) {
	type Req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var req Req
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(g.Map{"error": "Invalid request"})
		return
	}
	userMutex.Lock()
	defer userMutex.Unlock()
	for _, u := range userStore {
		if u.Username == req.Username && u.Password == req.Password {
			// Generate JWT token
			token, _ := GenerateJWT(u.ID)
			r.Response.WriteJson(g.Map{"message": "Login successful", "id": u.ID, "token": token})
			return
		}
	}
	r.Response.WriteJson(g.Map{"error": "Invalid username or password"})
}
