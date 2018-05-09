# Envoy Operator!

Uses this to effectivly and easily deploy envoy cluster to your kubernetes environment.

# Quick usage

Deploy to your cluster:
```
kubectl create -f https://raw.githubusercontent.com/solo-io/envoy-operator/master/deploy/rbac.yaml
kubectl create -f https://raw.githubusercontent.com/solo-io/envoy-operator/master/deploy/crd.yaml
kubectl create -f https://raw.githubusercontent.com/solo-io/envoy-operator/master/deploy/operator.yaml
```

Create an envoy cluster pointing to your ADS server:
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

# How it works?
The operator transforms the envoy spec defined [here](pkg/apis/envoy/v1alpha1/types.go) to a deployment
and a configmap that contains envoy's static config file.

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

The node id given to each envoy will match its pod name.

The full template interpolation interface is defined [here](pkg/downward/interface.go) and should cover all of the downward API (labels and annotations included).

# Use cases
This operator's main uses case is with an ADS server. We are looking to hear more from the community about what other uses cases are of interest.


# Future plans
- SSL \ mTLS configuration
- Pod Injection
- Provide Locality information for zone aware routing.
