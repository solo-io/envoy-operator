package downward_test

import (
	"bytes"
	"fmt"

	envoy_config_bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	envoy_config_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"
	"github.com/solo-io/envoy-operator/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/solo-io/envoy-operator/pkg/downward"
)

var _ = Describe("Transform", func() {

	It("should transform", func() {

		// envoy crd:
		e := api.Envoy{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Envoy",
				APIVersion: "envoy.solo.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "myingress",
			},
			Spec: api.EnvoySpec{
				ADSServer:         "test.blah.com",
				ADSPort:           1234,
				ClusterIdTemplate: "soloio",
				NodeIdTemplate:    "{{.PodName}}-soloio",
			},
		}
		cfg, err := kube.GenerateEnvoyConfig(&e, nil)
		Expect(err).NotTo(HaveOccurred())
		fmt.Fprintf(GinkgoWriter, "config: %s", cfg)
		var b bytes.Buffer
		b.WriteString(cfg)
		var outb bytes.Buffer
		err = NewTransformer().Transform(&b, &outb)
		Expect(err).NotTo(HaveOccurred())
		Expect(outb.String()).To(ContainSubstring("soloio"))
	})

	Context("bootstrap transforms", func() {
		var (
			api             *mockDownward
			bootstrapConfig *envoy_config_bootstrap.Bootstrap
		)
		BeforeEach(func() {
			api = &mockDownward{
				podName: "Test",
				nodeIp:  "5.5.5.5",
			}
			bootstrapConfig = new(envoy_config_bootstrap.Bootstrap)
			bootstrapConfig.Node = &envoy_core.Node{}
		})

		It("should transform node id", func() {

			bootstrapConfig.Node.Id = "{{.PodName}}"
			err := TransformConfigTemplatesWithApi(bootstrapConfig, api)
			Expect(err).NotTo(HaveOccurred())
			Expect(bootstrapConfig.Node.Id).To(Equal("Test"))
		})

		It("should transform cluster", func() {
			bootstrapConfig.Node.Cluster = "{{.PodName}}"
			err := TransformConfigTemplatesWithApi(bootstrapConfig, api)
			Expect(err).NotTo(HaveOccurred())
			Expect(bootstrapConfig.Node.Cluster).To(Equal("Test"))
		})

		It("should transform metadata", func() {
			bootstrapConfig.Node.Metadata = &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"foo": {
						Kind: &structpb.Value_StringValue{
							StringValue: "{{.PodName}}",
						},
					},
				},
			}

			err := TransformConfigTemplatesWithApi(bootstrapConfig, api)
			Expect(err).NotTo(HaveOccurred())
			Expect(bootstrapConfig.Node.Metadata.Fields["foo"].Kind.(*structpb.Value_StringValue).StringValue).To(Equal("Test"))
		})

		It("should transform static resources", func() {
			bootstrapConfig.StaticResources = &envoy_config_bootstrap.Bootstrap_StaticResources{
				Clusters: []*envoy_config_cluster.Cluster{{
					LoadAssignment: &envoy_config_endpoint.ClusterLoadAssignment{
						Endpoints: []*envoy_config_endpoint.LocalityLbEndpoints{{
							LbEndpoints: []*envoy_config_endpoint.LbEndpoint{{
								HostIdentifier: &envoy_config_endpoint.LbEndpoint_Endpoint{
									Endpoint: &envoy_config_endpoint.Endpoint{
										Address: &envoy_core.Address{
											Address: &envoy_core.Address_SocketAddress{
												SocketAddress: &envoy_core.SocketAddress{
													Address: "{{.NodeIp}}",
												},
											},
										},
									},
								},
							}},
						}},
					},
				}},
			}

			err := TransformConfigTemplatesWithApi(bootstrapConfig, api)
			Expect(err).NotTo(HaveOccurred())

			expectedAddress := bootstrapConfig.GetStaticResources().GetClusters()[0].GetLoadAssignment().GetEndpoints()[0].GetLbEndpoints()[0].GetEndpoint().GetAddress().GetSocketAddress().GetAddress()
			Expect(expectedAddress).To(Equal("5.5.5.5"))
		})
	})
})
