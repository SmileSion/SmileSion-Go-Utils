package rsa

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// RSAKeyPair 存储 RSA 公钥和私钥
type RSAKeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

// ======================= 生成 RSA 密钥对 =======================
func GenerateKey(bits int) (*RSAKeyPair, error) {
	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	return &RSAKeyPair{
		PrivateKey: priv,
		PublicKey:  &priv.PublicKey,
	}, nil
}

// ======================= PEM 编码 =======================
func PrivateKeyToPEM(priv *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})
}

func PublicKeyToPEM(pub *rsa.PublicKey) ([]byte, error) {
	b, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: b,
	}), nil
}

// ======================= PEM 解析 =======================
func PEMToPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("invalid private key PEM")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func PEMToPublicKey(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("invalid public key PEM")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not RSA public key")
	}
	return pub, nil
}

// ======================= 加密/解密 =======================

// 公钥加密
func Encrypt(pub *rsa.PublicKey, data []byte) ([]byte, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, pub, data)
}

// 私钥解密
func Decrypt(priv *rsa.PrivateKey, cipher []byte) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, priv, cipher)
}

// ======================= 签名/验签 =======================

// 私钥签名 (SHA256)
func Sign(priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	hash := sha256.Sum256(data)
	return rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hash[:])
}

// 公钥验签 (SHA256)
func Verify(pub *rsa.PublicKey, data, sig []byte) error {
	hash := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, hash[:], sig)
}