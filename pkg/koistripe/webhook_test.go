package koistripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func TestVerifyEventRejectsTamperingAndReplayOutsideTolerance(t *testing.T) {
	now := time.Now()
	payload := []byte(`{"id":"evt_1","type":"checkout.session.completed","data":{"object":{"id":"cs_1"}}}`)
	mac := hmac.New(sha256.New, []byte("whsec_test"))
	_, _ = fmt.Fprintf(mac, "%d.%s", now.Unix(), payload)
	header := fmt.Sprintf("t=%d,v1=%s", now.Unix(), hex.EncodeToString(mac.Sum(nil)))

	event, err := VerifyEvent(payload, header, "whsec_test")
	if err != nil || event.ID != "evt_1" {
		t.Fatalf("valid event rejected: %+v, %v", event, err)
	}
	if _, err := VerifyEvent(append(payload, ' '), header, "whsec_test"); err == nil {
		t.Fatal("tampered payload accepted")
	}
	old := now.Add(-6 * time.Minute)
	mac = hmac.New(sha256.New, []byte("whsec_test"))
	_, _ = fmt.Fprintf(mac, "%d.%s", old.Unix(), payload)
	header = fmt.Sprintf("t=%d,v1=%s", old.Unix(), hex.EncodeToString(mac.Sum(nil)))
	if _, err := VerifyEvent(payload, header, "whsec_test"); err == nil {
		t.Fatal("old signature accepted")
	}
}
