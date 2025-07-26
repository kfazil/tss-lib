package main

import (
	"sync"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type PairingSession struct {
	UserID      string
	DeviceIDs   []string
	PairingCode string
	Threshold   int
	Active      bool
}

var (
	pairingSessions = make(map[string]*PairingSession)
	pairingMutex    sync.Mutex
)

// Start pairing session after user registration
func StartPairingSession(r *ghttp.Request) {
	type Req struct {
		UserID    string `json:"user_id"`
		Threshold int    `json:"threshold"`
	}
	var req Req
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(g.Map{"error": "Invalid request"})
		return
	}
	code := uuid.NewString()[:6] // Short pairing code
	pairingMutex.Lock()
	pairingSessions[code] = &PairingSession{
		UserID:      req.UserID,
		DeviceIDs:   []string{},
		PairingCode: code,
		Threshold:   req.Threshold,
		Active:      true,
	}
	pairingMutex.Unlock()
	r.Response.WriteJson(g.Map{"pairing_code": code})
}

// Link device using pairing code
func LinkDeviceWithPairingCode(r *ghttp.Request) {
	type Req struct {
		PairingCode string `json:"pairing_code"`
		DeviceID    string `json:"device_id"`
	}
	var req Req
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(g.Map{"error": "Invalid request"})
		return
	}
	pairingMutex.Lock()
	defer pairingMutex.Unlock()
	if session, ok := pairingSessions[req.PairingCode]; ok && session.Active {
		session.DeviceIDs = append(session.DeviceIDs, req.DeviceID)
		r.Response.WriteJson(g.Map{"message": "Device linked"})
		return
	}
	r.Response.WriteJson(g.Map{"error": "Invalid or inactive pairing code"})
}

// Complete pairing and start TSS keygen
func CompletePairingAndKeygen(r *ghttp.Request) {
	type Req struct {
		PairingCode string `json:"pairing_code"`
	}
	var req Req
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(g.Map{"error": "Invalid request"})
		return
	}
	pairingMutex.Lock()
	defer pairingMutex.Unlock()
	if session, ok := pairingSessions[req.PairingCode]; ok && session.Active {
		if len(session.DeviceIDs) < session.Threshold {
			r.Response.WriteJson(g.Map{"error": "Not enough devices linked"})
			return
		}
		// Start TSS keygen for session.DeviceIDs
		keySaves, err := StartTSSKeygenSession(session.DeviceIDs, session.Threshold)
		if err != nil {
			r.Response.WriteJson(g.Map{"error": "Keygen failed"})
			return
		}
		session.Active = false
		// In production, securely deliver keySaves[i] to deviceIDs[i]
		r.Response.WriteJson(g.Map{"message": "Keygen complete", "devices": session.DeviceIDs, "key_shares": keySaves})
		return
	}
	r.Response.WriteJson(g.Map{"error": "Invalid or inactive pairing code"})
}
