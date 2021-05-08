import * as lb from "@jaxxstorm/pulumi-awsloadbalancercontroller";

const loadbalancer = new lb.Deployment("test", {
    oidcIssuer: "oidc.eks.us-west-2.amazonaws.com/id/D4064024788B184AFFA7747591BD643D",
    oidcProvider: "arn:aws:iam::616138583583:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/D4064024788B184AFFA7747591BD643D",
    namespace: "aws-loadbalancer-controller"
})
