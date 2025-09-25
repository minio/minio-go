/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2019-2022 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package credentials

import (
	"strings"
	"testing"
	"time"
)

func TestNewCLILoginClaims(t *testing.T) {
	port := 8080
	reqID := "test-req-id-123"

	claims := NewCLILoginClaims(port, reqID)

	if claims.Port() != port {
		t.Errorf("Expected port %d, got %d", port, claims.Port())
	}

	if claims.c.ReqID != reqID {
		t.Errorf("Expected reqID %s, got %s", reqID, claims.c.ReqID)
	}

	if claims.c.Expiry.Before(time.Now().UTC()) {
		t.Error("Expected expiry to be in the future")
	}

	if claims.c.Expiry.After(time.Now().UTC().Add(6 * time.Minute)) {
		t.Error("Expected expiry to be within 6 minutes")
	}
}

func TestCLILoginClaims_Valid(t *testing.T) {
	t.Run("Valid token", func(t *testing.T) {
		claims := NewCLILoginClaims(8080, "test-req-id")
		err := claims.c.Valid()
		if err != nil {
			t.Errorf("Expected valid token, got error: %v", err)
		}
	})

	t.Run("Expired token", func(t *testing.T) {
		claims := &CLILoginClaims{
			c: &cliLoginClaims{
				Port:   8080,
				ReqID:  "test-req-id",
				Expiry: time.Now().UTC().Add(-1 * time.Minute),
			},
		}
		err := claims.c.Valid()
		if err == nil {
			t.Error("Expected expired token to be invalid")
		}
		if !strings.Contains(err.Error(), "expired") {
			t.Errorf("Expected error to contain 'expired', got: %v", err)
		}
	})
}

func TestCLILoginClaims_ToTokenString(t *testing.T) {
	claims := NewCLILoginClaims(8080, "test-req-id")
	secret := "test-secret"

	tokenString, err := claims.ToTokenString(secret)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if tokenString == "" {
		t.Error("Expected non-empty token string")
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		t.Errorf("Expected JWT with 3 parts, got %d parts", len(parts))
	}
}

func TestCredentialsClaims_Valid(t *testing.T) {
	t.Run("Valid credentials without expiration", func(t *testing.T) {
		claims := &credentialsClaims{
			AccessKeyID:     "access-key",
			SecretAccessKey: "secret-key",
		}
		err := claims.Valid()
		if err != nil {
			t.Errorf("Expected valid credentials, got error: %v", err)
		}
	})

	t.Run("Valid credentials with future expiration", func(t *testing.T) {
		claims := &credentialsClaims{
			AccessKeyID:     "access-key",
			SecretAccessKey: "secret-key",
			Expiration:      time.Now().UTC().Add(1 * time.Hour),
		}
		err := claims.Valid()
		if err != nil {
			t.Errorf("Expected valid credentials, got error: %v", err)
		}
	})

	t.Run("Expired credentials", func(t *testing.T) {
		claims := &credentialsClaims{
			AccessKeyID:     "access-key",
			SecretAccessKey: "secret-key",
			Expiration:      time.Now().UTC().Add(-1 * time.Hour),
		}
		err := claims.Valid()
		if err == nil {
			t.Error("Expected expired credentials to be invalid")
		}
		if !strings.Contains(err.Error(), "expired") {
			t.Errorf("Expected error to contain 'expired', got: %v", err)
		}
	})
}

func TestCLILoginClaims_SignCredentials(t *testing.T) {
	cliClaims := NewCLILoginClaims(8080, "test-req-id")
	creds := Value{
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		SessionToken:    "test-session-token",
		Expiration:      time.Now().UTC().Add(1 * time.Hour),
	}

	tokenString, err := cliClaims.SignCredentials(creds)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if tokenString == "" {
		t.Error("Expected non-empty token string")
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		t.Errorf("Expected JWT with 3 parts, got %d parts", len(parts))
	}
}

func TestParseCLILoginClaims(t *testing.T) {
	originalClaims := NewCLILoginClaims(8080, "test-req-id")
	secret := "test-secret"

	tokenString, err := originalClaims.ToTokenString(secret)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	t.Run("Valid token", func(t *testing.T) {
		parsedClaims, err := ParseCLILoginClaims(tokenString, secret)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if parsedClaims == nil {
			t.Fatal("Expected non-nil parsed claims")
		}

		if parsedClaims.Port() != originalClaims.Port() {
			t.Errorf("Expected port %d, got %d", originalClaims.Port(), parsedClaims.Port())
		}

		if parsedClaims.c.ReqID != originalClaims.c.ReqID {
			t.Errorf("Expected reqID %s, got %s", originalClaims.c.ReqID, parsedClaims.c.ReqID)
		}
	})

	t.Run("Wrong secret", func(t *testing.T) {
		_, err := ParseCLILoginClaims(tokenString, "wrong-secret")
		if err == nil {
			t.Error("Expected error with wrong secret")
		}
	})

	t.Run("Invalid token format", func(t *testing.T) {
		_, err := ParseCLILoginClaims("invalid-token", secret)
		if err == nil {
			t.Error("Expected error with invalid token format")
		}
	})
}

func TestParseCredentialsClaims(t *testing.T) {
	cliClaims := NewCLILoginClaims(8080, "test-req-id")
	originalCreds := Value{
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		SessionToken:    "test-session-token",
		Expiration:      time.Now().UTC().Add(1 * time.Hour),
	}

	tokenString, err := cliClaims.SignCredentials(originalCreds)
	if err != nil {
		t.Fatalf("Failed to create credentials token: %v", err)
	}

	t.Run("Valid credentials token", func(t *testing.T) {
		parsedCreds, err := ParseSignedCredentials(tokenString, cliClaims.c.ReqID)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if parsedCreds.AccessKeyID != originalCreds.AccessKeyID {
			t.Errorf("Expected AccessKeyID %s, got %s", originalCreds.AccessKeyID, parsedCreds.AccessKeyID)
		}

		if parsedCreds.SecretAccessKey != originalCreds.SecretAccessKey {
			t.Errorf("Expected SecretAccessKey %s, got %s", originalCreds.SecretAccessKey, parsedCreds.SecretAccessKey)
		}

		if parsedCreds.SessionToken != originalCreds.SessionToken {
			t.Errorf("Expected SessionToken %s, got %s", originalCreds.SessionToken, parsedCreds.SessionToken)
		}

		if !parsedCreds.Expiration.Equal(originalCreds.Expiration) {
			t.Errorf("Expected Expiration %v, got %v", originalCreds.Expiration, parsedCreds.Expiration)
		}
	})

	t.Run("Wrong reqID", func(t *testing.T) {
		_, err := ParseSignedCredentials(tokenString, "wrong-req-id")
		if err == nil {
			t.Error("Expected error with wrong reqID")
		}
	})

	t.Run("Invalid token format", func(t *testing.T) {
		_, err := ParseSignedCredentials("invalid-token", cliClaims.c.ReqID)
		if err == nil {
			t.Error("Expected error with invalid token format")
		}
	})
}

func TestRoundTrip_CLILoginClaims(t *testing.T) {
	port := 9000
	reqID := "round-trip-test-req-id"
	secret := "round-trip-secret"

	originalClaims := NewCLILoginClaims(port, reqID)

	tokenString, err := originalClaims.ToTokenString(secret)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	parsedClaims, err := ParseCLILoginClaims(tokenString, secret)
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if parsedClaims.Port() != originalClaims.Port() {
		t.Errorf("Round trip failed: expected port %d, got %d", originalClaims.Port(), parsedClaims.Port())
	}

	if parsedClaims.c.ReqID != originalClaims.c.ReqID {
		t.Errorf("Round trip failed: expected reqID %s, got %s", originalClaims.c.ReqID, parsedClaims.c.ReqID)
	}
}

func TestRoundTrip_CredentialsClaims(t *testing.T) {
	cliClaims := NewCLILoginClaims(8080, "credentials-round-trip-req-id")
	originalCreds := Value{
		AccessKeyID:     "round-trip-access-key",
		SecretAccessKey: "round-trip-secret-key",
		SessionToken:    "round-trip-session-token",
		Expiration:      time.Now().UTC().Add(2 * time.Hour),
	}

	tokenString, err := cliClaims.SignCredentials(originalCreds)
	if err != nil {
		t.Fatalf("Failed to sign credentials: %v", err)
	}

	parsedCreds, err := ParseSignedCredentials(tokenString, cliClaims.c.ReqID)
	if err != nil {
		t.Fatalf("Failed to parse credentials: %v", err)
	}

	if parsedCreds.AccessKeyID != originalCreds.AccessKeyID {
		t.Errorf("Round trip failed: expected AccessKeyID %s, got %s", originalCreds.AccessKeyID, parsedCreds.AccessKeyID)
	}

	if parsedCreds.SecretAccessKey != originalCreds.SecretAccessKey {
		t.Errorf("Round trip failed: expected SecretAccessKey %s, got %s", originalCreds.SecretAccessKey, parsedCreds.SecretAccessKey)
	}

	if parsedCreds.SessionToken != originalCreds.SessionToken {
		t.Errorf("Round trip failed: expected SessionToken %s, got %s", originalCreds.SessionToken, parsedCreds.SessionToken)
	}

	if !parsedCreds.Expiration.Equal(originalCreds.Expiration) {
		t.Errorf("Round trip failed: expected Expiration %v, got %v", originalCreds.Expiration, parsedCreds.Expiration)
	}
}
