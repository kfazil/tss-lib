package main

import (
	"sync"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

// MessageStore holds encrypted messages
// Each message is encrypted with a threshold signature

type EncryptedMessage struct {
	ID           string
	UserID       string
	Ciphertext   string
	SessionID    string
}

var (
	messageStore      = make(map[string]*EncryptedMessage)
	messageStoreMutex sync.Mutex
)

// Accept a message, coordinate TSS signing, encrypt, and store
func StoreEncryptedMessage(r *ghttp.Request) {
	type Req struct {
		UserID    string `json:"user_id"`
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
	}
	var req Req
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(g.Map{"error": "Invalid request"})
		return
	}
	// Here, coordinate TSS signing using sessionID (stub for now)
	// In real flow, backend triggers TSS protocol, gets signature, uses it to encrypt
	ciphertext := "encrypted:" + req.Message // Replace with real encryption using signature
	id := uuid.NewString()
	messageStoreMutex.Lock()
	messageStore[id] = &EncryptedMessage{
		ID:         id,
		UserID:     req.UserID,
		Ciphertext: ciphertext,
		SessionID:  req.SessionID,
	}
	messageStoreMutex.Unlock()
	// Respond with stored message ID
	r.Response.WriteJson(g.Map{"message": "Message encrypted and stored", "id": id})
}

// Decrypt message (requires TSS session with 2 devices)
func DecryptStoredMessage(r *ghttp.Request) {
	type Req struct {
		UserID    string `json:"user_id"`
		MessageID string `json:"message_id"`
		SessionID string `json:"session_id"`
	}
	var req Req
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(g.Map{"error": "Invalid request"})
		return
	}
	messageStoreMutex.Lock()
	msg, ok := messageStore[req.MessageID]
	messageStoreMutex.Unlock()
	if !ok {
		r.Response.WriteJson(g.Map{"error": "Message not found"})
		return
	}
	// Here, coordinate TSS decryption using sessionID (stub for now)
	plaintext := "decrypted:" + msg.Ciphertext // Replace with real decryption using TSS
	r.Response.WriteJson(g.Map{"message": "Message decrypted", "plaintext": plaintext})
}
