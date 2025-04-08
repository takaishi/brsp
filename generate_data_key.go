package brsp

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"strings"
)

type GenerateDataKeyCommand struct {
	ssmClient *ssm.Client
	s3Client  *s3.Client
	kmsClient *kms.Client
	opt       *GenerateDataKeyCommandOption
}

type GenerateDataKeyCommandOption struct {
	TargetRegion     string `help:"target region"`
	BucketName       string `help:"bucket name"`
	Key              string `help:"key"`
	EncryptionKmsKey string `help:"KMS key for encryption"`
}

func NewGenerateDataKeyCommand(opt *GenerateDataKeyCommandOption) (*GenerateDataKeyCommand, error) {
	awsConfig, err := getAwsConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get aws config, %v", err)
	}
	targetAwsConfig, err := getTargetAwsConfig(opt.TargetRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to get target aws config, %v", err)
	}
	fmt.Println("region: ", targetAwsConfig.Region)
	return &GenerateDataKeyCommand{
		s3Client:  s3.NewFromConfig(targetAwsConfig),
		ssmClient: ssm.NewFromConfig(awsConfig),
		kmsClient: kms.NewFromConfig(targetAwsConfig),
		opt:       opt,
	}, nil
}

func (c *GenerateDataKeyCommand) Run() error {
	fmt.Printf("BucketName: %s\n", c.opt.BucketName)
	generateDataKeyInput := &kms.GenerateDataKeyInput{
		KeyId:   aws.String(c.opt.EncryptionKmsKey),
		KeySpec: kmsTypes.DataKeySpecAes256,
	}
	result, err := c.kmsClient.GenerateDataKey(context.TODO(), generateDataKeyInput)
	if err != nil {
		return err
	}

	_, err = c.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:  aws.String(c.opt.BucketName),
		Key:     aws.String(c.opt.Key),
		Body:    strings.NewReader(string(result.CiphertextBlob)),
		Tagging: aws.String(fmt.Sprintf("KeyId=%s", *result.KeyId)),
	})
	if err != nil {
		return err
	}

	return nil
}
