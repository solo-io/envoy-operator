package downward_test

import (
	"bytes"
	"fmt"

	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"
	kube "github.com/solo-io/envoy-operator/pkg/kube"
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
				ClusterIdTemplate: "yuval",
				NodeIdTemplate:    "{{.PodName}}-yuval",
			},
		}
		cfg, err := kube.GenerateEnvoyConfig(&e)
		Expect(err).NotTo(HaveOccurred())
		fmt.Fprintf(GinkgoWriter, "config: %s", cfg)
		var b bytes.Buffer
		b.WriteString(cfg)
		var outb bytes.Buffer
		err = NewTransformer().Transform(&b, &outb)
		Expect(err).NotTo(HaveOccurred())
	})

})
