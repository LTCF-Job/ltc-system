package crypto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateNationalID(t *testing.T) {
	tests := []struct {
		name  string
		nid   string
		valid bool
	}{
		{"合法身分證 (蔡曾切)", "A202559750", true},
		{"合法身分證 (司機郭澤威)", "G121806465", true},
		{"合法身分證 (服務人員)", "K120098177", true},
		{"合法外來人口統一證號 (男性)", "A800000014", true},
		{"合法外來人口統一證號 (女性)", "A900000016", true},
		{"檢查碼錯誤", "A202559751", false},
		{"字數不足", "A20255975", false},
		{"含非法字元", "A20255975A", false},
		{"首碼非英文字母", "1202559750", false},
		{"第二碼非1,2,8,9", "A302559750", false},
		{"小寫字母自動轉大寫驗證", "a202559750", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateNationalID(tt.nid)
			assert.Equal(t, tt.valid, got)
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := []byte("01234567890123456789012345678901") // 32 bytes key
	wrongKey := []byte("11234567890123456789012345678901")
	plain := "A202559750"

	// 加密
	cipher1, err := Encrypt(plain, key)
	require.NoError(t, err)
	require.NotEmpty(t, cipher1)

	// 同一輸入每次 Nonce 不同，密文不同
	cipher2, err := Encrypt(plain, key)
	require.NoError(t, err)
	assert.False(t, bytes.Equal(cipher1, cipher2), "ciphertexts should differ due to random nonce")

	// 正確解密
	decrypted1, err := Decrypt(cipher1, key)
	require.NoError(t, err)
	assert.Equal(t, plain, decrypted1)

	decrypted2, err := Decrypt(cipher2, key)
	require.NoError(t, err)
	assert.Equal(t, plain, decrypted2)

	// 錯誤金鑰解密應報錯，不得矇混
	_, err = Decrypt(cipher1, wrongKey)
	assert.ErrorIs(t, err, ErrDecryptionFailed)

	// 竄改密文應報錯
	corrupted := make([]byte, len(cipher1))
	copy(corrupted, cipher1)
	corrupted[len(corrupted)-1] ^= 0xFF
	_, err = Decrypt(corrupted, key)
	assert.ErrorIs(t, err, ErrDecryptionFailed)

	// 密文長度不足
	_, err = Decrypt([]byte("short"), key)
	assert.ErrorIs(t, err, ErrCiphertextTooShort)

	// 金鑰長度不合 32 bytes
	_, err = Encrypt(plain, []byte("short-key"))
	assert.ErrorIs(t, err, ErrInvalidKeySize)
}

func TestIndex(t *testing.T) {
	hmacKey := []byte("hmac-key-01234567890123456789012")
	plain1 := "A202559750"
	plain2 := "G121806465"

	idx1a := Index(plain1, hmacKey)
	idx1b := Index(plain1, hmacKey)
	idx2 := Index(plain2, hmacKey)

	// 同一輸入輸出固定
	assert.True(t, bytes.Equal(idx1a, idx1b))
	// 不同輸入輸出不同
	assert.False(t, bytes.Equal(idx1a, idx2))
}

func TestMask(t *testing.T) {
	assert.Equal(t, "A20***9750", Mask("A202559750"))
	assert.Equal(t, "G12***6465", Mask("G121806465"))
	assert.Equal(t, "******", Mask("123456"))
}
