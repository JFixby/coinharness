package coinharness

import (
	"testing"
)

func GetBalance(t *testing.T, w Wallet) *GetBalanceResult {
	b, e := w.GetBalance("")
	if e != nil {
		t.Fatalf("GetBalance error: %v", e)
	}
	return b
}
