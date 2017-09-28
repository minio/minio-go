/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2017 Minio, Inc.
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

package encrypt

import (
	"crypto/aes"

	"golang.org/x/crypto/scrypt"
)

// Key - generic interface to encrypt/decrypt a key.
// We use it to encrypt/decrypt content key which is the key
// that encrypt/decrypt object data.
type Key interface {
	// Encrypt data using to the set encryption key
	Encrypt([]byte) ([]byte, error)
	// Decrypt data using to the set encryption key
	Decrypt([]byte) ([]byte, error)
}

// SymmetricKey - encrypts data with a symmetric master key
type SymmetricKey struct {
	masterKey []byte
}

// Encrypt passed bytes
func (s *SymmetricKey) Encrypt(plain []byte) ([]byte, error) {
	// Initialize an AES encryptor using a master key
	keyBlock, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return []byte{}, err
	}

	// Pad the key before encryption
	plain, _ = pkcs5Pad(plain, aes.BlockSize)

	encKey := []byte{}
	encPart := make([]byte, aes.BlockSize)

	// Encrypt the passed key by block
	for {
		if len(plain) < aes.BlockSize {
			break
		}
		// Encrypt the passed key
		keyBlock.Encrypt(encPart, plain[:aes.BlockSize])
		// Add the encrypted block to the total encrypted key
		encKey = append(encKey, encPart...)
		// Pass to the next plain block
		plain = plain[aes.BlockSize:]
	}
	return encKey, nil
}

// Decrypt passed bytes
func (s *SymmetricKey) Decrypt(cipher []byte) ([]byte, error) {
	// Initialize AES decrypter
	keyBlock, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, err
	}

	var plain []byte
	plainPart := make([]byte, aes.BlockSize)

	// Decrypt the encrypted data block by block
	for {
		if len(cipher) < aes.BlockSize {
			break
		}
		keyBlock.Decrypt(plainPart, cipher[:aes.BlockSize])
		// Add the decrypted block to the total result
		plain = append(plain, plainPart...)
		// Pass to the next cipher block
		cipher = cipher[aes.BlockSize:]
	}

	// Unpad the resulted plain data
	plain, err = pkcs5Unpad(plain, aes.BlockSize)
	if err != nil {
		return nil, err
	}

	return plain, nil
}

// NewSymmetricKey creates a symmetric en/decryption
// key from the provided byte slice.
func NewSymmetricKey(b []byte) *SymmetricKey {
	return &SymmetricKey{masterKey: b}
}

// PBKDF specifies a password-based key-derivation-function
// to derive a symmetric encryption key from a password.
type PBKDF func([]byte, []byte) ([]byte, error)

func scrypt2009(password, salt []byte) ([]byte, error) {
	return scrypt.Key(password, salt, 16384, 8, 1, 32)
}

// DeriveKey derives a 256 bit symmetric encryption key from a
// password and a salt. The salt may be nil.
//
// The key is derived using
// scrypt with the parameters N=16384, r=8 and p=1.
func DeriveKey(password string, salt []byte) *SymmetricKey {
	key, err := DeriveKeyUsing(scrypt2009, password, salt)
	if err != nil {
		panic("key deriviation failed for fixed PBKDF - please report this bug at: https://github.com/minio/minio-go/issues")
	}
	return NewSymmetricKey(key)
}

// DeriveKeyUsing derives a symmetric encryption key from a
// password and a salt using the provided PBKDF.
func DeriveKeyUsing(pbkdf PBKDF, password string, salt []byte) ([]byte, error) {
	return pbkdf([]byte(password), salt)
}
