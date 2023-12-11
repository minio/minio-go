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
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	ini "gopkg.in/ini.v1"
)

var ErrNoExternalProcessDefined = errors.New("config file does not specify credential_process")
var ErrNoSSOConfig = errors.New("the specified config does not have sso configurations")

// A externalProcessCredentials stores the output of a credential_process
type externalProcessCredentials struct {
	Version         int
	SessionToken    string
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string
	Expiration      time.Time
}

// A ssoCredentials stores the result of getting role credentials for an
// SSO role.
type ssoCredentials struct {
	RoleCredentials ssoRoleCredentials `json:"roleCredentials"`
}

// A ssoRoleCredentials stores the role-specific credentials portion of
// an sso credentials request.
type ssoRoleCredentials struct {
	AccessKeyID     string `json:"accessKeyId"`
	Expiration      int64  `json:"expiration"`
	SecretAccessKey string `json:"secretAccessKey"`
	SessionToken    string `json:"sessionToken"`
}

func (s ssoRoleCredentials) GetExpiration() time.Time {
	return time.Unix(0, s.Expiration*int64(time.Millisecond))
}

// A FileAWSCredentials retrieves credentials from the current user's home
// directory, and keeps track if those credentials are expired.
//
// Profile ini file example: $HOME/.aws/credentials
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

	// overrideSSOCacheDir allows tests to override the path where SSO cached
	// credentials are stored (usually ~/.aws/sso/cache/ is used).
	overrideSSOCacheDir string

	// overrideSSOPortalURL allows tests to override the http URL that
	// serves SSO role tokens.
	overrideSSOPortalURL string

	// timeNow allows tests to override getting the current time to test
	// for expiration.
	timeNow func() time.Time
}

// NewFileAWSCredentials returns a pointer to a new Credentials object
// wrapping the Profile file provider.
func NewFileAWSCredentials(filename, profile string) *Credentials {
	return New(&FileAWSCredentials{
		Filename: filename,
		Profile:  profile,

		timeNow: time.Now,
	})
}

// Retrieve reads and extracts the shared credentials from the current
// users home directory.
func (p *FileAWSCredentials) Retrieve() (Value, error) {
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

	iniProfile, err := loadProfile(p.Filename, p.Profile)
	if err != nil {
		return Value{}, err
	}

	if externalProcessCreds, err := getExternalProcessCredentials(iniProfile); err == nil {
		p.retrieved = true
		p.SetExpiration(externalProcessCreds.Expiration, DefaultExpiryWindow)
		return Value{
			AccessKeyID:     externalProcessCreds.AccessKeyID,
			SecretAccessKey: externalProcessCreds.SecretAccessKey,
			SessionToken:    externalProcessCreds.SessionToken,
			SignerType:      SignatureV4,
		}, nil
	} else if err != ErrNoExternalProcessDefined {
		return Value{}, err
	}

	if ssoCreds, err := p.getSSOCredentials(iniProfile); err == nil {
		p.retrieved = true
		p.SetExpiration(ssoCreds.RoleCredentials.GetExpiration(), DefaultExpiryWindow)
		return Value{
			AccessKeyID:     ssoCreds.RoleCredentials.AccessKeyID,
			SecretAccessKey: ssoCreds.RoleCredentials.SecretAccessKey,
			SessionToken:    ssoCreds.RoleCredentials.SessionToken,
			SignerType:      SignatureV4,
		}, nil
	} else if err != ErrNoSSOConfig {
		return Value{}, err
	}

	// Default to empty string if not found.
	id := iniProfile.Key("aws_access_key_id")
	// Default to empty string if not found.
	secret := iniProfile.Key("aws_secret_access_key")
	// Default to empty string if not found.
	token := iniProfile.Key("aws_session_token")

	p.retrieved = true
	return Value{
		AccessKeyID:     id.String(),
		SecretAccessKey: secret.String(),
		SessionToken:    token.String(),
		SignerType:      SignatureV4,
	}, nil
}

// getExternalProcessCredentials calls the config credential_process, parses the process' response,
// and returns the result. If the profile ini passed does not have a credential_process,
// ErrNoExternalProcessDefined is returned.
func getExternalProcessCredentials(iniProfile *ini.Section) (externalProcessCredentials, error) {
	// If credential_process is defined, obtain credentials by executing
	// the external process
	credentialProcess := strings.TrimSpace(iniProfile.Key("credential_process").String())
	if credentialProcess == "" {
		return externalProcessCredentials{}, ErrNoExternalProcessDefined
	}

	args := strings.Fields(credentialProcess)
	if len(args) <= 1 {
		return externalProcessCredentials{}, errors.New("invalid credential process args")
	}
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.Output()
	if err != nil {
		return externalProcessCredentials{}, err
	}
	var externalProcessCreds externalProcessCredentials
	err = json.Unmarshal([]byte(out), &externalProcessCreds)
	if err != nil {
		return externalProcessCredentials{}, err
	}
	return externalProcessCreds, nil
}

type ssoCredentialsCacheFile struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
	Region      string    `json:"region"`
}

func (p *FileAWSCredentials) getSSOCredentials(iniProfile *ini.Section) (ssoCredentials, error) {
	ssoRoleName := iniProfile.Key("sso_role_name").String()
	if ssoRoleName == "" {
		return ssoCredentials{}, ErrNoSSOConfig
	}

	ssoSessionName := iniProfile.Key("sso_session").String()
	hash := sha1.New()
	if _, err := hash.Write([]byte(ssoSessionName)); err != nil {
		return ssoCredentials{}, fmt.Errorf("hashing sso session name \"%s\": %w", ssoSessionName, err)
	}

	cachedCredsFilename := fmt.Sprintf("%s.json", strings.ToLower(hex.EncodeToString(hash.Sum(nil))))

	cachedCredsFileDir := p.overrideSSOCacheDir
	if cachedCredsFileDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return ssoCredentials{}, fmt.Errorf("getting home dir: %w", err)
		}
		cachedCredsFileDir = filepath.Join(homeDir, ".aws", "sso", "cache")
	}
	cachedCredsFilepath := filepath.Join(cachedCredsFileDir, cachedCredsFilename)
	cachedCredsContentsRaw, err := ioutil.ReadFile(cachedCredsFilepath)
	if err != nil {
		return ssoCredentials{}, fmt.Errorf("reading credentials cache file \"%s\": %w", cachedCredsFilepath, err)
	}

	var cachedCredsContents ssoCredentialsCacheFile
	if err := json.Unmarshal(cachedCredsContentsRaw, &cachedCredsContents); err != nil {
		return ssoCredentials{}, fmt.Errorf("parsing cached sso credentials file \"%s\": %w", cachedCredsFilename, err)
	}
	if cachedCredsContents.ExpiresAt.Before(p.timeNow()) {
		return ssoCredentials{}, fmt.Errorf("sso credentials expired, refresh with AWS CLI")
	}

	ssoAccountID := iniProfile.Key("sso_account_id").String()

	portalURL := p.overrideSSOPortalURL
	if portalURL == "" {
		portalURL = fmt.Sprintf("https://portal.sso.%s.amazonaws.com", cachedCredsContents.Region)
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/federation/credentials", portalURL), nil)
	if err != nil {
		return ssoCredentials{}, fmt.Errorf("creating request to get role credentials: %w", err)
	}
	req.Header.Set("x-amz-sso_bearer_token", cachedCredsContents.AccessToken)
	query := req.URL.Query()
	query.Add("account_id", ssoAccountID)
	query.Add("role_name", ssoRoleName)
	req.URL.RawQuery = query.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ssoCredentials{}, fmt.Errorf("making request to get role credentials: %w", err)
	}
	defer resp.Body.Close()

	var ssoCreds ssoCredentials
	if err := json.NewDecoder(resp.Body).Decode(&ssoCreds); err != nil {
		return ssoCredentials{}, fmt.Errorf("parsing sso credentials response: %w", err)
	}

	return ssoCreds, nil
}

// loadProfiles loads from the file pointed to by shared credentials filename for profile.
// The credentials retrieved from the profile will be returned or error. Error will be
// returned if it fails to read from the file, or the data is invalid.
func loadProfile(filename, profile string) (*ini.Section, error) {
	config, err := ini.Load(filename)
	if err != nil {
		return nil, err
	}

	iniProfile, err := config.GetSection(profile)
	if err != nil {
		// aws allows specifying the profile as [profile myprofile]
		if strings.Contains(err.Error(), "does not exist") {
			iniProfile, err = config.GetSection(fmt.Sprintf("profile %s", profile))
		}
		if err != nil {
			return nil, err
		}
	}

	return iniProfile, nil
}
