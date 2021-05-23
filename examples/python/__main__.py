"""A Python Pulumi program"""

import pulumi
import jaxxstorm_pulumi_awsloadbalancercontroller as lb

loadbalancer = lb.Deployment("example",
    cluster_name="example-cluster",
    install_crds=False,
    namespace="aws-loadbalancer-controller",
    oidc_provider="arn:aws:iam::616138583583:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/6F5EB6A0B6482BE2960BD584BA77B7FB",
    oidc_issuer="oidc.eks.us-west-2.amazonaws.com/id/6F5EB6A0B6482BE2960BD584BA77B7FB"
)
