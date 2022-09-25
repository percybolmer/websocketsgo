package main

import (
	"context"
	"testing"
	"time"
)

func TestOTP_Retention(t *testing.T) {

	// Create context with cancel to stop goroutine
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// Create RM and add a few OTP with a few Seconds in between
	rm := NewRetentionMap(ctx, 1*time.Second)

	rm.NewOTP()
	rm.NewOTP()

	time.Sleep(2 * time.Second)

	otp := rm.NewOTP()

	// Make sure that only 1 password is still left and it matches the latest
	if len(rm) != 1 {
		t.Error("Failed to clean up")
	}

	if rm[otp.Key] != otp {
		t.Error("The key should still be in place")
	}
	cancel()
}
