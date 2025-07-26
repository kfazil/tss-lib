package main

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gogf/gf/v2/net/ghttp"
)

var (
	wsDevices      = make(map[*ghttp.WebSocket]string) // ws -> deviceID
	wsDevicesMutex sync.Mutex
)

// TSSWebSocketHandler handles WebSocket connections for TSS sessions
func TSSWebSocketHandler(r *ghttp.Request) {
	fmt.Printf("[WS] Incoming request for WebSocket upgrade: %s %s\n", r.Method, r.RequestURI)
	ws, err := r.WebSocket()
	if err != nil {
		fmt.Println("[WS] Failed to upgrade connection:", err)
		r.Response.WriteJson(map[string]string{"error": "WebSocket required"})
		return
	}
	fmt.Println("[WS] New WebSocket connection from", r.RemoteAddr)

	// Expect device_id and pairing_code as first message
	_, msg, err := ws.ReadMessage()
	if err != nil {
		fmt.Println("[WS] Error reading initial message:", err)
		ws.Close()
		return
	}
	var init struct {
		DeviceID    string `json:"device_id"`
		PairingCode string `json:"pairing_code"`
	}
	json.Unmarshal(msg, &init)
	fmt.Printf("[WS] Device joined: device_id=%s pairing_code=%s\n", init.DeviceID, init.PairingCode)
	// Send initial status to frontend
	ws.WriteMessage(1, []byte(`{"status":"joined","device_id":"`+init.DeviceID+`"}`))

	wsDevicesMutex.Lock()
	wsDevices[ws] = init.DeviceID
	// Gather all connected devices for this pairing code
	var connected []string
	for _, dev := range wsDevices {
		connected = append(connected, dev)
	}
	fmt.Printf("[WS] Broadcasting connected devices: %v\n", connected)
	// Broadcast connected devices to all
	for w := range wsDevices {
		b, _ := json.Marshal(map[string]interface{}{
			"connected_devices": connected,
		})
		w.WriteMessage(1, b)
	}
	wsDevicesMutex.Unlock()

	// Echo loop for demo
	for {
		msgType, msg, err := ws.ReadMessage()
		if err != nil {
			fmt.Printf("[WS] Device disconnected: device_id=%s\n", wsDevices[ws])
			wsDevicesMutex.Lock()
			delete(wsDevices, ws)
			wsDevicesMutex.Unlock()
			break
		}
		fmt.Printf("[WS] Received message from device_id=%s: %s\n", wsDevices[ws], string(msg))
		ws.WriteMessage(msgType, msg)
	}
}
