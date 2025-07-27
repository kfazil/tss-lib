package main

import (
	"sync"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
 "github.com/google/uuid"
)


// Send key share to device via WebSocket (final robust version)
func sendKeyShareToDevice(deviceID string, share interface{}) {
	wsDeviceConnsMutex.Lock()
	conn, ok := wsDeviceConns[deviceID]
	wsDeviceConnsMutex.Unlock()
	if !ok || conn == nil {
		fmt.Printf("[WS] No active WebSocket for device %s\n", deviceID)
		return
	}
	msg := g.Map{
		"type": "key_share",
		"device_id": deviceID,
		"key_share": share,
	}
	if err := conn.WriteJSON(msg); err != nil {
		fmt.Printf("[WS] Error sending key share to device %s: %v\n", deviceID, err)
	} else {
		fmt.Printf("[WS] Sent key share to device %s\n", deviceID)
	}
}

// WebSocket handler: expects device to send its device_id as first message
// func TSSWebSocketHandler(r *ghttp.Request) {
// 	ws, err := r.WebSocket()
// 	if err != nil || ws == nil {
// 		r.Response.WriteJson(g.Map{"error": "WebSocket upgrade failed"})
// 		return
// 	}
// 	// Wait for device to send its device_id as first message
// 	var deviceID string
// 	_, msg, err := ws.ReadMessage()
// 	if err != nil {
// 		fmt.Printf("[WS] Error reading device_id: %v\n", err)
// 		ws.Close()
// 		return
// 	}
// 	var m map[string]interface{}
// 	if err := gjson.DecodeTo(msg, &m); err == nil {
// 		if id, ok := m["device_id"].(string); ok {
// 			deviceID = id
// 		}
// 	}
// 	if deviceID == "" {
// 		fmt.Printf("[WS] No device_id provided on connect\n")
// 		ws.Close()
// 		return
// 	}
// 	// Register deviceID -> ws connection
// 	wsDeviceConnsMutex.Lock()
// 	wsDeviceConns[deviceID] = ws
// 	wsDeviceConnsMutex.Unlock()
// 	fmt.Printf("[WS] Registered device %s with WebSocket\n", deviceID)
// 	// Optionally: handle further messages or keep connection alive
// 	for {
// 		_, msg, err := ws.ReadMessage()
// 		if err != nil {
// 			fmt.Printf("[WS] Device %s disconnected: %v\n", deviceID, err)
// 			break
// 		}
// 		// Handle acknowledgements or other messages if needed
// 		fmt.Printf("[WS] Received from %s: %s\n", deviceID, string(msg))
// 	}
// 	// Cleanup on disconnect
// 	wsDeviceConnsMutex.Lock()
// 	delete(wsDeviceConns, deviceID)
// 	wsDeviceConnsMutex.Unlock()
// 	fmt.Printf("[WS] Device %s WebSocket removed\n", deviceID)
// }

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
		DeviceID  string `json:"device_id"`
		Threshold int    `json:"threshold"`
	}
	var req Req
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(g.Map{"error": "Invalid request"})
		return
	}
	// Debug log for deviceID
	fmt.Printf("[Pairing] StartPairingSession: user_id=%s, device_id=%s, threshold=%d\n", req.UserID, req.DeviceID, req.Threshold)
	code := uuid.NewString()[:6] // Short pairing code
	pairingMutex.Lock()
	pairingSessions[code] = &PairingSession{
		UserID:      req.UserID,
		DeviceIDs:   []string{req.DeviceID}, // include main device
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
		// Update session.DeviceIDs to match actual connected devices from wsDevices
		wsDevicesMutex.Lock()
		var connected []string
		for _, dev := range wsDevices {
			connected = append(connected, dev)
		}
		wsDevicesMutex.Unlock()
		fmt.Printf("[Pairing] Connected devices for keygen: %v\n", connected)
		session.DeviceIDs = connected
		fmt.Printf("[Pairing] CompletePairingAndKeygen: DeviceIDs=%v\n", session.DeviceIDs)
		if len(session.DeviceIDs) < session.Threshold {
			r.Response.WriteJson(g.Map{"error": "Not enough devices linked"})
			return
		}
		// Validate device IDs and log type/value
		for i, id := range session.DeviceIDs {
			fmt.Printf("[Pairing] DeviceID[%d]: %v (type: %T)\n", i, id, id)
			if id == "" {
				r.Response.WriteJson(g.Map{"error": "DeviceID at index " + fmt.Sprint(i) + " is empty"})
				return
			}
		}
		// Setup parties for TSS keygen
		partyIDs := make([]map[string]string, len(session.DeviceIDs))
		for i, id := range session.DeviceIDs {
			partyIDs[i] = map[string]string{"device_id": id, "party": string('A'+i)}
		}
		// Start TSS keygen for session.DeviceIDs
		keySaves, err := StartTSSKeygenSession(session.DeviceIDs, session.Threshold)
		if err != nil {
			r.Response.WriteJson(g.Map{"error": "Keygen failed: " + err.Error()})
			return
		}
		session.Active = false
		// Send key shares to devices via WebSocket
		for i, id := range session.DeviceIDs {
			share := keySaves[i]
			// sendKeyShareToDevice should send { key_share, device_id } to the device via WebSocket
			sendKeyShareToDevice(id, share)
		}
		// Only send main device's share in API response
		mainDeviceID := session.DeviceIDs[0]
		r.Response.WriteJson(g.Map{
			"message": "Keygen complete",
			"device_id": mainDeviceID,
			"key_share": keySaves[0],
			"devices": session.DeviceIDs,
			"parties": partyIDs,
		})
		return
	}
	r.Response.WriteJson(g.Map{"error": "Invalid or inactive pairing code"})
}
