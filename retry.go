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
	"time"
)

// MaxRetries is the maximum number of retries before stopping.
var MaxRetries = 3

// backOff is a time.Duration counter. It starts at Start.
// After every call to Duration() it is multiplied by Factor.
// It returns to Start on every call to Reset().
type binomialDuration struct {
	// Factor is the multiplying factor for each increment step
	attempts, Factor float64
	// Starting delay for 1st attempt.
	Start time.Duration
}

// Duration - Returns the current value of the counter and then
// multiplies it Factor
func (b *binomialDuration) Duration() time.Duration {
	d := b.forEachAttempt(b.attempts)
	b.attempts++
	return d
}

// forEachAttempt returns the duration for a specific attempt.
func (b *binomialDuration) forEachAttempt(attempt float64) time.Duration {
	// Starts from 1 sec for 1st attempt to infinity depending on the
	// attempts.
	if b.Start == 0 {
		b.Start = 1 * time.Second
	}
	if b.Factor == 0 {
		b.Factor = 2
	}
	// calculate this duration
	duration := float64(b.Start) * math.Pow(b.Factor, attempt)
	// return as time.Duration
	return time.Duration(duration)
}

// Reset the current value of the counter back to Min
func (b *binomialDuration) Reset() {
	b.attempts = 0
}

// Func represents functions that can be retried.
type Func func(attempt int) (retry bool, err error)

// retry keeps trying the function until the second argument
// returns false, or no error is returned.
func retry(fn Func) error {
	var err error
	var cont bool
	attempt := 1
	for {
		cont, err = fn(attempt)
		if !cont || err == nil {
			break
		}
		attempt++
		if attempt > MaxRetries {
			return err
		}
	}
	return err
}
