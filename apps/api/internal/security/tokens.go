package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// RandomHex 生成 n 字节的随机十六进制字符串（会话令牌 / OAuth state）。
func RandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// HashToken 对明文令牌做 SHA-256，数据库只存摘要。
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
