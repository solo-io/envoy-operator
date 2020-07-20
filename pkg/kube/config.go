package kube

import (
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/duration"

	envoy_config_v2 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"

	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"
	"k8s.io/api/core/v1"
)

func controlPlaneCluster(e *api.Envoy, tlsSecret *v1.Secret) *envoy_api_v2.Cluster {
	var ret envoy_api_v2.Cluster
	ret.Name = "ads-control-plane"
	ret.Http2ProtocolOptions = &envoy_api_v2_core.Http2ProtocolOptions{}
	ret.ConnectTimeout = &duration.Duration{
		Seconds: 5,
	}
	ret.HiddenEnvoyDeprecatedHosts = []*envoy_api_v2_core.Address{{
		Address: &envoy_api_v2_core.Address_SocketAddress{
			SocketAddress: &envoy_api_v2_core.SocketAddress{
				Address: e.Spec.ADSServer,
				PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
					PortValue: uint32(e.Spec.ADSPort),
				},
			},
		},
	}}

	if tlsSecret != nil {

		// get the secret and see if we have a client certificate

		TLSCAFile := filepath.Join(api.EnvoyTLSVolPath, api.TLSCA)
		TLSCertFile := filepath.Join(api.EnvoyTLSVolPath, api.TLSCert)
		TLSKeyFile := filepath.Join(api.EnvoyTLSVolPath, api.TLSKey)

		// create client tls context
		ret.HiddenEnvoyDeprecatedTlsContext = &envoy_api_v2_auth.UpstreamTlsContext{
			CommonTlsContext: &envoy_api_v2_auth.CommonTlsContext{
				ValidationContextType: &envoy_api_v2_auth.CommonTlsContext_ValidationContext{
					ValidationContext: &envoy_api_v2_auth.CertificateValidationContext{
						TrustedCa: toDataSource(TLSCAFile),
					},
				},
			},
			Sni: e.Spec.ADSServer,
		}

		needClientCert := hasKey(tlsSecret)
		if needClientCert {

			ret.HiddenEnvoyDeprecatedTlsContext.CommonTlsContext.TlsCertificates = []*envoy_api_v2_auth.TlsCertificate{{
				CertificateChain: toDataSource(TLSCertFile),
				PrivateKey:       toDataSource(TLSKeyFile),
			}}
		}

	}
	// TODO setup TLS

	return &ret
}

func hasKey(secret *v1.Secret) bool {
	if _, ok := secret.Data[api.TLSKey]; !ok {
		return false
	}
	if _, ok := secret.Data[api.TLSCert]; !ok {
		return false
	}
	return true
}

func toDataSource(f string) *envoy_api_v2_core.DataSource {
	return &envoy_api_v2_core.DataSource{
		Specifier: &envoy_api_v2_core.DataSource_Filename{
			Filename: f,
		},
	}
}

func GenerateEnvoyConfig(e *api.Envoy, tlsSecret *v1.Secret) (string, error) {

	var cfgData string
	var bootstrapConfig envoy_config_v2.Bootstrap
	bootstrapConfig.Node = &envoy_api_v2_core.Node{
		Id:      e.Spec.NodeIdTemplate,
		Cluster: e.Spec.ClusterIdTemplate,
	}

	bootstrapConfig.StaticResources = &envoy_config_v2.Bootstrap_StaticResources{
		Clusters: []*envoy_api_v2.Cluster{controlPlaneCluster(e, tlsSecret)},
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
	bootstrapConfig.Admin = &envoy_config_v2.Admin{}
	bootstrapConfig.Admin.AccessLogPath = "/dev/stderr"
	if e.Spec.AdminPort != 0 {
		bootstrapConfig.Admin.Address = &envoy_api_v2_core.Address{
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
