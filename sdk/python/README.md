# AWS LoadBalancer Controller Pulumi Package

This repo is a [Pulumi Package](https://www.pulumi.com/docs/guides/pulumi-packages/) representing the [AWS Load Balancer Controller](https://docs.aws.amazon.com/eks/latest/userguide/aws-load-balancer-controller.html). It installs everything needed to run an AWS Load Balancer Controller in an [Amazon EKS](https://aws.amazon.com/eks/) cluster. It will install:

  - An adequately scoped IAM role
  - A Kubernetes deployment, with configurable replicas
  - The CRDs, if specified

It's written in Go, but thanks to Pulumi's multi language SDK generating capability, it create usable SDKs for all of Pulumi's [supported languages](https://www.pulumi.com/docs/intro/languages/)

> :warning: **This package is a work in progress**: Please do not use this in a production environment!

# Installing

## Install Plugin Binary

Before you begin, you'll need to install the latest version of the Pulumi Plugin using `pulumi plugin install`:

```
pulumi plugin install resource awsloadbalancercontroller 0.0.1-alpha.1621481781+0b34526c --server https://lbriggs.jfrog.io/artifactory/pulumi-packages/pulumi-awsloadbalancercontroller
```

This installs the plugin into `~/.pulumi/plugins`.

## Install your chosen SDK

Next, you need to install your desired language SDK using your languages package manager.

### Python

```
pip3 install jaxxstorm-pulumi-awsloadbalancercontroller
```

### NodeJS

```
npm install @jaxxstorm/pulumi-awsloadbalancercontroller
```

### DotNet

```
Coming Soon
```

### Go

```
go get -t github.com/jaxxstorm/pulumi-awsloadbalancercontroller/sdk/go/awsloadbalancercontroller
```

# Usage

Once you've installed all the dependencies, you can use the library like any other Pulumi SDK. See the [examples](examples/) directory for examples of how you might use it.

# Limitations

Currently, this package will only work successfully on Amazon EKS clusters with [IAM Roles for Service Accounts](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) enabled.