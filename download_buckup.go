package brsp

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"io"
)

type DownloadBackupCommand struct {
	ssmClient *ssm.Client
	s3Client  *s3.Client
	kmsClient *kms.Client
	opt       *DownloadBackupCommandOption
}

type DownloadBackupCommandOption struct {
	BucketName        string `help:"S3 bucket name"`
	Key               string `help:"S3 object key"`
	WithDecryption    bool   `default:"false" help:"With decryption"`
	DataKeyBucketName string `help:"data key bucket name"`
	DataKeyKey        string `help:"data key key"`
	KmsKey            string `help:"KMS key for decryption"`
}

func NewDownloadBackupCommand(opt *DownloadBackupCommandOption) (*DownloadBackupCommand, error) {
	awsConfig, err := getAwsConfig()
	if err != nil {
		return nil, err
	}
	return &DownloadBackupCommand{
		ssmClient: ssm.NewFromConfig(awsConfig),
		s3Client:  s3.NewFromConfig(awsConfig),
		kmsClient: kms.NewFromConfig(awsConfig),
		opt:       opt,
	}, nil

}

func (c *DownloadBackupCommand) Run() error {
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

	fmt.Println(string(decrypted))

	return nil
}
