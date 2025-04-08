package brsp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	secretmanagerTypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"strings"
)

type BackupSecretsCommand struct {
	s3Client             *s3.Client
	secretsmanagerClient *secretsmanager.Client
	kmsClient            *kms.Client
	opt                  *BackupSecretsCommandOption
}

type BackupSecretsCommandOption struct {
	TargetRegion      string `help:"target region"`
	BucketName        string `help:"bucket name"`
	Key               string `help:"key"`
	DataKeyBucketName string `help:"data key bucket name"`
	DataKeyKey        string `help:"data key key"`
	KmsKey            string `help:"KMS key for encryption"`
}

type Secret struct {
	*secretmanagerTypes.SecretListEntry
	SecretValue string
}

func NewBackupSecretsCommand(opt *BackupSecretsCommandOption) (*BackupSecretsCommand, error) {
	awsConfig, err := getAwsConfig()
	if err != nil {
		return nil, err
	}
	targetAwsConfig, err := getTargetAwsConfig(opt.TargetRegion)
	if err != nil {
		return nil, err
	}
	return &BackupSecretsCommand{
		secretsmanagerClient: secretsmanager.NewFromConfig(awsConfig),
		s3Client:             s3.NewFromConfig(targetAwsConfig),
		kmsClient:            kms.NewFromConfig(targetAwsConfig),
		opt:                  opt,
	}, nil
}

func (c *BackupSecretsCommand) Run() error {
	paginator := secretsmanager.NewListSecretsPaginator(c.secretsmanagerClient, &secretsmanager.ListSecretsInput{})
	secrets := []Secret{}

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, secret := range page.SecretList {
			getSecretValueOutput, err := c.secretsmanagerClient.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{
				SecretId: aws.String(*secret.Name),
			})
			if err != nil {
				return err
			}
			secrets = append(secrets, Secret{
				SecretListEntry: &secret,
				SecretValue:     *getSecretValueOutput.SecretString,
			})
		}
	}

	body, err := json.Marshal(secrets)
	if err != nil {
		return err
	}

	dataKey, err := getDataKey(context.TODO(), c.kmsClient, c.s3Client, c.opt.DataKeyBucketName, c.opt.DataKeyKey)
	if err != nil {
		return err
	}

	ciphertext, nonce, err := encryptData(dataKey, body)
	if err != nil {
		return err
	}

	_, err = c.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(c.opt.BucketName),
		Key:    aws.String(c.opt.Key),
		Body:   strings.NewReader(string(ciphertext)),
	})
	if err != nil {
		return err
	}

	_, err = c.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(c.opt.BucketName),
		Key:    aws.String(fmt.Sprintf("%s.nonce", c.opt.Key)),
		Body:   strings.NewReader(string(nonce)),
	})
	if err != nil {
		return err
	}

	return nil
}
