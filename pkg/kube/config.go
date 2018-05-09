package kube

import (
	"time"

	"github.com/gogo/protobuf/jsonpb"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_config_v2 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"

	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"
)

func controlPlaneCluster(e *api.Envoy) envoy_api_v2.Cluster {
	var ret envoy_api_v2.Cluster

	ret.Name = "blah"
	ret.Http2ProtocolOptions = &envoy_api_v2_core.Http2ProtocolOptions{}
	ret.Type = envoy_api_v2.Cluster_STRICT_DNS
	ret.ConnectTimeout = 5 * time.Second
	ret.Hosts = []*envoy_api_v2_core.Address{{
		Address: &envoy_api_v2_core.Address_SocketAddress{
			SocketAddress: &envoy_api_v2_core.SocketAddress{
				Address: e.Spec.ADSServer,
				PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
					PortValue: uint32(e.Spec.ADSPort),
				},
			},
		},
	}}

	// TODO setup TLS

	return ret
}

func GenerateEnvoyConfig(e *api.Envoy) (string, error) {

	var cfgData string
	var bootstrapConfig envoy_config_v2.Bootstrap
	bootstrapConfig.Node = &envoy_api_v2_core.Node{
		Id:      e.Spec.NodeIdTemplate,
		Cluster: e.Spec.ClusterIdTemplate,
	}
	bootstrapConfig.StaticResources = &envoy_config_v2.Bootstrap_StaticResources{
		Clusters: []envoy_api_v2.Cluster{controlPlaneCluster(e)},
	}
	bootstrapConfig.DynamicResources = &envoy_config_v2.Bootstrap_DynamicResources{
		AdsConfig: &envoy_api_v2_core.ApiConfigSource{
			ApiType: envoy_api_v2_core.ApiConfigSource_GRPC,
			GrpcServices: []*envoy_api_v2_core.GrpcService{{
				TargetSpecifier: &envoy_api_v2_core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &envoy_api_v2_core.GrpcService_EnvoyGrpc{
						ClusterName: bootstrapConfig.StaticResources.Clusters[0].Name,
					},
				},
			}},
		},
		CdsConfig: &envoy_api_v2_core.ConfigSource{
			ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
				Ads: &envoy_api_v2_core.AggregatedConfigSource{},
			},
		},
		LdsConfig: &envoy_api_v2_core.ConfigSource{
			ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
				Ads: &envoy_api_v2_core.AggregatedConfigSource{},
			},
		},
	}
	bootstrapConfig.Admin.AccessLogPath = "/dev/stderr"
	if e.Spec.AdminPort != 0 {
		bootstrapConfig.Admin.Address = envoy_api_v2_core.Address{
			Address: &envoy_api_v2_core.Address_SocketAddress{
				SocketAddress: &envoy_api_v2_core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
						PortValue: uint32(e.Spec.AdminPort),
					},
				},
			},
		}
	}

	// TODO: change to yaml?
	var marshaller jsonpb.Marshaler
	if jsondata, err := marshaller.MarshalToString(&bootstrapConfig); err != nil {
		return "", err
	} else {
		cfgData = jsondata
	}
	return cfgData, nil
}
