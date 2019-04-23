/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2019 MinIO, Inc.
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
	"encoding/xml"
	"errors"
	"log"
	"net/http"
	"net/url"
	"time"

	kclient "github.com/minio/gokrb5/v7/client"
	kcfg "github.com/minio/gokrb5/v7/config"
	"github.com/minio/gokrb5/v7/crypto"
	"github.com/minio/gokrb5/v7/messages"
	"github.com/minio/gokrb5/v7/types"
)

// AssumeRoleWithKerberosResponse contains the result of successful
// AssumeRoleWithKerberosIdentity request
type AssumeRoleWithKerberosResponse struct {
	XMLName          xml.Name               `xml:"https://sts.amazonaws.com/doc/2011-06-15/ AssumeRoleWithClientGrantsResponse" json:"-"`
	Result           KerberosIdentityResult `xml:"AssumeRoleWithKerberosIdentity"`
	ResponseMetadata struct {
		RequestID string `xml:"RequestId,omitempty"`
	} `xml:"ResponseMetadata,omitempty"`
}

// KerberosIdentityResult - contains credentials for a successful
// AssumeRoleWithKerberosIdentity request.
type KerberosIdentityResult struct {
	Credentials struct {
		AccessKey    string    `xml:"AccessKeyId" json:"accessKey,omitempty"`
		SecretKey    string    `xml:"SecretAccessKey" json:"secretKey,omitempty"`
		Expiration   time.Time `xml:"Expiration" json:"expiration,omitempty"`
		SessionToken string    `xml:"SessionToken" json:"sessionToken,omitempty"`
	} `xml:",omitempty"`

	SubjectFromToken string `xml:",omitempty"`
}

func getKrbConfig(krbConfigFile string) *kcfg.Config {
	cfg, err := kcfg.Load(krbConfigFile)
	if err != nil {
		log.Fatalf("Error loading Kerberos client configuration file (%s): %v", krbConfigFile, err)
	}
	return cfg
}

// KerberosIdentity retrieves credentials from MinIO
type KerberosIdentity struct {
	Expiry

	stsEndpoint string

	// Minio server principal for Kerberos
	principal string

	// Kerberos client config
	krbConfig *kcfg.Config

	krbClient *kclient.Client
}

// NewKerberosIdentity returns new credentials object that uses
// Kerberos Identity.
//
// The krbConfig can be left as nil - in this case the library will
// load the configuration from /etc/krb5.conf.
//
// The krbRealm can be left empty, in which case the library will try
// to pick up the default realm from client configuration.
func NewKerberosIdentity(stsEndpoint string, krbConfig *kcfg.Config, krbUserPrincipal, krbPassword, krbRealm, minioServicePrincipal string) (*Credentials, error) {

	if krbConfig == nil {
		var err error
		krbConfig, err = kcfg.Load("/etc/krb5.conf")
		if err != nil {
			return nil, err
		}
	}

	if krbRealm == "" {
		krbRealm = krbConfig.LibDefaults.DefaultRealm
	}

	return New(&KerberosIdentity{
		stsEndpoint: stsEndpoint,
		principal:   minioServicePrincipal,
		krbConfig:   krbConfig,
		krbClient:   kclient.NewClientWithPassword(krbUserPrincipal, krbRealm, krbPassword, krbConfig),
	}), nil
}

// Retrieve gets the credential by calling the MinIO STS API for
// Kerberos on the configured stsEndpoint.
func (k *KerberosIdentity) Retrieve() (value Value, err error) {
	cl := k.krbClient
	tkt, key, kerr := cl.GetServiceTicket(k.principal)
	if kerr != nil {
		err = kerr
		return
	}

	auth, kerr := types.NewAuthenticator(cl.Credentials.Realm(), cl.Credentials.CName())
	if kerr != nil {
		err = kerr
		return
	}

	etype, kerr := crypto.GetEtype(key.KeyType)
	if kerr != nil {
		err = kerr
		return
	}

	err = auth.GenerateSeqNumberAndSubKey(key.KeyType, etype.GetKeyByteSize())
	if err != nil {
		return
	}

	auth.Cksum = types.Checksum{
		CksumType: -1,
		Checksum:  []byte{0, 0, 0, 0},
	}
	APReq, kerr := messages.NewAPReq(tkt, key, auth)
	if kerr != nil {
		err = kerr
		return
	}
	APReqBytes, kerr := APReq.Marshal()
	if kerr != nil {
		err = kerr
		return
	}
	b64APReqStr := base64.StdEncoding.EncodeToString(APReqBytes)

	// Send APReq to STS API
	u, kerr := url.Parse(k.stsEndpoint)
	if kerr != nil {
		err = kerr
		return
	}

	clnt := &http.Client{Transport: http.DefaultTransport}
	v := url.Values{}
	v.Set("Action", "AssumeRoleWithKerberosIdentity")
	v.Set("Version", "2011-06-15")
	v.Set("APReq", b64APReqStr)

	u.RawQuery = v.Encode()

	req, kerr := http.NewRequest("POST", u.String(), nil)
	if kerr != nil {
		err = kerr
		return
	}

	resp, kerr := clnt.Do(req)
	if kerr != nil {
		err = kerr
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = errors.New(resp.Status)
		return
	}

	r := AssumeRoleWithKerberosResponse{}
	if err = xml.NewDecoder(resp.Body).Decode(&r); err != nil {
		return
	}

	cr := r.Result.Credentials
	k.SetExpiration(cr.Expiration, DefaultExpiryWindow)
	return Value{
		AccessKeyID:     cr.AccessKey,
		SecretAccessKey: cr.SecretKey,
		SessionToken:    cr.SessionToken,
		SignerType:      SignatureV4,
	}, nil
}
