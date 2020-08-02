package crypt

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Bnei-Baruch/gxydb-api/common"
	_ "github.com/Bnei-Baruch/gxydb-api/pkg/testutil"
)

func TestEncryption(t *testing.T) {
	text := []byte("some plain text")
	secret := "12345678901234567890123456789012"

	encText, err := Encrypt(text, secret)
	assert.NoError(t, err, "Encrypt error")
	assert.NotEqual(t, text, encText)

	decText, err := Decrypt(encText, secret)
	assert.NoError(t, err, "Decrypt error")
	assert.Equal(t, string(text), decText)
}

func TestEncrypt(t *testing.T) {
	encText, err := Encrypt([]byte("janusoverlord"), common.Config.Secret)
	assert.NoError(t, err, "Encrypt error")
	str := base64.StdEncoding.EncodeToString(encText)
	fmt.Println(str)

	//dStr := []byte("$2a$10$XCSPObl1.cTTn0/CZqzdE.ou2tOUp0hXwMZ\neXyLU3rq/5BaPA4irO")
	dStr, err := base64.StdEncoding.DecodeString(str)
	assert.NoError(t, err, "DecodeString error")
	decText, err := Decrypt(dStr, common.Config.Secret)
	assert.NoError(t, err, "Decrypt error")
	assert.Equal(t, "janusoverlord", decText)
}
