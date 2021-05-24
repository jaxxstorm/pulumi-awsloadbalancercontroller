import * as lb from "@jaxxstorm/pulumi-awsloadbalancercontroller";
import * as k8s from "@pulumi/kubernetes"



const loadbalancer = new lb.Deployment("example", {
    oidcIssuer: "oidc.eks.us-west-2.amazonaws.com/id/D4064024788B184AFFA7747591BD643D",
    oidcProvider: "arn:aws:iam::616138583583:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/D4064024788B184AFFA7747591BD643D",
    namespace: "aws-loadbalancer-controller",
    installCRDs: true,
    clusterName: "example-cluster",
})
