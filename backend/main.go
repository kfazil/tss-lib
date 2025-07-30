package main

import (
	   "fmt"
	   "log"
	   "github.com/gogf/gf/v2/frame/g"
	   "github.com/gogf/gf/v2/net/ghttp"
	   "github.com/joho/godotenv"
// ...existing code...
)

func main() {
	   // Load .env file for configs
	   if err := godotenv.Load(".env"); err != nil {
			   log.Printf("No .env file found or error loading: %v", err)
	   }
	   // Initialize persistent storage
	   if err := InitDB(); err != nil {
			   panic("Failed to initialize DB: " + err.Error())
	   }

	s := g.Server()
	s.SetPort(40715)
	// Global middleware to log every HTTP request
	s.Use(func(r *ghttp.Request) {
		// CORS headers
		r.Response.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		r.Response.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		r.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		r.Response.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == "OPTIONS" {
			r.Response.WriteStatus(200)
			return
		}
		fmt.Printf("[HTTP] %s %s from %s\n", r.Method, r.RequestURI, r.RemoteAddr)
		// for k, v := range r.Header {
		// 	fmt.Printf("[HTTP] Header: %s=%v\n", k, v)
		// }
		r.Middleware.Next()
	})

	   s.Group("/api", func(group *ghttp.RouterGroup) {
			   // Public endpoints
			   group.GET("/health", HealthCheck)
			   group.POST("/register", RegisterUser)
			   group.POST("/login", LoginUser)
			   group.POST("/pairing/start", StartPairingSession)
			   group.POST("/pairing/link", LinkDeviceWithPairingCode)
			   group.POST("/pairing/complete", CompletePairingAndKeygen)
			   group.GET("/tss/ws", TSSWebSocketHandler)

			   // Signing session endpoints
			   group.POST("/signing/start", StartSigningSession)
			   group.POST("/signing/join", JoinSigningSession)
			   group.GET("/signing/info", GetSigningSessionInfo)

			   // Message encryption/decryption endpoints
		  group.POST("/message/store", StoreEncryptedMessage) // now uses TSS signing
		  group.POST("/message/decrypt", DecryptStoredMessage)

			   // Protected endpoints
			   group.Group("/", func(protected *ghttp.RouterGroup) {
					   protected.Middleware(AuthMiddleware)
					   protected.POST("/device/register", RegisterDevice)
					   protected.POST("/device/threshold", SetupThreshold)
					   protected.POST("/encrypt", EncryptData)
					   protected.POST("/decrypt", DecryptData)
			   })
	   })

	s.Run()
}

// Stubs are now in stubs.go
