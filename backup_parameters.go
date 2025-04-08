package brsp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmTypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"strings"
)

type BackupParametersCommand struct {
	ssmClient *ssm.Client
	s3Client  *s3.Client
	kmsClient *kms.Client
	opt       *BackupParametersCommandOption
}

type BackupParametersCommandOption struct {
	ParameterName     string `help:"parameter name"`
	TargetRegion      string `help:"target region"`
	BucketName        string `help:"bucket name"`
	Key               string `help:"key"`
	DataKeyBucketName string `help:"data key bucket name"`
	DataKeyKey        string `help:"data key key"`
	KmsKey            string `help:"KMS key to decrypt data key"`
}

type Parameter struct {
	*ssmTypes.Parameter
	KmsKey string
}

func NewBackupParametersCommand(opt *BackupParametersCommandOption) (*BackupParametersCommand, error) {
	awsConfig, err := getAwsConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get aws config, %v", err)
	}
	targetAwsConfig, err := getTargetAwsConfig(opt.TargetRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to get target aws config, %v", err)
	}
	return &BackupParametersCommand{
		s3Client:  s3.NewFromConfig(targetAwsConfig),
		ssmClient: ssm.NewFromConfig(awsConfig),
		kmsClient: kms.NewFromConfig(targetAwsConfig),
		opt:       opt,
	}, nil
}

func (c *BackupParametersCommand) Run() error {
	paginator := ssm.NewDescribeParametersPaginator(c.ssmClient, &ssm.DescribeParametersInput{})
	chunkSize := 10
	parameterNames := []string{}
	parameters := []Parameter{}

	if c.opt.ParameterName != "" {
		chunk := []string{c.opt.ParameterName}
		getParametersWithoutDecruptionOutput, err := c.ssmClient.GetParameters(context.TODO(), &ssm.GetParametersInput{
			Names:          chunk,
			WithDecryption: aws.Bool(false),
		})
		if err != nil {
			return err
		}
		getParametersOutput, err := c.ssmClient.GetParameters(context.TODO(), &ssm.GetParametersInput{
			Names:          chunk,
			WithDecryption: aws.Bool(true),
		})
		if err != nil {
			return err
		}
		for i, parameter := range getParametersOutput.Parameters {
			parameters = append(parameters, Parameter{Parameter: &parameter, KmsKey: *getParametersWithoutDecruptionOutput.Parameters[i].Value})
		}
	} else {
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(context.TODO())
			if err != nil {
				return err
			}
			for _, parameter := range page.Parameters {
				parameterNames = append(parameterNames, *parameter.Name)
			}
		}

		for i := 0; i < len(parameterNames); i += chunkSize {
			end := i + chunkSize
			if end > len(parameterNames) {
				end = len(parameterNames)
			}
			chunk := parameterNames[i:end]
			getParametersWithoutDecruptionOutput, err := c.ssmClient.GetParameters(context.TODO(), &ssm.GetParametersInput{
				Names:          chunk,
				WithDecryption: aws.Bool(false),
			})
			if err != nil {
				return err
			}
			getParametersOutput, err := c.ssmClient.GetParameters(context.TODO(), &ssm.GetParametersInput{
				Names:          chunk,
				WithDecryption: aws.Bool(true),
			})
			if err != nil {
				return err
			}
			for i, parameter := range getParametersOutput.Parameters {
				parameters = append(parameters, Parameter{Parameter: &parameter, KmsKey: *getParametersWithoutDecruptionOutput.Parameters[i].Value})
			}
		}
	}

	body, err := json.Marshal(parameters)
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
