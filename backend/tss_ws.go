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
// Robust TSSWebSocketHandler: expects device_id, maps deviceID to ws, broadcasts connected devices
var (
	wsDeviceConns      = make(map[string]*ghttp.WebSocket) // deviceID -> ws
	wsDeviceConnsMutex sync.Mutex
)

func TSSWebSocketHandler(r *ghttp.Request) {
	fmt.Printf("[WS] Incoming request for WebSocket upgrade: %s %s\n", r.Method, r.RequestURI)
	ws, err := r.WebSocket()
	if err != nil || ws == nil {
		fmt.Println("[WS] Failed to upgrade connection:", err)
		r.Response.WriteJson(map[string]string{"error": "WebSocket required"})
		return
	}
	fmt.Println("[WS] New WebSocket connection from", r.RemoteAddr)

	// Expect device_id (and optionally pairing_code) as first message
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
	if init.DeviceID == "" {
		fmt.Println("[WS] No device_id provided on connect")
		ws.Close()
		return
	}
	// Now update wsDevices for compatibility with pairing.go
	wsDevicesMutex.Lock()
	wsDevices[ws] = init.DeviceID
	wsDevicesMutex.Unlock()
	fmt.Printf("[WS] Device joined: device_id=%s pairing_code=%s\n", init.DeviceID, init.PairingCode)
	ws.WriteMessage(1, []byte(`{"status":"joined","device_id":"`+init.DeviceID+`"}`))

	// Register deviceID -> ws connection
	wsDeviceConnsMutex.Lock()
	wsDeviceConns[init.DeviceID] = ws
	// Gather all connected devices
	var connected []string
	for _, dev := range wsDevices {
		connected = append(connected, dev)
	}
	fmt.Printf("[WS] Broadcasting connected devices: %v\n", connected)
	// Broadcast connected devices to all
	for _, w := range wsDeviceConns {
		b, _ := json.Marshal(map[string]interface{}{
			"connected_devices": connected,
		})
		w.WriteMessage(1, b)
	}
	wsDeviceConnsMutex.Unlock()

	// Main message loop
	for {
		msgType, msg, err := ws.ReadMessage()
		if err != nil {
			fmt.Printf("[WS] Device disconnected: device_id=%s\n", init.DeviceID)
			wsDeviceConnsMutex.Lock()
			delete(wsDeviceConns, init.DeviceID)
			wsDeviceConnsMutex.Unlock()
			wsDevicesMutex.Lock()
			delete(wsDevices, ws)
			wsDevicesMutex.Unlock()
			break
		}
		fmt.Printf("[WS] Received message from device_id=%s: %s\n", init.DeviceID, string(msg))
		// Optionally echo or handle key share/ack logic here
		ws.WriteMessage(msgType, msg)
	}
}
