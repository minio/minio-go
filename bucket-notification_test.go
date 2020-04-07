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

package minio

import "testing"

func TestEqualNotificationEventTypeList(t *testing.T) {
	type args struct {
		a []NotificationEventType
		b []NotificationEventType
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "same order",
			args: args{
				a: []NotificationEventType{ObjectCreatedAll, ObjectAccessedAll},
				b: []NotificationEventType{ObjectCreatedAll, ObjectAccessedAll},
			},
			want: true,
		},
		{
			name: "different order",
			args: args{
				a: []NotificationEventType{ObjectCreatedAll, ObjectAccessedAll},
				b: []NotificationEventType{ObjectAccessedAll, ObjectCreatedAll},
			},
			want: true,
		},
		{
			name: "not equal",
			args: args{
				a: []NotificationEventType{ObjectCreatedAll, ObjectAccessedAll},
				b: []NotificationEventType{ObjectRemovedAll, ObjectAccessedAll},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EqualNotificationEventTypeList(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("EqualNotificationEventTypeList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEqualFilterRuleList(t *testing.T) {
	type args struct {
		a []FilterRule
		b []FilterRule
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "same order",
			args: args{
				a: []FilterRule{{Name: "prefix", Value: "prefix1"}, {Name: "suffix", Value: "suffix1"}},
				b: []FilterRule{{Name: "prefix", Value: "prefix1"}, {Name: "suffix", Value: "suffix1"}},
			},
			want: true,
		},
		{
			name: "different order",
			args: args{
				a: []FilterRule{{Name: "prefix", Value: "prefix1"}, {Name: "suffix", Value: "suffix1"}},
				b: []FilterRule{{Name: "suffix", Value: "suffix1"}, {Name: "prefix", Value: "prefix1"}},
			},
			want: true,
		},
		{
			name: "not equal",
			args: args{
				a: []FilterRule{{Name: "prefix", Value: "prefix1"}, {Name: "suffix", Value: "suffix1"}},
				b: []FilterRule{{Name: "prefix", Value: "prefix2"}, {Name: "suffix", Value: "suffix1"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EqualFilterRuleList(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("EqualFilterRuleList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotificationConfig_Equal(t *testing.T) {
	type fields struct {
		ID     string
		Arn    Arn
		Events []NotificationEventType
		Filter *Filter
	}
	type args struct {
		arn    string
		events []NotificationEventType
		prefix string
		suffix string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "same order",
			fields: fields{
				Arn:    NewArn("minio", "sqs", "", "1", "postgresql"),
				Events: []NotificationEventType{ObjectCreatedAll, ObjectAccessedAll},
				Filter: &Filter{
					S3Key: S3Key{
						FilterRules: []FilterRule{{Name: "prefix", Value: "prefix1"}, {Name: "suffix", Value: "suffix1"}},
					},
				},
			},
			args: args{
				arn:    "arn:minio:sqs::1:postgresql",
				events: []NotificationEventType{ObjectCreatedAll, ObjectAccessedAll},
				prefix: "prefix1",
				suffix: "suffix1",
			},
			want: true,
		},
		{
			name: "different order",
			fields: fields{
				Arn:    NewArn("minio", "sqs", "", "1", "postgresql"),
				Events: []NotificationEventType{ObjectAccessedAll, ObjectCreatedAll},
				Filter: &Filter{
					S3Key: S3Key{
						FilterRules: []FilterRule{{Name: "suffix", Value: "suffix1"}, {Name: "prefix", Value: "prefix1"}},
					},
				},
			},
			args: args{
				arn:    "arn:minio:sqs::1:postgresql",
				events: []NotificationEventType{ObjectCreatedAll, ObjectAccessedAll},
				prefix: "prefix1",
				suffix: "suffix1",
			},
			want: true,
		},
		{
			name: "not equal",
			fields: fields{
				Arn:    NewArn("minio", "sqs", "", "1", "postgresql"),
				Events: []NotificationEventType{ObjectAccessedAll},
				Filter: &Filter{
					S3Key: S3Key{
						FilterRules: []FilterRule{{Name: "suffix", Value: "suffix1"}, {Name: "prefix", Value: "prefix1"}},
					},
				},
			},
			args: args{
				arn:    "arn:minio:sqs::1:postgresql",
				events: []NotificationEventType{ObjectCreatedAll, ObjectAccessedAll},
				prefix: "prefix1",
				suffix: "suffix1",
			},
			want: false,
		},
		{
			name: "different arn",
			fields: fields{
				Arn:    NewArn("minio", "sqs", "", "2", "postgresql"),
				Events: []NotificationEventType{ObjectAccessedAll},
				Filter: &Filter{
					S3Key: S3Key{
						FilterRules: []FilterRule{{Name: "suffix", Value: "suffix1"}, {Name: "prefix", Value: "prefix1"}},
					},
				},
			},
			args: args{
				arn:    "arn:minio:sqs::1:postgresql",
				events: []NotificationEventType{ObjectCreatedAll, ObjectAccessedAll},
				prefix: "prefix1",
				suffix: "suffix1",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nc := &NotificationConfig{
				ID:     tt.fields.ID,
				Arn:    tt.fields.Arn,
				Events: tt.fields.Events,
				Filter: tt.fields.Filter,
			}
			if got := nc.Equal(tt.args.arn, tt.args.events, tt.args.prefix, tt.args.suffix); got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}
