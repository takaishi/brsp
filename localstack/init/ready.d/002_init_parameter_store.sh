#!/bin/bash

awslocal ssm put-parameter --name /dev/parameter_1 --type=String       --value "non_secret_value"  --region ap-northeast-1
awslocal ssm put-parameter --name /dev/parameter_2 --type=SecureString --value "secret_value"  --region ap-northeast-1

awslocal ssm describe-parameters --region ap-northeast-1
