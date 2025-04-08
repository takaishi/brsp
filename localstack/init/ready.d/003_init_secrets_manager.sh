#!/bin/bash

keyId=$(awslocal kms list-keys --region ap-northeast-1 --query 'Keys[0].KeyId' --output text)
awslocal secretsmanager create-secret --name /dev/secret_1 --secret-string "secret_value_1" --kms-key-id $keyId --region ap-northeast-1
awslocal secretsmanager create-secret --name /dev/secret_2 --secret-string "secret_value_2" --kms-key-id $keyId --region ap-northeast-1
