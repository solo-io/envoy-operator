package kube

import (
	"path/filepath"

	"github.com/golang/protobuf/ptypes"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/duration"

	envoy_config_bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	envoy_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_tls "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"

	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

func controlPlaneCluster(e *api.Envoy, tlsSecret *v1.Secret) (*envoy_cluster.Cluster, error) {
	var ret envoy_cluster.Cluster
	ret.Name = "ads-control-plane"
	ret.Http2ProtocolOptions = &envoy_core.Http2ProtocolOptions{}
	ret.ConnectTimeout = &duration.Duration{
		Seconds: 5,
	}
	hostAddress := &envoy_core.Address{
		Address: &envoy_core.Address_SocketAddress{
			SocketAddress: &envoy_core.SocketAddress{
				Address: e.Spec.ADSServer,
				PortSpecifier: &envoy_core.SocketAddress_PortValue{
					PortValue: uint32(e.Spec.ADSPort),
				},
			},
		},
	}
	ret.LoadAssignment = &envoy_endpoint.ClusterLoadAssignment{
		Endpoints: []*envoy_endpoint.LocalityLbEndpoints{{
			LbEndpoints: []*envoy_endpoint.LbEndpoint{{
				HostIdentifier: &envoy_endpoint.LbEndpoint_Endpoint{
					Endpoint: &envoy_endpoint.Endpoint{
						Address: hostAddress,
					},
				},
			}},
		}},
	}

	if tlsSecret != nil {

		// get the secret and see if we have a client certificate

		TLSCAFile := filepath.Join(api.EnvoyTLSVolPath, api.TLSCA)
		TLSCertFile := filepath.Join(api.EnvoyTLSVolPath, api.TLSCert)
		TLSKeyFile := filepath.Join(api.EnvoyTLSVolPath, api.TLSKey)

		// create client tls context
		tlsContext := &envoy_tls.UpstreamTlsContext{
			CommonTlsContext: &envoy_tls.CommonTlsContext{
				ValidationContextType: &envoy_tls.CommonTlsContext_ValidationContext{
					ValidationContext: &envoy_tls.CertificateValidationContext{
						TrustedCa: toDataSource(TLSCAFile),
					},
				},
			},
			Sni: e.Spec.ADSServer,
		}

		needClientCert := hasKey(tlsSecret)
		if needClientCert {

			tlsContext.CommonTlsContext.TlsCertificates = []*envoy_tls.TlsCertificate{{
				CertificateChain: toDataSource(TLSCertFile),
				PrivateKey:       toDataSource(TLSKeyFile),
			}}
		}

		tlsContextAny, err := ptypes.MarshalAny(tlsContext)
		if err != nil {
			return nil, err
		}
		ret.TransportSocket = &envoy_core.TransportSocket{
			Name: "tls",
			ConfigType: &envoy_core.TransportSocket_TypedConfig{
				TypedConfig: tlsContextAny,
			},
		}

	}
	// TODO setup TLS

	return &ret, nil
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

func toDataSource(f string) *envoy_core.DataSource {
	return &envoy_core.DataSource{
		Specifier: &envoy_core.DataSource_Filename{
			Filename: f,
		},
	}
}

func GenerateEnvoyConfig(e *api.Envoy, tlsSecret *v1.Secret) (string, error) {

	var cfgData string
	var bootstrapConfig envoy_config_bootstrap.Bootstrap
	bootstrapConfig.Node = &envoy_core.Node{
		Id:      e.Spec.NodeIdTemplate,
		Cluster: e.Spec.ClusterIdTemplate,
	}

	cluster, err := controlPlaneCluster(e, tlsSecret)
	if err != nil {
		return "", err
	}
	bootstrapConfig.StaticResources = &envoy_config_bootstrap.Bootstrap_StaticResources{
		Clusters: []*envoy_cluster.Cluster{cluster},
	}
	bootstrapConfig.DynamicResources = &envoy_config_bootstrap.Bootstrap_DynamicResources{
		AdsConfig: &envoy_core.ApiConfigSource{
			ApiType: envoy_core.ApiConfigSource_GRPC,
			GrpcServices: []*envoy_core.GrpcService{{
				TargetSpecifier: &envoy_core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &envoy_core.GrpcService_EnvoyGrpc{
						ClusterName: bootstrapConfig.StaticResources.Clusters[0].Name,
					},
				},
			}},
		},
		CdsConfig: &envoy_core.ConfigSource{
			ConfigSourceSpecifier: &envoy_core.ConfigSource_Ads{
				Ads: &envoy_core.AggregatedConfigSource{},
			},
		},
		LdsConfig: &envoy_core.ConfigSource{
			ConfigSourceSpecifier: &envoy_core.ConfigSource_Ads{
				Ads: &envoy_core.AggregatedConfigSource{},
			},
		},
	}
	bootstrapConfig.Admin = &envoy_config_bootstrap.Admin{}
	bootstrapConfig.Admin.AccessLogPath = "/dev/stderr"
	if e.Spec.AdminPort != 0 {
		bootstrapConfig.Admin.Address = &envoy_core.Address{
			Address: &envoy_core.Address_SocketAddress{
				SocketAddress: &envoy_core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &envoy_core.SocketAddress_PortValue{
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
