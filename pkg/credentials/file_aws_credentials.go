/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2017 MinIO, Inc.
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
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

// ssoPortalRequestTimeout bounds the AWS SSO portal role-credentials request.
const ssoPortalRequestTimeout = time.Minute

// A externalProcessCredentials stores the output of a credential_process
type externalProcessCredentials struct {
	Version         int
	SessionToken    string
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string
	Expiration      time.Time
}

// A ssoCredentials stores the response of fetching role credentials for
// an AWS SSO role from the SSO portal.
type ssoCredentials struct {
	RoleCredentials ssoRoleCredentials `json:"roleCredentials"`
}

// A ssoRoleCredentials stores the role-specific credentials portion of
// an SSO role credentials response. Expiration is milliseconds since
// the Unix epoch.
type ssoRoleCredentials struct {
	AccessKeyID     string `json:"accessKeyId"`
	Expiration      int64  `json:"expiration"`
	SecretAccessKey string `json:"secretAccessKey"`
	SessionToken    string `json:"sessionToken"`
}

func (s ssoRoleCredentials) expirationTime() time.Time {
	return time.Unix(0, s.Expiration*int64(time.Millisecond))
}

// ssoCachedToken is the subset of the cached token file written by
// `aws sso login` under ~/.aws/sso/cache/ that we need.
type ssoCachedToken struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
	Region      string    `json:"region"`
}

// A FileAWSCredentials retrieves credentials from the current user's home
// directory, and keeps track if those credentials are expired.
//
// Profile ini file example: $HOME/.aws/credentials
//
// Profiles configured for AWS SSO (sso_session / sso_* keys) live in the AWS
// config file ($HOME/.aws/config), whose non-default sections are named
// "profile <name>"; point Filename at that file to use them.
type FileAWSCredentials struct {
	Expiry

	// Path to the shared credentials file.
	//
	// If empty will look for "AWS_SHARED_CREDENTIALS_FILE" env variable. If the
	// env value is empty will default to current user's home directory.
	// Linux/OSX: "$HOME/.aws/credentials"
	// Windows:   "%USERPROFILE%\.aws\credentials"
	Filename string

	// AWS Profile to extract credentials from the shared credentials file. If empty
	// will default to environment variable "AWS_PROFILE" or "default" if
	// environment variable is also not set.
	Profile string

	// retrieved states if the credentials have been successfully retrieved.
	retrieved bool

	// overrideSSOCacheDir overrides, for tests, the directory holding the
	// cached SSO tokens (defaults to ~/.aws/sso/cache).
	overrideSSOCacheDir string

	// overrideSSOPortalURL overrides, for tests, the AWS SSO portal URL
	// serving role credentials.
	overrideSSOPortalURL string
}

// NewFileAWSCredentials returns a pointer to a new Credentials object
// wrapping the Profile file provider.
func NewFileAWSCredentials(filename, profile string) *Credentials {
	return New(&FileAWSCredentials{
		Filename: filename,
		Profile:  profile,
	})
}

func (p *FileAWSCredentials) retrieve(cc *CredContext) (Value, error) {
	if p.Filename == "" {
		p.Filename = os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
		if p.Filename == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return Value{}, err
			}
			p.Filename = filepath.Join(homeDir, ".aws", "credentials")
		}
	}
	if p.Profile == "" {
		p.Profile = os.Getenv("AWS_PROFILE")
		if p.Profile == "" {
			p.Profile = "default"
		}
	}

	p.retrieved = false

	iniConfig, iniProfile, err := loadProfile(p.Filename, p.Profile)
	if err != nil {
		return Value{}, err
	}

	// Default to empty string if not found.
	id := iniProfile.Key("aws_access_key_id")
	// Default to empty string if not found.
	secret := iniProfile.Key("aws_secret_access_key")
	// Default to empty string if not found.
	token := iniProfile.Key("aws_session_token")

	// If credential_process is defined, obtain credentials by executing
	// the external process
	credentialProcess := strings.TrimSpace(iniProfile.Key("credential_process").String())
	if credentialProcess != "" {
		args := strings.Fields(credentialProcess)
		if len(args) <= 1 {
			return Value{}, errors.New("invalid credential process args")
		}
		cmd := exec.Command(args[0], args[1:]...)
		out, err := cmd.Output()
		if err != nil {
			return Value{}, err
		}
		var externalProcessCredentials externalProcessCredentials
		err = json.Unmarshal([]byte(out), &externalProcessCredentials)
		if err != nil {
			return Value{}, err
		}
		p.retrieved = true
		p.SetExpiration(externalProcessCredentials.Expiration, DefaultExpiryWindow)
		return Value{
			AccessKeyID:     externalProcessCredentials.AccessKeyID,
			SecretAccessKey: externalProcessCredentials.SecretAccessKey,
			SessionToken:    externalProcessCredentials.SessionToken,
			Expiration:      externalProcessCredentials.Expiration,
			SignerType:      SignatureV4,
		}, nil
	}

	// If the profile is configured for AWS SSO, obtain credentials from the
	// token cached by `aws sso login`.
	if iniProfile.Key("sso_role_name").String() != "" {
		ssoCreds, err := p.getSSOCredentials(cc, iniConfig, iniProfile)
		if err != nil {
			return Value{}, err
		}
		expiration := ssoCreds.RoleCredentials.expirationTime()
		p.retrieved = true
		p.SetExpiration(expiration, DefaultExpiryWindow)
		return Value{
			AccessKeyID:     ssoCreds.RoleCredentials.AccessKeyID,
			SecretAccessKey: ssoCreds.RoleCredentials.SecretAccessKey,
			SessionToken:    ssoCreds.RoleCredentials.SessionToken,
			Expiration:      expiration,
			SignerType:      SignatureV4,
		}, nil
	}

	// A profile carrying SSO configuration without sso_role_name cannot
	// engage the SSO flow above; if it also has no static keys, returning
	// the empty values below would silently yield anonymous credentials.
	if id.String() == "" && secret.String() == "" &&
		(iniProfile.Key("sso_session").String() != "" || iniProfile.Key("sso_start_url").String() != "") {
		return Value{}, errors.New("profile has SSO configuration but no sso_role_name, and no static credentials")
	}

	p.retrieved = true
	return Value{
		AccessKeyID:     id.String(),
		SecretAccessKey: secret.String(),
		SessionToken:    token.String(),
		SignerType:      SignatureV4,
	}, nil
}

// getSSOCredentials fetches role credentials for an SSO-configured profile
// from the AWS SSO portal, using the access token cached by `aws sso login`.
func (p *FileAWSCredentials) getSSOCredentials(cc *CredContext, iniConfig *ini.File, iniProfile *ini.Section) (ssoCredentials, error) {
	ssoAccountID := iniProfile.Key("sso_account_id").String()
	ssoRoleName := iniProfile.Key("sso_role_name").String()
	if ssoAccountID == "" {
		return ssoCredentials{}, errors.New("profile defines sso_role_name but no sso_account_id")
	}

	// Modern config: the profile references an [sso-session <name>] section
	// and the cached token file is named after the SHA1 of the session name.
	// Legacy config: sso_start_url/sso_region live directly on the profile
	// and the cached token file is named after the SHA1 of the start URL.
	ssoSessionName := iniProfile.Key("sso_session").String()
	cacheKey := ssoSessionName
	ssoRegion := iniProfile.Key("sso_region").String()
	if ssoSessionName != "" {
		if sessionSection, err := iniConfig.GetSection("sso-session " + ssoSessionName); err == nil {
			if region := sessionSection.Key("sso_region").String(); region != "" {
				ssoRegion = region
			}
		}
	} else {
		startURL := iniProfile.Key("sso_start_url").String()
		if startURL == "" {
			return ssoCredentials{}, errors.New("profile defines sso_role_name but neither sso_session nor sso_start_url")
		}
		cacheKey = startURL
	}

	hash := sha1.Sum([]byte(cacheKey))
	cachedTokenFilename := hex.EncodeToString(hash[:]) + ".json"

	cacheDir := p.overrideSSOCacheDir
	if cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return ssoCredentials{}, fmt.Errorf("getting home dir: %w", err)
		}
		cacheDir = filepath.Join(homeDir, ".aws", "sso", "cache")
	}
	cachedTokenPath := filepath.Join(cacheDir, cachedTokenFilename)
	cachedTokenRaw, err := os.ReadFile(cachedTokenPath)
	if err != nil {
		return ssoCredentials{}, fmt.Errorf("reading cached SSO token %q (try `aws sso login`): %w", cachedTokenPath, err)
	}

	var cachedToken ssoCachedToken
	if err := json.Unmarshal(cachedTokenRaw, &cachedToken); err != nil {
		return ssoCredentials{}, fmt.Errorf("parsing cached SSO token %q: %w", cachedTokenPath, err)
	}
	now := time.Now
	if p.CurrentTime != nil {
		now = p.CurrentTime
	}
	if cachedToken.ExpiresAt.Before(now()) {
		return ssoCredentials{}, errors.New("cached SSO token is expired, refresh it with `aws sso login`")
	}

	if ssoRegion == "" {
		ssoRegion = cachedToken.Region
	}
	if ssoRegion == "" {
		return ssoCredentials{}, errors.New("unable to determine AWS SSO region from profile, sso-session or cached token")
	}

	portalURL := p.overrideSSOPortalURL
	if portalURL == "" {
		portalURL = fmt.Sprintf("https://portal.sso.%s.amazonaws.com", ssoRegion)
	}
	ctx, cancel := context.WithTimeout(context.Background(), ssoPortalRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, portalURL+"/federation/credentials", nil)
	if err != nil {
		return ssoCredentials{}, fmt.Errorf("creating SSO role credentials request: %w", err)
	}
	req.Header.Set("x-amz-sso_bearer_token", cachedToken.AccessToken)
	query := req.URL.Query()
	query.Add("account_id", ssoAccountID)
	query.Add("role_name", ssoRoleName)
	req.URL.RawQuery = query.Encode()

	if cc == nil {
		cc = defaultCredContext
	}
	client := cc.Client
	if client == nil {
		client = defaultCredContext.Client
	}
	resp, err := client.Do(req)
	if err != nil {
		return ssoCredentials{}, fmt.Errorf("fetching SSO role credentials: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return ssoCredentials{}, fmt.Errorf("fetching SSO role credentials: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var ssoCreds ssoCredentials
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&ssoCreds); err != nil {
		return ssoCredentials{}, fmt.Errorf("parsing SSO role credentials response: %w", err)
	}
	if ssoCreds.RoleCredentials.AccessKeyID == "" || ssoCreds.RoleCredentials.SecretAccessKey == "" {
		return ssoCredentials{}, errors.New("SSO portal returned empty role credentials")
	}
	return ssoCreds, nil
}

// Retrieve reads and extracts the shared credentials from the current
// users home directory.
//
// Deprecated: Retrieve() exists for historical compatibility and should not
// be used. To get new credentials use the RetrieveWithCredContext function.
func (p *FileAWSCredentials) Retrieve() (Value, error) {
	return p.retrieve(nil)
}

// RetrieveWithCredContext retrieves credentials from the file like Retrieve,
// using the context's HTTP client for any SSO portal call.
func (p *FileAWSCredentials) RetrieveWithCredContext(cc *CredContext) (Value, error) {
	return p.retrieve(cc)
}

// loadProfile loads from the file pointed to by shared credentials filename for profile.
// The credentials retrieved from the profile will be returned or error. Error will be
// returned if it fails to read from the file, or the data is invalid.
func loadProfile(filename, profile string) (*ini.File, *ini.Section, error) {
	config, err := ini.Load(filename)
	if err != nil {
		return nil, nil, err
	}
	iniProfile, err := config.GetSection(profile)
	if err != nil {
		// AWS config files (~/.aws/config) name non-default profile
		// sections "profile <name>".
		var sectionErr error
		iniProfile, sectionErr = config.GetSection("profile " + profile)
		if sectionErr != nil {
			return nil, nil, err
		}
	}
	return config, iniProfile, nil
}
