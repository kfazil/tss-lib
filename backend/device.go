package main

import (
	"sync"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Device struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	Name   string `json:"name"`
}

var (
	deviceStore = make(map[string]*Device)
	deviceMutex sync.Mutex
)

func RegisterDevice(r *ghttp.Request) {
	type Req struct {
		UserID string `json:"user_id"`
		Name   string `json:"name"`
	}
	var req Req
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(g.Map{"error": "Invalid request"})
		return
	}
	deviceMutex.Lock()
	defer deviceMutex.Unlock()
	id := uuid.NewString()
	deviceStore[id] = &Device{
		ID:     id,
		UserID: req.UserID,
		Name:   req.Name,
	}
	// Add device to user
	userMutex.Lock()
	if user, ok := userStore[req.UserID]; ok {
		user.Devices = append(user.Devices, id)
	}
	userMutex.Unlock()
	r.Response.WriteJson(g.Map{"message": "Device registered", "id": id})
}

func SetupThreshold(r *ghttp.Request) {
	type Req struct {
		UserID    string `json:"user_id"`
		Threshold int    `json:"threshold"`
	}
	var req Req
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(g.Map{"error": "Invalid request"})
		return
	}
	userMutex.Lock()
	defer userMutex.Unlock()
	if user, ok := userStore[req.UserID]; ok {
		user.Threshold = req.Threshold
		r.Response.WriteJson(g.Map{"message": "Threshold updated", "threshold": req.Threshold})
		return
	}
	r.Response.WriteJson(g.Map{"error": "User not found"})
}
