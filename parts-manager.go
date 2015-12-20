/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015 Minio, Inc.
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

package minio

import (
	"crypto/md5"
	"crypto/sha256"
	"hash"
	"io"
	"io/ioutil"
	"os"
)

// sectionManager reads from *os.File, partitions data into individual *partMetadata{}*.
//
// This method runs until an EOF or an error occurs. Before returning, the channel is always closed.
func sectionManager(fileData *os.File, fileSize, partSize int64, enableSha256Sum bool, doneCh <-chan struct{}) <-chan partMetadata {
	partMetadataCh := make(chan partMetadata, 3)
	go sectionManagerInRoutine(fileData, fileSize, partSize, enableSha256Sum, partMetadataCh, doneCh)
	return partMetadataCh
}

func sectionManagerInRoutine(fileData *os.File, fileSize, partSize int64, enableSha256Sum bool, partMetadataCh chan<- partMetadata, doneCh <-chan struct{}) {
	defer close(partMetadataCh)
	// MD5 and Sha256 hasher.
	var hashMD5, hashSha256 hash.Hash

	// totalRead counter
	var totalRead int64

	// Loop through until end of file.
	for totalRead <= fileSize {
		// Create a hash multiwriter.
		hashMD5 = md5.New()
		hashWriter := io.MultiWriter(hashMD5)
		if enableSha256Sum {
			hashSha256 = sha256.New()
			hashWriter = io.MultiWriter(hashMD5, hashSha256)
		}
		// Get a section reader on a particular offset.
		sectionReader := io.NewSectionReader(fileData, totalRead, partSize)
		size, err := io.Copy(hashWriter, sectionReader)
		if err != nil {
			partMetadataCh <- partMetadata{
				Err: err,
			}
			return
		}
		// Seek back to its primary location.
		if _, err := sectionReader.Seek(0, 0); err != nil {
			partMetadataCh <- partMetadata{
				Err: err,
			}
			return
		}
		partMdata := partMetadata{
			MD5Sum:     hashMD5.Sum(nil),
			ReadCloser: ioutil.NopCloser(sectionReader),
			Size:       size,
			Err:        nil,
		}
		if enableSha256Sum {
			partMdata.Sha256Sum = hashSha256.Sum(nil)
		}
		select {
		// Reply with new partMetadata.
		case partMetadataCh <- partMdata:
			totalRead += partSize
		// If done channel receives prematurely, return here to close the internal channel.
		case <-doneCh:
			return
		}
	}
}

// partsManager reads from io.Reader, partitions data into individual *partMetadata{}*.
// backed by a temporary file which purges itself upon Close().
//
// This method runs until an EOF or an error occurs. Before returning, the channel is always closed.
func partsManager(reader io.Reader, partSize int64, enableSha256Sum bool) <-chan partMetadata {
	ch := make(chan partMetadata, 3)
	go partsManagerInRoutine(reader, partSize, enableSha256Sum, ch)
	return ch
}

func partsManagerInRoutine(reader io.Reader, partSize int64, enableSha256Sum bool, ch chan<- partMetadata) {
	defer close(ch)
	// Any error generated when creating parts.
	var err error
	// Size of the each part read, could be shorter than partSize.
	var size int64
	// Tempfile structure backed by Closer to clean itself up.
	var tmpFile *tempFile
	// MD5 and Sha256 hasher.
	var hashMD5, hashSha256 hash.Hash
	// Collective multi writer.
	var writer io.Writer
	for {
		tmpFile, err = newTempFile("multiparts$")
		if err != nil {
			break
		}
		// Create a hash multiwriter.
		hashMD5 = md5.New()
		hashWriter := io.MultiWriter(hashMD5)
		if enableSha256Sum {
			hashSha256 = sha256.New()
			hashWriter = io.MultiWriter(hashMD5, hashSha256)
		}
		writer = io.MultiWriter(tmpFile, hashWriter)
		size, err = io.CopyN(writer, reader, partSize)
		if err != nil {
			break
		}
		// Seek back to beginning.
		tmpFile.Seek(0, 0)
		partMdata := partMetadata{
			MD5Sum:     hashMD5.Sum(nil),
			ReadCloser: tmpFile,
			Size:       size,
			Err:        nil,
		}
		if enableSha256Sum {
			partMdata.Sha256Sum = hashSha256.Sum(nil)
		}
		ch <- partMdata
	}
	// If end of file reached, we send the last part.
	if err == io.EOF {
		// Seek back to beginning.
		tmpFile.Seek(0, 0)

		// last part.
		partMdata := partMetadata{
			MD5Sum:     hashMD5.Sum(nil),
			ReadCloser: tmpFile,
			Size:       size,
			Err:        nil,
		}
		if enableSha256Sum {
			partMdata.Sha256Sum = hashSha256.Sum(nil)
		}
		ch <- partMdata
		return
	}
	if err != io.EOF {
		ch <- partMetadata{
			Err: err,
		}
		return
	}
}
