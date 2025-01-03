# Terraform Deployment Guide

This guide explains how to deploy the infrastructure using Terraform. The infrastructure code is located in the [terraform](../terraform)
folder and sets up an AWS environment with Lambda functions and API Gateway.

## Infrastructure Components

The Terraform configuration deploys the following AWS resources:

- **AWS Lambda Functions**: Serverless functions that handle the application logic
- **API Gateway**: REST API endpoints that trigger the Lambda functions
- **IAM Roles and Policies**: Required permissions for Lambda execution
- **CloudWatch Log Groups**: For Lambda function logging

## Prerequisites

Before deploying, ensure you have:

1. [Terraform](https://www.terraform.io/downloads.html) installed (version 1.9.0 or later)
2. AWS CLI configured with appropriate credentials
3. Access to an AWS account with necessary permissions
4. Datadog API key, API URL, and App Key

** AWS and Datadog credentials are suposed to be set in the environment variables. **

However, you can modify the [providers.tf](../terraform/providers.tf) file and set the credentials there.

## Deployment Steps

1. Initialize Terraform:
   ```bash
   cd terraform
   terraform init
   ```

2. Review the planned changes:
   ```bash
   terraform plan
   ```

3. Apply the infrastructure changes:
   ```bash
   terraform apply
   ```
   When prompted, type `yes` to confirm the deployment.

## Verification

After deployment:

1. Check the AWS Console to verify all resources are created
2. Test the API endpoints using the provided API Gateway URL
3. Monitor CloudWatch logs for Lambda function execution

## Cleanup

To remove all deployed resources:

```bash
terraform destroy
```

## Troubleshooting

Common issues and solutions:

1. **Deployment Failures**
   - Verify AWS credentials are correct
   - Check IAM permissions
   - Review CloudWatch logs for errors

2. **API Gateway Issues**
   - Verify Lambda integration settings
   - Check CORS configuration
   - Test endpoints with proper request format

3. **Lambda Function Errors**
   - Review function permissions
   - Check CloudWatch logs for execution errors
   - Verify environment variables
