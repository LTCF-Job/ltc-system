package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

var (
	ErrInvalidKeySize    = errors.New("key must be exactly 32 bytes for AES-256")
	ErrCiphertextTooShort = errors.New("ciphertext too short to contain nonce")
	ErrDecryptionFailed  = errors.New("decryption failed (authentication tag mismatch or corrupted data)")
	ErrInvalidNationalID = errors.New("invalid national identification number")
)

var letterCodeMap = map[byte]int{
	'A': 10, 'B': 11, 'C': 12, 'D': 13, 'E': 14, 'F': 15, 'G': 16, 'H': 17, 'I': 34,
	'J': 18, 'K': 19, 'L': 20, 'M': 21, 'N': 22, 'O': 35, 'P': 23, 'Q': 24, 'R': 25,
	'S': 26, 'T': 27, 'U': 28, 'V': 29, 'W': 32, 'X': 30, 'Y': 31, 'Z': 33,
}

// ValidateNationalID 驗證中華民國身分證與外來人口統一證號檢查碼。
func ValidateNationalID(nid string) bool {
	nid = strings.ToUpper(strings.TrimSpace(nid))
	if len(nid) != 10 {
		return false
	}

	firstChar := nid[0]
	code, exists := letterCodeMap[firstChar]
	if !exists {
		return false
	}

	// 第 2 碼（性別碼）：本國人 1, 2；外來人口居留證 8, 9
	secondChar := nid[1]
	if secondChar != '1' && secondChar != '2' && secondChar != '8' && secondChar != '9' {
		return false
	}

	for i := 1; i < 10; i++ {
		if !unicode.IsDigit(rune(nid[i])) {
			return false
		}
	}

	n1 := code / 10
	n2 := code % 10

	sum := n1 + n2*9
	weights := []int{8, 7, 6, 5, 4, 3, 2, 1}
	for i := 0; i < 8; i++ {
		digit := int(nid[i+1] - '0')
		sum += digit * weights[i]
	}
	checkDigit := int(nid[9] - '0')
	sum += checkDigit

	return sum%10 == 0
}

// Encrypt 使用 AES-256-GCM 與隨機 12 位元組 nonce 加密明文字串。
func Encrypt(plain string, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate random nonce: %w", err)
	}

	// 輸出 nonce ‖ ciphertext ‖ tag
	ciphertext := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return ciphertext, nil
}

// Decrypt 使用 AES-256-GCM 解密密文並驗證完整性。
func Decrypt(ciphertext []byte, key []byte) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrCiphertextTooShort
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// Index 計算用於資料庫查詢與唯一性比對的 HMAC-SHA256 索引值。
func Index(plain string, hmacKey []byte) []byte {
	mac := hmac.New(sha256.New, hmacKey)
	mac.Write([]byte(plain))
	return mac.Sum(nil)
}

// Mask 將身分證遮罩為「前 3 碼 + *** + 後 4 碼」（如 A20***9750）。
func Mask(plain string) string {
	plain = strings.TrimSpace(plain)
	if len(plain) < 10 {
		return strings.Repeat("*", len(plain))
	}
	return plain[:3] + "***" + plain[len(plain)-4:]
}
