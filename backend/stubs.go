package main

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// EncryptData stub
func EncryptData(r *ghttp.Request) {
	r.Response.WriteJson(g.Map{"message": "Data encrypted (stub)"})
}

// DecryptData stub
func DecryptData(r *ghttp.Request) {
	r.Response.WriteJson(g.Map{"message": "Data decrypted (stub)"})
}

// HealthCheck endpoint
func HealthCheck(r *ghttp.Request) {
	r.Response.WriteJson(map[string]string{"status": "ok"})
}
