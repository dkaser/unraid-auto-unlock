package awssecrets

/*
	autounlock - Unraid Auto Unlock
	Copyright (C) 2025-2026 Derek Kaser

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"strings"
	"testing"
)

func TestSecretsManagerFetcher_Match(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "valid secrets manager path",
			path: "aws-secrets://key:secret@us-east-1/my-secret",
			want: true,
		},
		{
			name: "valid secrets manager path with complex secret",
			path: "aws-secrets://AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY@us-west-2/prod/database/password",
			want: true,
		},
		{
			name: "ssm path should not match",
			path: "aws-ssm://key:secret@us-east-1/my-param",
			want: false,
		},
		{
			name: "empty path",
			path: "",
			want: false,
		},
		{
			name: "file path",
			path: "file:///path/to/secret",
			want: false,
		},
	}

	f := &SecretsManagerFetcher{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.Match(tt.path); got != tt.want {
				t.Errorf("SecretsManagerFetcher.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecretsManagerFetcher_Priority(t *testing.T) {
	f := &SecretsManagerFetcher{}
	if got := f.Priority(); got != PriorityAWS {
		t.Errorf("SecretsManagerFetcher.Priority() = %v, want %v", got, PriorityAWS)
	}
}

func TestSSMFetcher_Match(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "valid ssm path",
			path: "aws-ssm://key:secret@us-east-1/my-param",
			want: true,
		},
		{
			name: "valid ssm path with nested parameter",
			path: "aws-ssm://AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY@us-west-2/prod/database/connection",
			want: true,
		},
		{
			name: "secrets manager path should not match",
			path: "aws-secrets://key:secret@us-east-1/my-secret",
			want: false,
		},
		{
			name: "empty path",
			path: "",
			want: false,
		},
		{
			name: "file path",
			path: "file:///path/to/secret",
			want: false,
		},
	}

	f := &SSMFetcher{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.Match(tt.path); got != tt.want {
				t.Errorf("SSMFetcher.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSSMFetcher_Priority(t *testing.T) {
	f := &SSMFetcher{}
	if got := f.Priority(); got != PriorityAWS {
		t.Errorf("SSMFetcher.Priority() = %v, want %v", got, PriorityAWS)
	}
}

func TestParseAWSPath(t *testing.T) { //nolint:funlen // Length due to multiple test cases
	tests := []struct {
		name            string
		path            string
		prefix          string
		wantRegion      string
		wantResource    string
		wantErr         bool
		wantErrContains string
	}{
		{
			name:         "valid secrets manager path",
			path:         "aws-secrets://AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMI@us-east-1/my-secret",
			prefix:       "aws-secrets://",
			wantRegion:   "us-east-1",
			wantResource: "my-secret",
			wantErr:      false,
		},
		{
			name:         "valid ssm path",
			path:         "aws-ssm://AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMI@us-west-2/my-parameter",
			prefix:       "aws-ssm://",
			wantRegion:   "us-west-2",
			wantResource: "my-parameter",
			wantErr:      false,
		},
		{
			name:         "secret key with slash",
			path:         "aws-secrets://AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY@us-east-1/my-secret",
			prefix:       "aws-secrets://",
			wantRegion:   "us-east-1",
			wantResource: "my-secret",
			wantErr:      false,
		},
		{
			name:         "nested resource path",
			path:         "aws-secrets://AKIAIOSFODNN7EXAMPLE:secret@us-east-1/prod/database/password",
			prefix:       "aws-secrets://",
			wantRegion:   "us-east-1",
			wantResource: "prod/database/password",
			wantErr:      false,
		},
		{
			name:            "missing credentials",
			path:            "aws-secrets://us-east-1/my-secret",
			prefix:          "aws-secrets://",
			wantErr:         true,
			wantErrContains: "invalid path format",
		},
		{
			name:            "missing region",
			path:            "aws-secrets://AKIAIOSFODNN7EXAMPLE:secret@/my-secret",
			prefix:          "aws-secrets://",
			wantErr:         true,
			wantErrContains: "invalid path format",
		},
		{
			name:            "missing resource",
			path:            "aws-secrets://AKIAIOSFODNN7EXAMPLE:secret@us-east-1/",
			prefix:          "aws-secrets://",
			wantErr:         true,
			wantErrContains: "invalid path format",
		},
		{
			name:            "missing secret key",
			path:            "aws-secrets://AKIAIOSFODNN7EXAMPLE:@us-east-1/my-secret",
			prefix:          "aws-secrets://",
			wantErr:         true,
			wantErrContains: "invalid path format",
		},
		{
			name:            "missing access key",
			path:            "aws-secrets://:secret@us-east-1/my-secret",
			prefix:          "aws-secrets://",
			wantErr:         true,
			wantErrContains: "invalid path format",
		},
		{
			name:            "no resource path",
			path:            "aws-secrets://AKIAIOSFODNN7EXAMPLE:secret@us-east-1",
			prefix:          "aws-secrets://",
			wantErr:         true,
			wantErrContains: "invalid path format",
		},
		{
			name:            "malformed - no @",
			path:            "aws-secrets://AKIAIOSFODNN7EXAMPLE:secret-us-east-1/my-secret",
			prefix:          "aws-secrets://",
			wantErr:         true,
			wantErrContains: "invalid path format",
		},
		{
			name:            "malformed - no colon",
			path:            "aws-secrets://AKIAIOSFODNN7EXAMPLE@us-east-1/my-secret",
			prefix:          "aws-secrets://",
			wantErr:         true,
			wantErrContains: "invalid path format",
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, region, resource, err := parseAWSPath(ctx, tt.path, tt.prefix)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseAWSPath() error = nil, wantErr %v", tt.wantErr)

					return
				}

				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf(
						"parseAWSPath() error = %v, want error containing %v",
						err,
						tt.wantErrContains,
					)
				}

				return
			}

			if err != nil {
				t.Errorf("parseAWSPath() unexpected error = %v", err)

				return
			}

			if region != tt.wantRegion {
				t.Errorf("parseAWSPath() region = %v, want %v", region, tt.wantRegion)
			}

			if resource != tt.wantResource {
				t.Errorf("parseAWSPath() resource = %v, want %v", resource, tt.wantResource)
			}

			// Verify config was created
			if cfg.Region != tt.wantRegion {
				t.Errorf("parseAWSPath() cfg.Region = %v, want %v", cfg.Region, tt.wantRegion)
			}
		})
	}
}
