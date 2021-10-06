/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2017 MinIO, Inc.
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

package utils

import (
	"context"
	"math/rand"
	"time"
)

// NewRetryTimer creates a timer with exponentially increasing delays until the maximum retry attempts are reached.
func NewRetryTimer(ctx context.Context, maxRetry int, unit time.Duration, cap time.Duration, jitter, maxJitter float64) <-chan int {

	attemptCh := make(chan int)

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	// computes the exponential backoff duration according to
	// https://www.awsarchitectureblog.com/2015/03/backoff.html
	exponentialBackoffWait := func(attempt int) time.Duration {
		// normalize jitter to the range [0, 1.0]
		if jitter < 0.0 {
			jitter = 0.0
		}
		if jitter > maxJitter {
			jitter = maxJitter
		}

		//sleep = random_between(0, min(cap, base * 2 ** attempt))
		sleep := unit * time.Duration(1<<uint(attempt))
		if sleep > cap {
			sleep = cap
		}
		if jitter != 0.0 {
			sleep -= time.Duration(rnd.Float64() * float64(sleep) * jitter)
		}
		return sleep
	}

	go func() {
		defer close(attemptCh)
		for i := 0; i < maxRetry; i++ {
			select {
			case attemptCh <- i + 1:
			case <-ctx.Done():
				return
			}

			select {
			case <-time.After(exponentialBackoffWait(i)):
			case <-ctx.Done():
				return
			}
		}
	}()
	return attemptCh
}
