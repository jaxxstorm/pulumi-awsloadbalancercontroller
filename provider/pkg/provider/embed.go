package provider

import (
	_ "embed"
)

//go:embed iam_policy.json
var iamPolicyData []byte
