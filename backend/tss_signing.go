package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// MessageStore holds encrypted messages
// Each message is encrypted with a threshold signature

type EncryptedMessage struct {
	ID           string
	UserID       string
	Ciphertext   []byte
	SessionID    string
}

var (
	messageStore      = make(map[string]*EncryptedMessage)
	messageStoreMutex sync.Mutex
)

// Helper: Load key share from file (replace with DB in production)
func loadKeyShare(deviceID string) (*keygen.LocalPartySaveData, error) {
	filename := fmt.Sprintf("keyshare_%s.json", deviceID)
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var key keygen.LocalPartySaveData
	if err := json.Unmarshal(data, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

// Encrypt message using signature-derived key
func encryptWithSignatureKey(message string, sig *common.SignatureData) ([]byte, error) {
	key := sha256.Sum256(append(sig.R, sig.S...))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	plaintext := []byte(message)
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	copy(iv, key[:aes.BlockSize])
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	return ciphertext, nil
}

// Store encrypted message after TSS signing
func StoreEncryptedMessage(r *ghttp.Request) {
	   type Req struct {
			   UserID    string `json:"user_id"`
			   Message   string `json:"message"`
			   SessionID string `json:"session_id"`
			   DeviceIDs []string `json:"device_ids"`
			   KeyShares []json.RawMessage `json:"key_shares"`
	   }
	   var req Req
	   if err := r.Parse(&req); err != nil {
			   r.Response.WriteJson(g.Map{"error": "Invalid request"})
			   return
	   }
	   if len(req.DeviceIDs) != 2 || len(req.KeyShares) != 2 {
			   r.Response.WriteJson(g.Map{"error": "Exactly 2 device IDs and 2 key shares required for signing"})
			   return
	   }
	   // Parse key shares from JSON
	   var key1, key2 keygen.LocalPartySaveData
	   if err := json.Unmarshal(req.KeyShares[0], &key1); err != nil {
			   r.Response.WriteJson(g.Map{"error": "Failed to parse key share for device 1"})
			   return
	   }
	   if err := json.Unmarshal(req.KeyShares[1], &key2); err != nil {
			   r.Response.WriteJson(g.Map{"error": "Failed to parse key share for device 2"})
			   return
	   }
	// Create TSS parties
	party1ID := tss.NewPartyID("P1", req.DeviceIDs[0], key1.Ks[0])
	party2ID := tss.NewPartyID("P2", req.DeviceIDs[1], key2.Ks[0])
	partyIDs := tss.SortPartyIDs([]*tss.PartyID{party1ID, party2ID})
	peerCtx := tss.NewPeerContext(partyIDs)
	params := []*tss.Parameters{
		tss.NewParameters(tss.S256(), peerCtx, party1ID, 2, 1),
		tss.NewParameters(tss.S256(), peerCtx, party2ID, 2, 1),
	}
	msg := []byte(req.Message)
	msgInt := new(big.Int).SetBytes(msg)
	outs := []chan tss.Message{make(chan tss.Message, 100), make(chan tss.Message, 100)}
	ends := []chan *common.SignatureData{make(chan *common.SignatureData, 1), make(chan *common.SignatureData, 1)}
	// Start parties
	party1 := signing.NewLocalParty(msgInt, params[0], *key1, outs[0], ends[0])
	party2 := signing.NewLocalParty(msgInt, params[1], *key2, outs[1], ends[1])
	go party1.Start()
	go party2.Start()
	// Orchestrate protocol
	var sig *common.SignatureData
	for sig == nil {
		select {
		case m := <-outs[0]:
			party2.UpdateFromBytes(m.WireBytesNoSig(), party1ID, true)
		case m := <-outs[1]:
			party1.UpdateFromBytes(m.WireBytesNoSig(), party2ID, true)
		case s := <-ends[0]:
			sig = s
		case s := <-ends[1]:
			sig = s
		}
	}
	// Encrypt message
	ciphertext, err := encryptWithSignatureKey(req.Message, sig)
	if err != nil {
		r.Response.WriteJson(g.Map{"error": "Encryption failed"})
		return
	}
	id := req.SessionID
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
