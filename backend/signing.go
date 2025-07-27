package main

import (
	"sync"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

// SigningSession holds info about a threshold signing session
// Devices join with sessionID, backend coordinates TSS protocol

type SigningSession struct {
	SessionID   string
	Message     string // Message to sign
	DeviceIDs   []string
	Active      bool
}

var (
	signingSessions      = make(map[string]*SigningSession)
	signingSessionsMutex sync.Mutex
)

// Start a new signing session (Device A)
func StartSigningSession(r *ghttp.Request) {
	type Req struct {
		DeviceID string `json:"device_id"`
		Message  string `json:"message"`
	}
	var req Req
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(g.Map{"error": "Invalid request"})
		return
	}
	id := uuid.NewString()
	signingSessionsMutex.Lock()
	signingSessions[id] = &SigningSession{
		SessionID: id,
		Message:   req.Message,
		DeviceIDs: []string{req.DeviceID},
		Active:    true,
	}
	signingSessionsMutex.Unlock()
	r.Response.WriteJson(g.Map{"message": "Signing session started", "session_id": id})
}

// Device B joins signing session
func JoinSigningSession(r *ghttp.Request) {
	type Req struct {
		DeviceID  string `json:"device_id"`
		SessionID string `json:"session_id"`
	}
	var req Req
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(g.Map{"error": "Invalid request"})
		return
	}
	signingSessionsMutex.Lock()
	session, ok := signingSessions[req.SessionID]
	if !ok || !session.Active {
		signingSessionsMutex.Unlock()
		r.Response.WriteJson(g.Map{"error": "Session not found or inactive"})
		return
	}
	session.DeviceIDs = append(session.DeviceIDs, req.DeviceID)
	signingSessionsMutex.Unlock()
	r.Response.WriteJson(g.Map{"message": "Device joined signing session", "session_id": req.SessionID})
}

// Get session info (for debugging/UI)
func GetSigningSessionInfo(r *ghttp.Request) {
	sessionID := r.Get("session_id")
	signingSessionsMutex.Lock()
	session, ok := signingSessions[sessionID]
	signingSessionsMutex.Unlock()
	if !ok {
		r.Response.WriteJson(g.Map{"error": "Session not found"})
		return
	}
	r.Response.WriteJson(g.Map{"session": session})
}
