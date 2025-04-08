package brsp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	secretsmanagerTypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"io"
)

type RestoreSecretsCommand struct {
	ssmClient            *ssm.Client
	s3Client             *s3.Client
	kmsClient            *kms.Client
	secretsmanagerClient *secretsmanager.Client
	opt                  *RestoreSecretsCommandOption
}

type RestoreSecretsCommandOption struct {
	BucketName        string `help:"S3 bucket name"`
	Key               string `help:"S3 object key"`
	WithDecryption    bool   `default:"false" help:"With decryption"`
	DataKeyBucketName string `help:"data key bucket name"`
	DataKeyKey        string `help:"data key key"`
	KmsKey            string `help:"KMS key for decryption"`
	DestinationSuffix string `help:"Destination suffix"`
	DryRun            bool   `default:"true" help:"Dry run"`
}

func NewRestoreSecretsCommand(opt *RestoreSecretsCommandOption) (*RestoreSecretsCommand, error) {
	awsConfig, err := getAwsConfig()
	if err != nil {
		return nil, err
	}
	return &RestoreSecretsCommand{
		ssmClient:            ssm.NewFromConfig(awsConfig),
		s3Client:             s3.NewFromConfig(awsConfig),
		kmsClient:            kms.NewFromConfig(awsConfig),
		secretsmanagerClient: secretsmanager.NewFromConfig(awsConfig),
		opt:                  opt,
	}, nil

}

func (c *RestoreSecretsCommand) Run() error {
	output, err := c.s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(c.opt.BucketName),
		Key:    aws.String(c.opt.Key),
	})
	if err != nil {
		return err
	}
	defer output.Body.Close()

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return err
	}

	getNonceObjectOutput, err := c.s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(c.opt.BucketName),
		Key:    aws.String(fmt.Sprintf("%s.nonce", c.opt.Key)),
	})
	if err != nil {
		return err
	}
	defer getNonceObjectOutput.Body.Close()

	nonce, err := io.ReadAll(getNonceObjectOutput.Body)
	if err != nil {
		return err
	}
	getObjectOutput, err := c.s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(c.opt.BucketName),
		Key:    aws.String(c.opt.DataKeyKey),
	})
	if err != nil {
		return err
	}

	dataKey, err := io.ReadAll(getObjectOutput.Body)
	if err != nil {
		return err
	}

	decryptOutput, err := c.kmsClient.Decrypt(context.TODO(), &kms.DecryptInput{
		CiphertextBlob: dataKey,
	})
	if err != nil {
		return err
	}

	decrypted, err := decryptData(decryptOutput.Plaintext, nonce, data)
	if err != nil {
		return err
	}

	var secrets []Secret
	err = json.Unmarshal(decrypted, &secrets)
	if err != nil {
		return err
	}

	for _, secret := range secrets {
		listSecretsOutput, err := c.secretsmanagerClient.ListSecrets(context.TODO(), &secretsmanager.ListSecretsInput{
			Filters: []secretsmanagerTypes.Filter{
				{
					Key:    secretsmanagerTypes.FilterNameStringTypeName,
					Values: []string{*secret.Name + "_"},
				},
			},
		})
		if err != nil {
			return err
		}
		if len(listSecretsOutput.SecretList) == 0 {
			fmt.Printf("Secret %s is not found\n", *secret.Name)
			continue
		}
		for _, s := range listSecretsOutput.SecretList {
			getSecretValueOutput, err := c.secretsmanagerClient.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{
				SecretId: s.ARN,
			})
			if err != nil {
				return err
			}

			if *getSecretValueOutput.SecretString != "DUMMY" {
				fmt.Printf("Skip secret %s because it is not DUMMY\n", *s.Name)
				continue
			}

			if c.opt.DryRun {
				fmt.Printf("[DRY RUN] Restore secret value to %s from %s\n", *s.Name, *secret.Name)
				continue
			}

			fmt.Printf("Restore secret value to %s from %s\n", *s.Name, *secret.Name)
			_, err = c.secretsmanagerClient.PutSecretValue(context.TODO(), &secretsmanager.PutSecretValueInput{
				SecretId:     s.ARN,
				SecretString: aws.String(secret.SecretValue),
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
