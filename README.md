<h1 align="center">
    <img src="images/Envoy operator.png" alt="Envoy operator" width="419" height="150">
</h1>

The Envoy Operator project is a [Kubernetes Operator](https://coreos.com/operators/). It's purpose is to enable
easy deployment of Envoy proxies using a high level declarative API.

The Envoy Operator currently supports deploying proxies as standalone pods, but will soon
support injecting Envoy proxies as sidecar containers into existing pods to serve as transparent
proxies for use in a service mesh [such as Istio](https://istio.io/).

The Envoy Operator was built using the [operator sdk](https://github.com/operator-framework/operator-sdk).

# Quick usage

Deploy the Operator:
```
kubectl create -f https://raw.githubusercontent.com/solo-io/envoy-operator/master/deploy/rbac.yaml
kubectl create -f https://raw.githubusercontent.com/solo-io/envoy-operator/master/deploy/crd.yaml
kubectl create -f https://raw.githubusercontent.com/solo-io/envoy-operator/master/deploy/operator.yaml
```

Create an Envoy pod configured to use `ads-service.default.svc.cluster.local` as its ADS server:
```
cat <<EOF | kubectl create -f -
apiVersion: "envoy.solo.io/v1alpha1"
kind: "Envoy"
metadata:
  name: "myingress"
spec:
  adsServer: ads-service.default.svc.cluster.local
  adsPort: 8081
  clusterIdTemplate: ingress
  nodeIdTemplate: "ingress-{{.PodName}}"
EOF
```

# How does it work?
The operator transforms the Envoy spec defined [here](pkg/apis/envoy/v1alpha1/types.go) to a deployment
and a configmap that contains Envoy's static config file.

Note that some of the parameters are templates. these templates can be filled with the kube downward api.
Example:
```
apiVersion: "envoy.solo.io/v1alpha1"
kind: "Envoy"
metadata:
  name: "myingress"
spec:
  ...
  nodeIdTemplate: "{{.PodName}}"
```

The node id given to each Envoy will match its pod name.

The full template interpolation interface is defined [here](pkg/downward/interface.go) and should cover all of the downward API (labels and annotations included).

# Use cases
This operator's main uses case is with an ADS server [such as Gloo](https://github.com/solo-io/gloo). We are looking to hear more from the community about what other uses cases are of interest.


# Road Map
- SSL \ mTLS configuration
- Pod Injection
- Provide Locality information for zone aware routing.

# Help
Please join us on our slack channel [https://slack.solo.io/](https://slack.solo.io/) with any questions, feedback, or suggestions.
