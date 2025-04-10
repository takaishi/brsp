# brsp

## Install

```bash
go install github.com/takaishi/brsp/cmd/brsp
```

## Usage

Backup all parameters in specified region (REGION) to specified S3 bucket in target region (TARGET_REGION).

```
% AWS_REGION=${REGION} ./dist/aws_secret_backuper backup-parameters --target-region ${TARGET_REGION} --bucket-name ${BUCKET_NAME} --key ${KEY} --with-encryption --encryption-kms-key ${KMS_KEY_ID}
```

Backup all secrets in specified region (REGION) to specified S3 bucket in target region (TARGET_REGION).

```
% AWS_REGION=${REGION} ./dist/aws_secret_backuper backup-secrets --target-region ${TARGET_REGION}  --bucket-name ${BUCKET_NAME} --key ${KEY} --with-encryption --encryption-kms-key ${KMS_KEY_ID}
```

Download and print backup to stdout in specified region (REGION).

```
% AWS_REGION=${REGION} ./dist/aws_secret_backuper download-backup --bucket-name ${BUCKET_NAME} --key ${KEY} --with-decryption --decryption-kms-key ${KMS_KEY_ID}
```


## Development

```
# python3 -m venv .venv
# source .venv/bin/activate
```

```
# docker compose up
```

```
% awslocal kms list-keys --region ap-northeast-3 --query 'Keys[0].KeyId' --output text
00000000-0000-0000-0000-000000000001
```
