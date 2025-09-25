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
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type CLILoginClaims struct {
	c *cliLoginClaims
}

type cliLoginClaims struct {
	Port   int       `json:"port"`
	ReqID  string    `json:"req_id"`
	Expiry time.Time `json:"expiry"`
}

func NewCLILoginClaims(port int, reqID string) *CLILoginClaims {
	return &CLILoginClaims{
		c: &cliLoginClaims{
			Port:   port,
			ReqID:  reqID,
			Expiry: time.Now().UTC().Add(5 * time.Minute),
		},
	}
}
func ParseCLILoginClaims(tokenString, secret string) (*CLILoginClaims, error) {
	decodedToken, err := base64.RawURLEncoding.DecodeString(tokenString)
	if err != nil {
		return nil, err
	}

	claims := &cliLoginClaims{}
	_, err = jwt.ParseWithClaims(string(decodedToken), claims, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	return &CLILoginClaims{c: claims}, nil
}

func (c *cliLoginClaims) Valid() error {
	if time.Now().UTC().After(c.Expiry) {
		return errors.New("token is expired")
	}
	return nil
}

func (c *CLILoginClaims) Port() int {
	return c.c.Port
}

func (c *CLILoginClaims) ToTokenString(secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, c.c)
	sString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString([]byte(sString)), nil
}

func (c *CLILoginClaims) SignCredentials(creds Value) (string, error) {
	claims := &credentialsClaims{
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
		Expiration:      creds.Expiration,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	sString, err := token.SignedString([]byte(c.c.ReqID))
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString([]byte(sString)), nil
}

type credentialsClaims struct {
	AccessKeyID     string    `json:"access_key_id"`
	SecretAccessKey string    `json:"secret_access_key"`
	SessionToken    string    `json:"session_token,omitempty"`
	Expiration      time.Time `json:"expiration,omitempty"`
}

func (c *credentialsClaims) Valid() error {
	if !c.Expiration.IsZero() && time.Now().UTC().After(c.Expiration) {
		return errors.New("credentials token is expired")
	}
	return nil
}

func ParseSignedCredentials(tokenString, reqID string) (Value, error) {
	decodedToken, err := base64.RawURLEncoding.DecodeString(tokenString)
	if err != nil {
		return Value{}, err
	}

	claims := &credentialsClaims{}
	_, err = jwt.ParseWithClaims(string(decodedToken), claims, func(token *jwt.Token) (any, error) {
		return []byte(reqID), nil
	})
	if err != nil {
		return Value{}, err
	}

	return Value{
		AccessKeyID:     claims.AccessKeyID,
		SecretAccessKey: claims.SecretAccessKey,
		SessionToken:    claims.SessionToken,
		Expiration:      claims.Expiration,
	}, nil
}
