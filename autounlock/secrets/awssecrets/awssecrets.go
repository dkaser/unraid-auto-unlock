package awssecrets

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/dkaser/unraid-auto-unlock/autounlock/secrets/registry"
)

const PriorityAWS = 25

func init() {
	registry.Register(&SecretsManagerFetcher{})
	registry.Register(&SSMFetcher{})
}

// SecretsManagerFetcher handles AWS Secrets Manager.
type SecretsManagerFetcher struct{}

func (f *SecretsManagerFetcher) Match(path string) bool {
	return strings.HasPrefix(path, "aws-secrets://")
}

func (f *SecretsManagerFetcher) Priority() int {
	return PriorityAWS
}

func (f *SecretsManagerFetcher) Fetch(ctx context.Context, path string) (string, error) {
	cfg, region, secretName, err := parseAWSPath(ctx, path, "aws-secrets://")
	if err != nil {
		return "", err
	}

	if region == "" {
		return "", errors.New(
			"region is required in path: aws-secrets://access_key:secret_key@region/secret-name",
		)
	}

	if secretName == "" {
		return "", errors.New("secret name is required in path")
	}

	client := secretsmanager.NewFromConfig(cfg)

	result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get secret: %w", err)
	}

	if result.SecretString != nil {
		return strings.TrimSpace(*result.SecretString), nil
	}

	return "", errors.New("secret contains binary data, not string")
}

// SSMFetcher handles AWS Systems Manager Parameter Store.
type SSMFetcher struct{}

func (f *SSMFetcher) Match(path string) bool {
	return strings.HasPrefix(path, "aws-ssm://")
}

func (f *SSMFetcher) Priority() int {
	return PriorityAWS
}

func (f *SSMFetcher) Fetch(ctx context.Context, path string) (string, error) {
	cfg, region, paramName, err := parseAWSPath(ctx, path, "aws-ssm://")
	if err != nil {
		return "", err
	}

	if region == "" {
		return "", errors.New(
			"region is required in path: aws-ssm://access_key:secret_key@region/parameter-name",
		)
	}

	if paramName == "" {
		return "", errors.New("parameter name is required in path")
	}

	client := ssm.NewFromConfig(cfg)

	result, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(paramName),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get parameter: %w", err)
	}

	return strings.TrimSpace(*result.Parameter.Value), nil
}

// parseAWSPath parses AWS URL format:  aws-secrets://access_key:secret_key@region/path
// Credentials are REQUIRED.
func parseAWSPath(
	ctx context.Context,
	path string,
	prefix string,
) (aws.Config, string, string, error) {
	path = strings.TrimPrefix(path, prefix)

	// Regex: ^([^:]+):([^@]+)@([^/]+)/(.+)$
	//   1: access key
	//   2: secret key (may contain /)
	//   3: region
	//   4: resource name (no leading slash)
	re := regexp.MustCompile(`^([^:]+):([^@]+)@([^/]+)/(.+)$`)

	matches := re.FindStringSubmatch(path)
	if matches == nil || len(matches) != 5 {
		return aws.Config{}, "", "", fmt.Errorf(
			"invalid path format: expected %saccess_key:secret_key@region/resource",
			prefix,
		)
	}

	accessKey := matches[1]
	secretKey := matches[2]
	region := matches[3]
	resourceName := matches[4]

	if accessKey == "" || secretKey == "" || region == "" || resourceName == "" {
		return aws.Config{}, "", "", errors.New(
			"all fields (access key, secret key, region, resource name) are required in path",
		)
	}

	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		return aws.Config{}, "", "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	return cfg, region, resourceName, nil
}
