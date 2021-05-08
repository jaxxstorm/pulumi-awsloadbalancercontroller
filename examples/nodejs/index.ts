import * as lb from "@jaxxstorm/pulumi-awsloadbalancercontroller";
import * as k8s from "@pulumi/kubernetes"

const yaml = new k8s.yaml.ConfigGroup("crds", {
    files: [
        "https://raw.githubusercontent.com/kubernetes-sigs/aws-load-balancer-controller/main/config/crd/bases/elbv2.k8s.aws_targetgroupbindings.yaml",
	    "https://raw.githubusercontent.com/kubernetes-sigs/aws-load-balancer-controller/main/config/crd/bases/elbv2.k8s.aws_ingressclassparams.yaml",
    ]
})

const loadbalancer = new lb.Deployment("test", {
    oidcIssuer: "oidc.eks.us-west-2.amazonaws.com/id/D4064024788B184AFFA7747591BD643D",
    oidcProvider: "arn:aws:iam::616138583583:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/D4064024788B184AFFA7747591BD643D",
    namespace: "aws-loadbalancer-controller",
    installCRDs: false,
    clusterName: "pulumi-nginx-demo-eksCluster-aa2add8",
}, { dependsOn: yaml })

const example = new k8s.yaml.ConfigFile("2048", {
    file: "https://raw.githubusercontent.com/kubernetes-sigs/aws-load-balancer-controller/main/docs/examples/2048/2048_full.yaml",
}, {
    dependsOn: loadbalancer
})
