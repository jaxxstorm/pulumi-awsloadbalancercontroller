package main

import (
	lb "github.com/jaxxstorm/pulumi-awsloadbalancercontroller/sdk/go/awsloadbalancercontroller"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		_, err := lb.NewDeployment(ctx, "example", &lb.DeploymentArgs{
			ClusterName:  "example-cluster",
			InstallCRDs:  true,
			Namespace:    pulumi.String("aws-loadbalancer-controller"),
			OidcIssuer:   "arn:aws:iam::616138583583:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/6F5EB6A0B6482BE2960BD584BA77B7FB",
			OidcProvider: "oidc.eks.us-west-2.amazonaws.com/id/6F5EB6A0B6482BE2960BD584BA77B7FB",
		})

		if err != nil {
			return err
		}

		return nil
	})
}
