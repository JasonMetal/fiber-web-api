// ------------------------------------------------------------------------
// ------------------------       统一的加密解密      ------------------------
// ------------------------------------------------------------------------

package utils

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fiber-web-api/internal/app/common/mylog"
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

// ====================================  MD5加密开始  ====================================

// md5加密
func MD5(v string) string {
	d := []byte(v)
	m := md5.New()
	m.Write(d)
	return hex.EncodeToString(m.Sum(nil))
}

// ====================================  RSA加解密开始  ====================================

var (
	PrivateKey *rsa.PrivateKey
	PublicKey  rsa.PublicKey
)

// 生成随机密钥对
func GenerateKeyPair() (*rsa.PrivateKey, *rsa.PublicKey) {
	var err any
	PrivateKey, err = rsa.GenerateKey(crand.Reader, 2048) //生成私钥
	if err != nil {
		panic(err)
	}
	PublicKey = PrivateKey.PublicKey //生成公钥
	return PrivateKey, &PublicKey
}

// 获取公钥
func GetPublicKey() string {
	publicKeyBytes := x509.MarshalPKCS1PublicKey(&PublicKey)
	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	return base64.StdEncoding.EncodeToString(pem.EncodeToMemory(pemBlock))
}

// RSA加密
func RSAEncrypt(str string) string {
	if str == "" {
		return str
	}
	//根据公钥加密
	encryptedBytes, err := rsa.EncryptOAEP(sha256.New(), crand.Reader, &PublicKey, []byte(str), nil)
	if err != nil {
		panic(err)
		return ""
	}
	// 加密后进行base64编码
	encryptBase64 := base64.StdEncoding.EncodeToString(encryptedBytes)
	return encryptBase64
}

// RSA解密
func RSADecrypt(str string) string {
	// base64解码
	decodedBase64, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		panic(err)
		mylog.Error(err.Error())
		return ""
	}
	//根据私钥解密
	decryptBytes, err := PrivateKey.Decrypt(nil, decodedBase64, &rsa.OAEPOptions{Hash: crypto.SHA256})
	if err != nil {
		panic(err)
		mylog.Error(err.Error())
		return ""
	}
	return string(decryptBytes)
}

// ====================================  盐值加密开始  ====================================

// 盐值加密（根据明文密码，获取密文）
func GetEncryptedPassword(password string) (string, error) {
	// 加密密码，使用 bcrypt 包当中的 GenerateFromPassword 方法，bcrypt.DefaultCost 代表使用默认加密成本
	encryptPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
		mylog.Error(err.Error())
		return "", err
	}
	return string(encryptPassword), nil
}

// 判断密码是否正确（根据明文和密文对比）plaintextPassword 明文 encryptedPassword 密文
func AuthenticatePassword(plaintextPassword string, encryptedPassword string) bool {
	// 使用 bcrypt 当中的 CompareHashAndPassword 对比密码是否正确，第一个参数为密文，第二个参数为明文
	err := bcrypt.CompareHashAndPassword([]byte(encryptedPassword), []byte(plaintextPassword))
	// 对比密码是否正确会返回一个异常，按照官方的说法是只要异常是 nil 就证明密码正确
	return err == nil
}

// ====================================  AES加解密开始  ====================================

// AES加密（要加密的内容、密钥、偏移量）
func AESEncrypt(content, key, iv string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", fmt.Errorf("创建AES加密块失败: %s", err)
	}
	blockSize := block.BlockSize()
	plaintext := []byte(content)
	plaintextLength := len(plaintext)
	if plaintextLength%blockSize != 0 {
		plaintextLength = plaintextLength + (blockSize - (plaintextLength % blockSize))
	}
	paddedPlaintext := make([]byte, plaintextLength)
	copy(paddedPlaintext, plaintext)
	ciphertext := make([]byte, len(paddedPlaintext))
	mode := cipher.NewCBCEncrypter(block, []byte(iv))
	mode.CryptBlocks(ciphertext, paddedPlaintext)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AES解密（要解密的内容、密钥、偏移量）
func AESDecrypt(content, key, iv string) (string, error) {
	encrypted, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return "", fmt.Errorf("base64解码失败: %s", err)
	}
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", fmt.Errorf("创建AES加密块失败: %s", err)
	}
	if len(encrypted) < aes.BlockSize {
		return "", fmt.Errorf("密文长度不足")
	}
	decrypted := make([]byte, len(encrypted))
	mode := cipher.NewCBCDecrypter(block, []byte(iv))
	mode.CryptBlocks(decrypted, encrypted)
	return string(bytes.TrimRight(decrypted, "\x00")), nil
}
