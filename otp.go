package main

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OTP struct {
	Username string
	Key      string
	Created  time.Time
}

type RetentionMap map[string]OTP

func NewRetentionMap(ctx context.Context, retentionPeriod time.Duration) RetentionMap {
	rm := make(RetentionMap)

	// spawn a process that runs in the background and checks for expired OTPs
	go rm.Retention(ctx, retentionPeriod)

	return rm
}

func (rm RetentionMap) NewOTP(username string) OTP {
	otp := OTP{
		Username: username,
		Key:      uuid.NewString(),
		Created:  time.Now(),
	}

	rm[otp.Key] = otp
	return otp
}

func (rm RetentionMap) VerifyOTP(otp string) string {
	if _, exists := rm[otp]; !exists {
		return ""
	}
	username := rm[otp].Username
	delete(rm, otp)
	return username
}

func (rm RetentionMap) Retention(ctx context.Context, retentionPeriod time.Duration) {
	ticker := time.NewTicker(400 * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			for _, otp := range rm {
				if otp.Created.Add(retentionPeriod).Before(time.Now()) {
					delete(rm, otp.Key)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
