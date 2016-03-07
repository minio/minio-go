/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015, 2016 Minio, Inc.
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
	"math"
	"net"
	"time"
)

// MaxRetry is the maximum number of retries before stopping.
var MaxRetry = 3

// newRetryTimer creates a timer with binomially increasing delays
// until the maximum retry attempts are reached.
func newRetryTimer(maxRetry int, unit time.Duration) <-chan int {
	attemptCh := make(chan int)
	go func() {
		defer close(attemptCh)
		for i := 0; i < maxRetry; i++ {
			// Grow the interval at a binomial rate.
			time.Sleep(time.Second * time.Duration(math.Pow(2, float64(i))))
			attemptCh <- i + 1 // Attempts start from 1.
		}
	}()
	return attemptCh
}

// isNetErrorRetryable - is network error retryable.
func isNetErrorRetryable(err error) bool {
	switch err.(type) {
	case *net.DNSError, *net.OpError, net.UnknownNetworkError:
		return true
	}
	return false
}

// isS3CodeRetryable - is s3 error code retryable.
func isS3CodeRetryable(s3Code string) bool {
	switch s3Code {
	case "RequestError", "RequestTimeout", "Throttling", "ThrottlingException":
		fallthrough
	case "RequestLimitExceeded", "RequestThrottled", "InternalError":
		fallthrough
	case "ExpiredToken", "ExpiredTokenException":
		return true
	}
	return false
}
