package provider

import (
	// Embed necessary
	_ "embed"
)

//go:embed iam/iam_policy.json
var iamPolicyData []byte

//go:embed manifests/elbv2.k8s.aws_ingressclassparams.yaml
var ingressClassYAML string

//go:embed manifests/elbv2.k8s.aws_targetgroupbindings.yaml
var targetGroupBindingsYAML string
