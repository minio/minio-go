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
	"errors"
	"io"
	"os"
	"path/filepath"
)

// FPutObject - put object a file.
func (c Client) FPutObject(bucketName, objectName, fileName string) (string, error) {
	return "", errors.New("Not implemented yet")
}

// FGetObject - get object to a file.
func (c Client) FGetObject(bucketName, objectName, fileName string) error {
	// Verify if destination already exists.
	st, err := os.Stat(fileName)
	if err == nil {
		// If the destination exists and is a directory.
		if st.IsDir() {
			return ErrInvalidArgument("fileName is a directory.")
		}
	}

	// Proceed if file does not exist. return for all other errors.
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	// Extract dir name.
	objectDir, _ := filepath.Split(fileName)
	if objectDir != "" {
		// Create any missing top level directories.
		if err := os.MkdirAll(objectDir, 0700); err != nil {
			return err
		}
	}

	// Write to a temporary file "fileName.part.minio-go" before saving.
	filePartPath := fileName + ".part.minio-go"

	// If exists, open in append mode. If not create it as a part file.
	filePart, err := os.OpenFile(filePartPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	// Issue Stat to get the current offset.
	st, err = filePart.Stat()
	if err != nil {
		return err
	}

	// Seek to current position for incoming reader.
	objectReader, objectStat, err := c.getObject(bucketName, objectName, st.Size(), 0)
	if err != nil {
		return err
	}

	// Write to the part file.
	if _, err = io.CopyN(filePart, objectReader, objectStat.Size); err != nil {
		return err
	}

	// Close the file before rename.
	filePart.Close()

	// Safely completed. Now commit by renaming to actual filename.
	if err = os.Rename(filePartPath, fileName); err != nil {
		return err
	}

	// Return.
	return nil
}
