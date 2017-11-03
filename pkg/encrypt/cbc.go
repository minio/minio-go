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
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// Crypt mode - encryption or decryption
type cryptMode bool

const (
	encryptMode cryptMode = true
	decryptMode           = !encryptMode
)

type aesCbcPkcs5 struct {
	key aesCbcKey
}

// NewAesCbcPkcs5 returns a new Cipher which implements the
// encryption algorithm AES-CBC-PKCS-5 as implemented by AWS.
//
// AES-CBC only provides confidentiality but no authenticity of an
// encrypted object. Therefore it is not recommended to use AES-CBC.
//
// Notice that AWS calls the padding scheme PKCS-5 but uses PKCS-7.
func NewAesCbcPkcs5(key []byte) (Cipher, error) {
	aesKey := make(aesCbcKey, len(key))
	copy(aesKey, key)
	return aesCbcPkcs5{aesKey}, nil
}

func (c aesCbcPkcs5) Seal(header map[string]string, r io.Reader) (io.ReadCloser, error) {
	contentKey := make([]byte, aes.BlockSize*2)
	if _, err := io.ReadFull(rand.Reader, contentKey); err != nil {
		return nil, err
	}
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	encryptedKey, err := c.key.Encrypt(contentKey)
	if err != nil {
		return nil, err
	}
	aesCipher, err := aes.NewCipher(contentKey)
	if err != nil {
		return nil, err
	}
	header[cseIV] = base64.StdEncoding.EncodeToString(iv)
	header[cseKey] = base64.StdEncoding.EncodeToString(encryptedKey)
	header[cseAlgorithm] = "AES/CBC/PKCS5"

	return &aesCbcPkcs5Reader{
		stream:    r,
		srcBuf:    bytes.NewBuffer(nil),
		dstBuf:    bytes.NewBuffer(nil),
		cryptMode: encryptMode,
		blockMode: cipher.NewCBCEncrypter(aesCipher, iv),
	}, nil
}

func (c aesCbcPkcs5) Open(header map[string]string, r io.Reader) (io.ReadCloser, error) {
	if header[cseAlgorithm] != "AES/CBC/PKCS5" {
		return nil, errors.New("invalid encryption algorithm")
	}
	iv, err := base64.StdEncoding.DecodeString(header[cseIV])
	if err != nil {
		return nil, err
	}
	encryptedKey, err := base64.StdEncoding.DecodeString(header[cseKey])
	if err != nil {
		return nil, err
	}
	contentKey, err := c.key.Decrypt(encryptedKey)
	if err != nil {
		return nil, err
	}
	aesCipher, err := aes.NewCipher(contentKey)
	if err != nil {
		return nil, err
	}
	return &aesCbcPkcs5Reader{
		stream:    r,
		srcBuf:    bytes.NewBuffer(nil),
		dstBuf:    bytes.NewBuffer(nil),
		cryptMode: decryptMode,
		blockMode: cipher.NewCBCDecrypter(aesCipher, iv),
	}, nil
}

func (c aesCbcPkcs5) Overhead(size int64) int64 {
	return size + (aes.BlockSize - (size % aes.BlockSize))
}

// CBCSecureMaterials encrypts/decrypts data using AES CBC algorithm
type aesCbcPkcs5Reader struct {

	// Data stream to encrypt/decrypt
	stream io.Reader

	// Last internal error
	err error

	// End of file reached
	eof bool

	// Holds initial data
	srcBuf *bytes.Buffer

	// Holds transformed data (encrypted or decrypted)
	dstBuf *bytes.Buffer

	// Indicate if we are going to encrypt or decrypt
	cryptMode cryptMode

	// Helper that encrypts/decrypts data
	blockMode cipher.BlockMode
}

func (r *aesCbcPkcs5Reader) Close() error {
	if closer, ok := r.stream.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (r *aesCbcPkcs5Reader) Read(buf []byte) (n int, err error) {
	// Always fill buf from bufChunk at the end of this function
	defer func() {
		if r.err != nil {
			n, err = 0, r.err
		} else {
			n, err = r.dstBuf.Read(buf)
		}
	}()

	// Return
	if r.eof {
		return
	}

	// Fill dest buffer if its length is less than buf
	for !r.eof && r.dstBuf.Len() < len(buf) {

		srcPart := make([]byte, aes.BlockSize)
		dstPart := make([]byte, aes.BlockSize)

		// Fill src buffer
		for r.srcBuf.Len() < aes.BlockSize*2 {
			_, err = io.CopyN(r.srcBuf, r.stream, aes.BlockSize)
			if err != nil {
				break
			}
		}

		// Quit immediately for errors other than io.EOF
		if err != nil && err != io.EOF {
			r.err = err
			return
		}

		// Mark current encrypting/decrypting as finished
		r.eof = (err == io.EOF)

		if r.eof && r.cryptMode == encryptMode {
			if srcPart, err = pkcs5Pad(r.srcBuf.Bytes(), aes.BlockSize); err != nil {
				r.err = err
				return
			}
		} else {
			_, _ = r.srcBuf.Read(srcPart)
		}

		// Crypt srcPart content
		for len(srcPart) > 0 {

			// Crypt current part
			r.blockMode.CryptBlocks(dstPart, srcPart[:aes.BlockSize])

			// Unpad when this is the last part and we are decrypting
			if r.eof && r.cryptMode == decryptMode {
				dstPart, err = pkcs5Unpad(dstPart, aes.BlockSize)
				if err != nil {
					r.err = err
					return
				}
			}

			// Send crypted data to dstBuf
			if _, wErr := r.dstBuf.Write(dstPart); wErr != nil {
				r.err = wErr
				return
			}
			// Move to the next part
			srcPart = srcPart[aes.BlockSize:]
		}
	}
	return
}

type aesCbcKey []byte

// Encrypt passed bytes
func (s aesCbcKey) Encrypt(plain []byte) ([]byte, error) {
	// Initialize an AES encryptor using a master key
	keyBlock, err := aes.NewCipher(s[:])
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
func (s aesCbcKey) Decrypt(cipher []byte) ([]byte, error) {
	// Initialize AES decrypter
	keyBlock, err := aes.NewCipher(s[:])
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

// Unpad a set of bytes following PKCS5 algorithm
func pkcs5Unpad(buf []byte, blockSize int) ([]byte, error) {
	len := len(buf)
	if len == 0 {
		return nil, errors.New("buffer is empty")
	}
	pad := int(buf[len-1])
	if pad > len || pad > blockSize {
		return nil, errors.New("invalid padding size")
	}
	return buf[:len-pad], nil
}

// Pad a set of bytes following PKCS5 algorithm
func pkcs5Pad(buf []byte, blockSize int) ([]byte, error) {
	len := len(buf)
	pad := blockSize - (len % blockSize)
	padText := bytes.Repeat([]byte{byte(pad)}, pad)
	return append(buf, padText...), nil
}
