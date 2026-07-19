package legacycrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
)

// javaAPIKeyAESKey 与 Java AESCryptUtils 保持一致，仅用于读取迁移期遗留的 api_key 数据。
// 新建 API key 应迁移到不可逆 hash 存储，不能继续扩散该历史加密方案。
const javaAPIKeyAESKey = "agentscope_5qAI8#nO-d@xK7$kdF+Dh"

// JavaAPIKeyEncryptor 复现 Java AESCryptUtils.encrypt 的 AES/ECB/PKCS5Padding 输出。
type JavaAPIKeyEncryptor struct {
	block cipher.Block
}

func NewJavaAPIKeyEncryptor() (*JavaAPIKeyEncryptor, error) {
	block, err := aes.NewCipher([]byte(javaAPIKeyAESKey))
	if err != nil {
		return nil, fmt.Errorf("create Java API key cipher: %w", err)
	}
	return &JavaAPIKeyEncryptor{block: block}, nil
}

func (e *JavaAPIKeyEncryptor) Encrypt(plain string) (string, error) {
	if e == nil || e.block == nil {
		return "", fmt.Errorf("Java API key encryptor is unavailable")
	}

	padded := pkcs7Pad([]byte(plain), e.block.BlockSize())
	ciphertext := make([]byte, len(padded))
	for offset := 0; offset < len(padded); offset += e.block.BlockSize() {
		e.block.Encrypt(ciphertext[offset:offset+e.block.BlockSize()], padded[offset:offset+e.block.BlockSize()])
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *JavaAPIKeyEncryptor) Decrypt(encrypted string) (string, error) {
	if e == nil || e.block == nil {
		return "", fmt.Errorf("Java API key encryptor is unavailable")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("decode Java API key ciphertext: %w", err)
	}
	if len(ciphertext) == 0 || len(ciphertext)%e.block.BlockSize() != 0 {
		return "", fmt.Errorf("invalid Java API key ciphertext length")
	}
	plain := make([]byte, len(ciphertext))
	for offset := 0; offset < len(ciphertext); offset += e.block.BlockSize() {
		e.block.Decrypt(plain[offset:offset+e.block.BlockSize()], ciphertext[offset:offset+e.block.BlockSize()])
	}
	plain, err = pkcs7Unpad(plain, e.block.BlockSize())
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func pkcs7Pad(plain []byte, blockSize int) []byte {
	padding := blockSize - len(plain)%blockSize
	padded := make([]byte, len(plain)+padding)
	copy(padded, plain)
	for index := len(plain); index < len(padded); index++ {
		padded[index] = byte(padding)
	}
	return padded
}

func pkcs7Unpad(padded []byte, blockSize int) ([]byte, error) {
	if len(padded) == 0 || len(padded)%blockSize != 0 {
		return nil, fmt.Errorf("invalid PKCS#7 padded data")
	}
	padding := int(padded[len(padded)-1])
	if padding == 0 || padding > blockSize || padding > len(padded) {
		return nil, fmt.Errorf("invalid PKCS#7 padding")
	}
	for _, value := range padded[len(padded)-padding:] {
		if int(value) != padding {
			return nil, fmt.Errorf("invalid PKCS#7 padding")
		}
	}
	return padded[:len(padded)-padding], nil
}
