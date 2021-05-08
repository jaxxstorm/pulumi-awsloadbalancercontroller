package provider

import (
	"encoding/json"

	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/iam"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// The set of arguments for creating a AWSLBController component resource.
type AWSLBControllerArgs struct {
	Namespace    string `pulumi:"namespace"` // FIXME: should be a stringinput
	OidcIssuer   string `pulumi:"oidcIssuer"`
	OidcProvider string `pulumi:"oidcProvider"`
}

// The AWSLBController component resource.
type AWSLBController struct {
	pulumi.ResourceState
}

// NewAWSLBController creates a new AWSLBController component resource.
func NewAWSLBController(ctx *pulumi.Context,
	name string, args *AWSLBControllerArgs, opts ...pulumi.ResourceOption) (*AWSLBController, error) {
	if args == nil {
		args = &AWSLBControllerArgs{}
	}

	component := &AWSLBController{}
	err := ctx.RegisterComponentResource(AWSLBControllerToken, name, component, opts...)
	if err != nil {
		return nil, err
	}

	var namespace *corev1.Namespace

	namespace, err = corev1.NewNamespace(ctx, fmt.Sprintf("%s-ns", name), &corev1.NamespaceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String(args.Namespace),
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("error creating namespace: %v", err)
	}

	_ = namespace

	assumeRolePolicyJSON, _ := json.Marshal(map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []interface{}{
			map[string]interface{}{
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"Federated": args.OidcProvider,
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": map[string]interface{}{
					"StringEquals": map[string]interface{}{
						fmt.Sprintf("%s:sub", args.OidcIssuer): fmt.Sprintf("system:serviceaccount:%s:%s-serviceaccount", args.Namespace, name),
					},
				},
			},
		},
	})

	role, err := iam.NewRole(ctx, fmt.Sprintf("%s-role", name), &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(assumeRolePolicyJSON),
	}, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("error creating IAM role: %v", err)
	}

	policy, err := iam.NewPolicy(ctx, fmt.Sprintf("%s-policy", name), &iam.PolicyArgs{
		Policy: pulumi.String(iamPolicyData),
	}, pulumi.Parent(role))
	if err != nil {
		return nil, fmt.Errorf("error creating IAM policy: %v", err)
	}

	_, err = iam.NewRolePolicyAttachment(ctx, fmt.Sprintf("%s-policyAttachment", name), &iam.RolePolicyAttachmentArgs{
		Role:      role,
		PolicyArn: policy.Arn,
	}, pulumi.Parent(policy))
	if err != nil {
		return nil, fmt.Errorf("error creating IAM policy attachment: %v", err)
	}

	_, err = corev1.NewServiceAccount(ctx, fmt.Sprintf("%s-serviceAccount", name), &corev1.ServiceAccountArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.Sprintf("%s-serviceaccount", name),
			Namespace: namespace.Metadata.Name().Elem(),
			Annotations: pulumi.StringMap{
				"eks.amazonaws.com/role-arn": role.Arn.ApplyT(func(arn string) string {
					return arn
				}).(pulumi.StringOutput),
			},
		},
	}, pulumi.Parent(namespace))
	if err != nil {
		return nil, fmt.Errorf("error creating service account: %v", err)
	}

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{}); err != nil {
		return nil, err
	}

	return component, nil
}
