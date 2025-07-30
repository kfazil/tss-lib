package main

import (
	   "sync"
	   "github.com/gogf/gf/v2/frame/g"
	   "github.com/gogf/gf/v2/net/ghttp"
	   "github.com/google/uuid"
)

// User type is now in types.go
// Remove duplicate type definition

var (
	   userStore = make(map[string]*User)
	   userMutex sync.Mutex
)

func RegisterUser(r *ghttp.Request) {
	   type Req struct {
			   Username string `json:"username"`
			   Password string `json:"password"`
			   DeviceName string `json:"device_name"`
	   }
	   var req Req
	   if err := r.Parse(&req); err != nil {
			   r.Response.WriteJson(g.Map{"error": "Invalid request"})
			   return
	   }
	   // Check if user already exists in DB
	   var count int64
	   if err := DB.Model(&User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
			   r.Response.WriteJson(g.Map{"error": "DB error"})
			   return
	   }
	   if count > 0 {
			   r.Response.WriteJson(g.Map{"error": "Username already exists"})
			   return
	   }
	   id := uuid.NewString()
	   deviceID := uuid.NewString()
	   deviceName := req.DeviceName
	   if deviceName == "" {
			   deviceName = "Main Device"
	   }
	   mainDevice := Device{
			   ID: deviceID,
			   UserID: id,
			   Name: deviceName,
	   }
	   newUser := &User{
			   ID:        id,
			   Username:  req.Username,
			   Password:  req.Password, // In production, hash this!
			   Devices:   []Device{},
			   Threshold: 2,
	   }
	   // Create user and main device in DB (use transaction)
	   tx := DB.Begin()
	   if err := tx.Create(newUser).Error; err != nil {
			   tx.Rollback()
			   r.Response.WriteJson(g.Map{"error": "Failed to register user in DB"})
			   return
	   }
	   if err := tx.Create(&mainDevice).Error; err != nil {
			   tx.Rollback()
			   r.Response.WriteJson(g.Map{"error": "Failed to register main device in DB"})
			   return
	   }
	   tx.Commit()
	   r.Response.WriteJson(g.Map{"message": "User registered", "id": id, "device_id": deviceID})
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
