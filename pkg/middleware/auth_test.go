package middleware

import (
	"fmt"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestMakePassword(t *testing.T) {
	pwd := "mylovelypassword"

	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	fmt.Println(string(hash))

	err = bcrypt.CompareHashAndPassword(hash, []byte(pwd))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
