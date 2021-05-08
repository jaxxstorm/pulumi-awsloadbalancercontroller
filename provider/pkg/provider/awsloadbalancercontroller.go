package provider

import (
	"encoding/base64"
	"encoding/json"

	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/iam"
	addregv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/admissionregistration/v1"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	rbacv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/rbac/v1"
	yaml "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/yaml"
	tls "github.com/pulumi/pulumi-tls/sdk/v4/go/tls"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// The set of arguments for creating a AWSLBController component resource.
type AWSLBControllerArgs struct {
	Namespace    pulumi.StringInput `pulumi:"namespace"`
	ClusterName  string             `pulumi:"clusterName"`
	OidcIssuer   string             `pulumi:"oidcIssuer"`
	OidcProvider string             `pulumi:"oidcProvider"`
	InstallCRDs  bool               `pulumi:"installCRDs"`
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
			Name: args.Namespace,
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("error creating namespace: %v", err)
	}

	assumeRolePolicyJSON := namespace.Metadata.Name().Elem().ApplyT(
		func(ns string) (string, error) {
			policyJSON, err := json.Marshal(map[string]interface{}{
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
								fmt.Sprintf("%s:sub", args.OidcIssuer): fmt.Sprintf("system:serviceaccount:%s:%s-serviceaccount", ns, name),
							},
						},
					},
				},
			})
			if err != nil {
				return "", err
			}
			return string(policyJSON), nil
		},
	).(pulumi.StringOutput)

	iamRole, err := iam.NewRole(ctx, fmt.Sprintf("%s-role", name), &iam.RoleArgs{
		AssumeRolePolicy: assumeRolePolicyJSON,
	}, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("error creating IAM role: %v", err)
	}

	policy, err := iam.NewPolicy(ctx, fmt.Sprintf("%s-policy", name), &iam.PolicyArgs{
		Policy: pulumi.String(iamPolicyData),
	}, pulumi.Parent(iamRole))
	if err != nil {
		return nil, fmt.Errorf("error creating IAM policy: %v", err)
	}

	_, err = iam.NewRolePolicyAttachment(ctx, fmt.Sprintf("%s-policy-attachment", name), &iam.RolePolicyAttachmentArgs{
		Role:      iamRole,
		PolicyArn: policy.Arn,
	}, pulumi.Parent(policy))
	if err != nil {
		return nil, fmt.Errorf("error creating IAM policy attachment: %v", err)
	}

	// Shared labels for all resources
	labels := pulumi.StringMap{
		"app.kubernetes.io/name":     pulumi.String("aws-loadbalancer-controller"),
		"app.kubernetes.io/instance": pulumi.String(name),
	}

	serviceAccount, err := corev1.NewServiceAccount(ctx, fmt.Sprintf("%s-serviceaccount", name), &corev1.ServiceAccountArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.Sprintf("%s-serviceaccount", name),
			Namespace: namespace.Metadata.Name().Elem(),
			Labels:    labels,
			Annotations: pulumi.StringMap{
				"eks.amazonaws.com/role-arn": iamRole.Arn.ApplyT(func(arn string) string {
					return arn
				}).(pulumi.StringOutput),
			},
		},
	}, pulumi.Parent(namespace))
	if err != nil {
		return nil, fmt.Errorf("error creating service account: %v", err)
	}

	clusterRole, err := rbacv1.NewClusterRole(ctx, fmt.Sprintf("%s-clusterrole", name), &rbacv1.ClusterRoleArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Labels: labels,
		},
		Rules: &rbacv1.PolicyRuleArray{
			&rbacv1.PolicyRuleArgs{
				ApiGroups: pulumi.StringArray{
					pulumi.String("elbv2.k8s.aws"),
				},
				Resources: pulumi.StringArray{
					pulumi.String("targetgroupbindings"),
				},
				Verbs: pulumi.StringArray{
					pulumi.String("create"),
					pulumi.String("delete"),
					pulumi.String("get"),
					pulumi.String("list"),
					pulumi.String("patch"),
					pulumi.String("update"),
					pulumi.String("watch"),
				},
			},
			&rbacv1.PolicyRuleArgs{
				ApiGroups: pulumi.StringArray{
					pulumi.String(""),
				},
				Resources: pulumi.StringArray{
					pulumi.String("events"),
				},
				Verbs: pulumi.StringArray{
					pulumi.String("create"),
					pulumi.String("patch"),
				},
			},
			&rbacv1.PolicyRuleArgs{
				ApiGroups: pulumi.StringArray{
					pulumi.String(""),
				},
				Resources: pulumi.StringArray{
					pulumi.String("pods"),
				},
				Verbs: pulumi.StringArray{
					pulumi.String("get"),
					pulumi.String("list"),
					pulumi.String("watch"),
				},
			},
			&rbacv1.PolicyRuleArgs{
				ApiGroups: pulumi.StringArray{
					pulumi.String(""),
					pulumi.String("extensions"),
					pulumi.String("networking.k8s.io"),
				},
				Resources: pulumi.StringArray{
					pulumi.String("services"),
					pulumi.String("ingresses"),
				},
				Verbs: pulumi.StringArray{
					pulumi.String("get"),
					pulumi.String("list"),
					pulumi.String("patch"),
					pulumi.String("update"),
					pulumi.String("watch"),
				},
			},
			&rbacv1.PolicyRuleArgs{
				ApiGroups: pulumi.StringArray{
					pulumi.String(""),
				},
				Resources: pulumi.StringArray{
					pulumi.String("nodes"),
					pulumi.String("secrets"),
					pulumi.String("namespaces"),
					pulumi.String("endpoints"),
				},
				Verbs: pulumi.StringArray{
					pulumi.String("get"),
					pulumi.String("list"),
					pulumi.String("watch"),
				},
			},
			&rbacv1.PolicyRuleArgs{
				ApiGroups: pulumi.StringArray{
					pulumi.String(""),
					pulumi.String("elbv2.k8s.aws"),
					pulumi.String("extensions"),
					pulumi.String("networking.k8s.io"),
				},
				Resources: pulumi.StringArray{
					pulumi.String("targetgroupbindings/status"),
					pulumi.String("pods/status"),
					pulumi.String("services/status"),
					pulumi.String("ingresses/status"),
				},
				Verbs: pulumi.StringArray{
					pulumi.String("update"),
					pulumi.String("patch"),
				},
			},
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("error creating cluster role: %v", err)
	}

	_, err = rbacv1.NewClusterRoleBinding(ctx, fmt.Sprintf("%s-clusterrole-binding", name), &rbacv1.ClusterRoleBindingArgs{
		Metadata: metav1.ObjectMetaArgs{
			Labels: labels,
		},
		RoleRef: &rbacv1.RoleRefArgs{
			ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
			Kind:     pulumi.String("ClusterRole"),
			Name:     clusterRole.Metadata.Name().Elem(),
		},
		Subjects: &rbacv1.SubjectArray{
			&rbacv1.SubjectArgs{
				Kind:      pulumi.String("ServiceAccount"),
				Name:      serviceAccount.Metadata.Name().Elem(),
				Namespace: namespace.Metadata.Name().Elem(),
			},
		},
	}, pulumi.Parent(clusterRole))
	if err != nil {
		return nil, fmt.Errorf("error creating cluster role binding: %v", err)
	}

	role, err := rbacv1.NewRole(ctx, fmt.Sprintf("%s-role", name), &rbacv1.RoleArgs{
		Metadata: metav1.ObjectMetaArgs{
			Labels:    labels,
			Namespace: namespace.Metadata.Name().Elem(),
		},
		Rules: &rbacv1.PolicyRuleArray{
			&rbacv1.PolicyRuleArgs{
				ApiGroups: pulumi.StringArray{
					pulumi.String(""),
				},
				Resources: pulumi.StringArray{
					pulumi.String("configmaps"),
				},
				Verbs: pulumi.StringArray{
					pulumi.String("create"),
				},
			},
			&rbacv1.PolicyRuleArgs{
				ApiGroups: pulumi.StringArray{
					pulumi.String(""),
				},
				Resources: pulumi.StringArray{
					pulumi.String("configmaps"),
				},
				ResourceNames: pulumi.StringArray{
					pulumi.String("aws-load-balancer-controller-leader"),
				},
				Verbs: pulumi.StringArray{
					pulumi.String("get"),
					pulumi.String("patch"),
					pulumi.String("update"),
				},
			},
		},
	}, pulumi.Parent(namespace))
	if err != nil {
		return nil, fmt.Errorf("error creating kubernetes role: %v", err)
	}

	_, err = rbacv1.NewRoleBinding(ctx, fmt.Sprintf("%s-rolebinding", name), &rbacv1.RoleBindingArgs{
		Metadata: metav1.ObjectMetaArgs{
			Labels:    labels,
			Namespace: namespace.Metadata.Name().Elem(),
		},
		RoleRef: &rbacv1.RoleRefArgs{
			ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
			Kind:     pulumi.String("Role"),
			Name:     role.Metadata.Name().Elem(),
		},
		Subjects: &rbacv1.SubjectArray{
			&rbacv1.SubjectArgs{
				Kind:      pulumi.String("ServiceAccount"),
				Name:      serviceAccount.Metadata.Name().Elem(),
				Namespace: namespace.Metadata.Name().Elem(),
			},
		},
	}, pulumi.Parent(role))
	if err != nil {
		return nil, fmt.Errorf("error creating role binding: %v", err)
	}

	/*
	 * Create certificates used by the webhook service
	 */

	// This is the certificate authority
	caKey, err := tls.NewPrivateKey(ctx, fmt.Sprintf("%s-ca-privatekey", name), &tls.PrivateKeyArgs{
		Algorithm:  pulumi.String("RSA"),
		EcdsaCurve: pulumi.String("P256"),
		RsaBits:    pulumi.Int(2048),
	}, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("error creating CA private key: %v", err)
	}

	caCert, err := tls.NewSelfSignedCert(ctx, fmt.Sprintf("%s-cacert", name), &tls.SelfSignedCertArgs{
		KeyAlgorithm:        caKey.Algorithm,
		PrivateKeyPem:       caKey.PrivateKeyPem,
		IsCaCertificate:     pulumi.Bool(true),
		ValidityPeriodHours: pulumi.Int(88600),
		AllowedUses: pulumi.StringArray{
			pulumi.String("cert_signing"),
			pulumi.String("digital_signature"),
			pulumi.String("key_encipherment"),
		},
		Subjects: &tls.SelfSignedCertSubjectArray{
			&tls.SelfSignedCertSubjectArgs{
				CommonName: pulumi.Sprintf("%s-aws-load-balancer-controller", name),
			},
		},
	}, pulumi.Parent(caKey))
	if err != nil {
		return nil, fmt.Errorf("error creating CA Cert: %v", err)
	}

	webhookSvc, err := corev1.NewService(ctx, fmt.Sprintf("%s-webhook-service", name), &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Labels:    labels,
			Namespace: namespace.Metadata.Name().Elem(),
			Annotations: pulumi.StringMap{
				"pulumi.com/skipAwait": pulumi.String("true"), // FIXME: why are we skipping await here?
			},
		},
		Spec: &corev1.ServiceSpecArgs{
			Ports: &corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Port:       pulumi.Int(443),
					TargetPort: pulumi.Int(9443),
				},
			},
			Selector: labels,
		},
	}, pulumi.Parent(namespace))
	if err != nil {
		return nil, fmt.Errorf("error creating Webhook Service: %v", err)
	}

	// Certificate and key used by the webhook
	certKey, err := tls.NewPrivateKey(ctx, fmt.Sprintf("%s-webhook-privatekey", name), &tls.PrivateKeyArgs{
		Algorithm:  pulumi.String("RSA"),
		EcdsaCurve: pulumi.String("P256"),
		RsaBits:    pulumi.Int(2048),
	}, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("error creating Webhook Certificate Key: %v", err)
	}

	certRequest, err := tls.NewCertRequest(ctx, fmt.Sprintf("%s-webhook-cert-request", name), &tls.CertRequestArgs{
		KeyAlgorithm:  pulumi.String("RSA"),
		PrivateKeyPem: certKey.PrivateKeyPem,
		DnsNames: pulumi.StringArray{ // FIXME: we shouldn't need to rerun this
			pulumi.All(webhookSvc.Metadata.Name().Elem(), namespace.Metadata.Name().Elem()).ApplyT(func(args interface{}) string {
				webhookName := args.([]interface{})[0].(string)
				namespaceName := args.([]interface{})[1].(string)
				return fmt.Sprintf("%s.%s", webhookName, namespaceName)
			}).(pulumi.StringOutput),
			pulumi.All(webhookSvc.Metadata.Name().Elem(), namespace.Metadata.Name().Elem()).ApplyT(func(args interface{}) string {
				webhookName := args.([]interface{})[0].(string)
				namespaceName := args.([]interface{})[1].(string)
				return fmt.Sprintf("%s.%s.svc", webhookName, namespaceName)
			}).(pulumi.StringOutput),
		},
		Subjects: &tls.CertRequestSubjectArray{
			&tls.CertRequestSubjectArgs{
				CommonName: webhookSvc.Metadata.Name().Elem(),
			},
		},
	}, pulumi.Parent(certKey))
	if err != nil {
		return nil, fmt.Errorf("error creating Webhook Certificate Request: %v", err)
	}

	cert, err := tls.NewLocallySignedCert(ctx, fmt.Sprintf("%s-webhook-certificate", name), &tls.LocallySignedCertArgs{
		CertRequestPem:      certRequest.CertRequestPem,
		CaKeyAlgorithm:      caKey.Algorithm,
		CaPrivateKeyPem:     caKey.PrivateKeyPem,
		CaCertPem:           caCert.CertPem,
		ValidityPeriodHours: pulumi.Int(88600),
		AllowedUses: pulumi.StringArray{
			pulumi.String("key_encipherment"),
			pulumi.String("digital_signature"),
		},
	}, pulumi.Parent(certRequest))
	if err != nil {
		return nil, fmt.Errorf("error creating Webhook Certificate: %v", err)
	}

	tlsSecret, err := corev1.NewSecret(ctx, fmt.Sprintf("%s-tls-secret", name), &corev1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Labels:    labels,
			Namespace: namespace.Metadata.Name().Elem(),
		},
		Type: pulumi.String("kubernetes.io/tls"),
		StringData: pulumi.StringMap{
			"ca.crt":  caCert.CertPem,
			"tls.crt": cert.CertPem,
			"tls.key": certKey.PrivateKeyPem,
		},
	}, pulumi.Parent(namespace), pulumi.DependsOn([]pulumi.Resource{certKey, cert, certRequest}))
	if err != nil {
		return nil, fmt.Errorf("error creating Webhook Secret: %v", err)
	}

	_, err = appsv1.NewDeployment(ctx, fmt.Sprintf("%s-deployment", name), &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Labels:    labels,
			Namespace: namespace.Metadata.Name().Elem(),
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1), // FIXME: make configurable
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: labels,
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: labels,
				},
				Spec: &corev1.PodSpecArgs{
					ServiceAccountName: serviceAccount.Metadata.Name().Elem(),
					Volumes: &corev1.VolumeArray{
						&corev1.VolumeArgs{
							Name: pulumi.String("cert"),
							Secret: &corev1.SecretVolumeSourceArgs{
								DefaultMode: pulumi.Int(420),
								SecretName:  tlsSecret.Metadata.Name().Elem(),
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContextArgs{
						FsGroup: pulumi.Int(65534),
					},
					Containers: &corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name: pulumi.String("aws-load-balancer-controller"),
							Args: pulumi.StringArray{
								pulumi.Sprintf("--cluster-name=%s", args.ClusterName),
								pulumi.String("--aws-region=us-west-2"), // FIXME: make region configurable
								pulumi.String("--ingress-class=alb"),    // FIXME: make ingress class configurable
							},
							Command: pulumi.StringArray{
								pulumi.String("/controller"),
							},
							SecurityContext: &corev1.SecurityContextArgs{
								AllowPrivilegeEscalation: pulumi.Bool(false),
								ReadOnlyRootFilesystem:   pulumi.Bool(true),
								RunAsNonRoot:             pulumi.Bool(true),
							},
							ImagePullPolicy: pulumi.String("IfNotPresent"),
							Image:           pulumi.String("amazon/aws-alb-ingress-controller:v2.1.3"),
							VolumeMounts: &corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{
									MountPath: pulumi.String("/tmp/k8s-webhook-server/serving-certs"),
									Name:      pulumi.String("cert"),
									ReadOnly:  pulumi.Bool(true),
								},
							},
							Ports: &corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{
									Name:          pulumi.String("webhook-server"),
									ContainerPort: pulumi.Int(9443),
									Protocol:      pulumi.String("TCP"),
								},
								&corev1.ContainerPortArgs{
									Name:          pulumi.String("metrics-server"),
									ContainerPort: pulumi.Int(8080),
									Protocol:      pulumi.String("TCP"),
								},
							},
							LivenessProbe: &corev1.ProbeArgs{
								FailureThreshold: pulumi.Int(2),
								HttpGet: &corev1.HTTPGetActionArgs{
									Path:   pulumi.String("/healthz"),
									Port:   pulumi.Int(61779),
									Scheme: pulumi.String("HTTP"),
								},
								InitialDelaySeconds: pulumi.Int(30),
								TimeoutSeconds:      pulumi.Int(10),
							},
						},
					},
					TerminationGracePeriodSeconds: pulumi.Int(10),
				},
			},
		},
	}, pulumi.Parent(namespace))
	if err != nil {
		return nil, fmt.Errorf("error creating Deployment: %v", err)
	}

	_, err = addregv1.NewMutatingWebhookConfiguration(ctx, fmt.Sprintf("%s-mutating-webhook", name), &addregv1.MutatingWebhookConfigurationArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Labels: labels,

			Namespace: namespace.Metadata.Name().Elem(),
		},
		Webhooks: &addregv1.MutatingWebhookArray{
			&addregv1.MutatingWebhookArgs{
				ClientConfig: &addregv1.WebhookClientConfigArgs{
					CaBundle: caCert.CertPem.ApplyT(func(pem string) string {
						return base64.StdEncoding.EncodeToString([]byte(pem))
					}).(pulumi.StringOutput),
					Service: &addregv1.ServiceReferenceArgs{
						Name:      webhookSvc.Metadata.Name().Elem(),
						Namespace: namespace.Metadata.Name().Elem(),
						Path:      pulumi.String("/mutate-elbv2-k8s-aws-v1beta1-targetgroupbinding"),
					},
				},
				FailurePolicy: pulumi.String("Fail"),
				Name:          pulumi.String("mtargetgroupbinding.elbv2.k8s.aws"),
				AdmissionReviewVersions: pulumi.StringArray{
					pulumi.String("v1beta1"),
				},
				Rules: &addregv1.RuleWithOperationsArray{
					&addregv1.RuleWithOperationsArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("elbv2.k8s.aws"),
						},
						ApiVersions: pulumi.StringArray{
							pulumi.String("v1beta1"),
						},
						Operations: pulumi.StringArray{
							pulumi.String("CREATE"),
							pulumi.String("UPDATE"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("targetgroupbindings"),
						},
					},
				},
				SideEffects: pulumi.String("None"),
			},
			&addregv1.MutatingWebhookArgs{
				ClientConfig: &addregv1.WebhookClientConfigArgs{
					CaBundle: caCert.CertPem.ApplyT(func(pem string) string {
						return base64.StdEncoding.EncodeToString([]byte(pem))
					}).(pulumi.StringOutput),
					Service: &addregv1.ServiceReferenceArgs{
						Name:      webhookSvc.Metadata.Name().Elem(),
						Namespace: namespace.Metadata.Name().Elem(),
						Path:      pulumi.String("/mutate-v1-pod"),
					},
				},
				FailurePolicy: pulumi.String("Fail"),
				Name:          pulumi.String("mpod.elbv2.k8s.aws"),
				AdmissionReviewVersions: pulumi.StringArray{
					pulumi.String("v1beta1"),
				},
				NamespaceSelector: &metav1.LabelSelectorArgs{
					MatchExpressions: &metav1.LabelSelectorRequirementArray{
						&metav1.LabelSelectorRequirementArgs{
							Key:      pulumi.String("elbv2.k8s.aws/pod-readiness-gate-inject"),
							Operator: pulumi.String("In"),
							Values: pulumi.StringArray{
								pulumi.String(
									pulumi.String("enabled"),
								),
							},
						},
					},
				},
				Rules: &addregv1.RuleWithOperationsArray{
					&addregv1.RuleWithOperationsArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String(""),
						},
						ApiVersions: pulumi.StringArray{
							pulumi.String("v1"),
						},
						Operations: pulumi.StringArray{
							pulumi.String("CREATE"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("pods"),
						},
					},
				},
				SideEffects: pulumi.String("None"),
			},
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("error creating mutating webhook: %v", err)
	}

	_, err = addregv1.NewValidatingWebhookConfiguration(ctx, fmt.Sprintf("%s-validating-webhook", name), &addregv1.ValidatingWebhookConfigurationArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Labels:    labels,
			Namespace: namespace.Metadata.Name().Elem(),
		},
		Webhooks: &addregv1.ValidatingWebhookArray{
			&addregv1.ValidatingWebhookArgs{
				ClientConfig: &addregv1.WebhookClientConfigArgs{
					CaBundle: caCert.CertPem.ApplyT(func(pem string) string {
						return base64.StdEncoding.EncodeToString([]byte(pem))
					}).(pulumi.StringOutput),
					Service: &addregv1.ServiceReferenceArgs{
						Name:      webhookSvc.Metadata.Name().Elem(),
						Namespace: namespace.Metadata.Name().Elem(),
						Path:      pulumi.String("/validate-elbv2-k8s-aws-v1beta1-targetgroupbinding"),
					},
				},
				FailurePolicy: pulumi.String("Fail"),
				Name:          pulumi.String("vtargetgroupbinding.elbv2.k8s.aws"),
				AdmissionReviewVersions: pulumi.StringArray{
					pulumi.String("v1beta1"),
				},
				Rules: &addregv1.RuleWithOperationsArray{
					&addregv1.RuleWithOperationsArgs{
						ApiGroups: pulumi.StringArray{
							pulumi.String("elbv2.k8s.aws"),
						},
						ApiVersions: pulumi.StringArray{
							pulumi.String("v1beta1"),
						},
						Operations: pulumi.StringArray{
							pulumi.String("CREATE"),
							pulumi.String("UPDATE"),
						},
						Resources: pulumi.StringArray{
							pulumi.String("targetgroupbindings"),
						},
					},
				},
				SideEffects: pulumi.String("None"),
			},
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("error creating validating webhook: %v", err)
	}

	if args.InstallCRDs {
		_, err = yaml.NewConfigFile(ctx, fmt.Sprintf("%s-crds", name), &yaml.ConfigFileArgs{
			// File: []string{
			// 	"https://raw.githubusercontent.com/kubernetes-sigs/aws-load-balancer-controller/main/config/crd/bases/elbv2.k8s.aws_targetgroupbindings.yaml",
			// 	"https://raw.githubusercontent.com/kubernetes-sigs/aws-load-balancer-controller/main/config/crd/bases/elbv2.k8s.aws_ingressclassparams.yaml",
			// },
			File: "https://raw.githubusercontent.com/kubernetes-sigs/aws-load-balancer-controller/main/config/crd/bases/elbv2.k8s.aws_targetgroupbindings.yaml",
		})
		if err != nil {
			return nil, fmt.Errorf("error installing CRDs: %v", err)
		}
	}

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{}); err != nil {
		return nil, err
	}

	return component, nil
}
