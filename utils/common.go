// utils/common.go
package utils

import (
    "crypto/rand"
    "encoding/hex"
    "log"
    "time"
    "math"
)

// GenerateID generates a unique ID with an optional prefix
func GenerateID(prefix string) string {
    bytes := make([]byte, 8)
    if _, err := rand.Read(bytes); err != nil {
        log.Fatalf("Failed to generate random bytes: %v", err)
    }
    return prefix + "_" + hex.EncodeToString(bytes)
}

// TimeNow returns the current Unix timestamp
func TimeNow() int64 {
    return time.Now().Unix()
}

// Contains checks if a string slice contains a specific string
func Contains(slice []string, str string) bool {
    for _, s := range slice {
        if s == str {
            return true
        }
    }
    return false
}

// RetryWithTimeout executes a function with retries
func RetryWithTimeout(fn func() error, attempts int, delay time.Duration) error {
    var err error
    for i := 0; i < attempts; i++ {
        if err = fn(); err == nil {
            return nil
        }
        if i < attempts-1 {
            time.Sleep(delay)
        }
    }
    return err
}

// CalculateHotScore calculates a Reddit-like hot score
func CalculateHotScore(ups int, downs int, timestamp int64) float64 {
    score := float64(ups - downs)
    order := math.Log10(math.Max(math.Abs(score), 1))
    sign := 1.0
    if score < 0 {
        sign = -1
    }
    
    seconds := float64(timestamp - 1577836800) // Time since 2020-01-01
    return sign*order + seconds/45000
}