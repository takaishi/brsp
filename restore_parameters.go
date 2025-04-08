package brsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type RestoreParametersCommand struct {
	ssmClient            *ssm.Client
	s3Client             *s3.Client
	kmsClient            *kms.Client
	secretsmanagerClient *secretsmanager.Client
	opt                  *RestoreParametersCommandOption
}

type RestoreParametersCommandOption struct {
	BucketName        string `help:"S3 bucket name"`
	Key               string `help:"S3 object key"`
	WithDecryption    bool   `default:"false" help:"With decryption"`
	DataKeyBucketName string `help:"data key bucket name"`
	DataKeyKey        string `help:"data key key"`
	KmsKey            string `help:"KMS key for decryption"`
	DestinationSuffix string `help:"Destination suffix"`
	DryRun            bool   `default:"true" help:"Dry run"`
}

func NewRestoreParametersCommand(opt *RestoreParametersCommandOption) (*RestoreParametersCommand, error) {
	awsConfig, err := getAwsConfig()
	if err != nil {
		return nil, err
	}
	return &RestoreParametersCommand{
		ssmClient:            ssm.NewFromConfig(awsConfig),
		s3Client:             s3.NewFromConfig(awsConfig),
		kmsClient:            kms.NewFromConfig(awsConfig),
		secretsmanagerClient: secretsmanager.NewFromConfig(awsConfig),
		opt:                  opt,
	}, nil

}

func (c *RestoreParametersCommand) Run() error {
	fmt.Println("Restoring parameters")
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

	var parameters []Parameter
	err = json.Unmarshal(decrypted, &parameters)
	if err != nil {
		return err
	}

	for _, parameter := range parameters {
		fmt.Printf("Restoring parameter %s\n", *parameter.Name)
		getParametersOutput, err := c.ssmClient.GetParameters(context.TODO(), &ssm.GetParametersInput{
			Names: []string{*parameter.Name},
		})
		if err != nil {
			return err
		}
		if len(getParametersOutput.Parameters) == 0 {
			fmt.Printf("Parameter %s not found\n", *parameter.Name)
			continue
		}
		for _, p := range getParametersOutput.Parameters {
			getParametersOutput, err := c.ssmClient.GetParameter(context.TODO(), &ssm.GetParameterInput{
				Name: p.Name,
			})
			if err != nil {
				return err
			}
			if *getParametersOutput.Parameter.Value != "DUMMY" {
				fmt.Printf("Skip restoring parameter %s because it's not a dummy value\n", *parameter.Name)
				continue
			}

			if c.opt.DryRun {
				fmt.Printf("[DRY RUN] Restore parameter value to %s from %s\n", *p.Name, *parameter.Name)
				continue
			}

			fmt.Printf("Restore parameter value to %s from %s\n", *p.Name, *parameter.Name)
			err = retry(3, 2*time.Second, func() error {
				_, err = c.ssmClient.PutParameter(context.TODO(), &ssm.PutParameterInput{
					Name:      p.Name,
					Value:     parameter.Value,
					Overwrite: aws.Bool(true),
				})
				return err
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func retry(attempts int, sleep time.Duration, fn func() error) error {
	for i := 0; i < attempts; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		fmt.Printf("Attempt %d failed; retrying in %v\n", i+1, sleep)
		time.Sleep(sleep)
	}
	return fmt.Errorf("all attempts failed")
}
