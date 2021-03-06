package sign

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"github.com/pjoc-team/tracing/logger"
)

// PKCS8 sign by pkcs8 key
func PKCS8(src []byte, privateKey string, hash crypto.Hash) ([]byte, error) {
	var h = hash.New()
	var err error
	_, err = h.Write(src)
	if err != nil {
		return nil, err
	}

	var hashed = h.Sum(nil)

	bytes, _ := base64.StdEncoding.DecodeString(privateKey)
	pri, err := x509.ParsePKCS8PrivateKey(bytes)
	if err != nil {
		logger.Log().Errorf("Parse private key with error: %v", err.Error())
		return nil, err
	}
	// rsa.Sign
	return rsa.SignPKCS1v15(rand.Reader, pri.(*rsa.PrivateKey), hash, hashed)
}

// PKCS1v15WithStringKey sign by pkcs1v15 key
func PKCS1v15WithStringKey(src []byte, privateKeyString string, hash crypto.Hash) ([]byte, error) {
	privateKey := ParsePrivateKey(privateKeyString)
	return PKCS1v15(src, privateKey, hash)
}

// VerifyPKCS1v15WithStringKey verify by pkcs1v15 key
func VerifyPKCS1v15WithStringKey(src, sig []byte, publicKeyString string, hash crypto.Hash) error {
	publicKey := ParsePublicKey(publicKeyString)
	return VerifyPKCS1v15(src, sig, publicKey, hash)
}

// PKCS1v15 sign by pkcs1v15 key
func PKCS1v15(src, privateKey []byte, hash crypto.Hash) ([]byte, error) {
	var h = hash.New()
	if _, err2 := h.Write(src); err2 != nil {
		return nil, err2
	}

	var hashed = h.Sum(nil)

	var err error
	var block *pem.Block
	block, _ = pem.Decode(privateKey)
	if block == nil {
		return nil, errors.New("private key error")
	}

	var pri *rsa.PrivateKey
	pri, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.SignPKCS1v15(rand.Reader, pri, hash, hashed)
}

// VerifyPKCS1v15 verify by pkcs1v15 key
func VerifyPKCS1v15(src, sig, publicKey []byte, hash crypto.Hash) error {
	var h = hash.New()
	if _, err2 := h.Write(src); err2 != nil {
		return err2
	}
	var hashed = h.Sum(nil)

	var err error
	var block *pem.Block
	block, _ = pem.Decode(publicKey)
	if block == nil {
		return errors.New("public key error")
	}

	var pubInterface interface{}
	pubInterface, err = x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}
	var pub = pubInterface.(*rsa.PublicKey)

	return rsa.VerifyPKCS1v15(pub, hash, hashed, sig)
}
