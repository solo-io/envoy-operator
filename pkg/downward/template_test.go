package downward_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/solo-io/envoy-operator/pkg/downward"
)

type mockDownward struct {
	podName        string
	podNamespace   string
	podIp          string
	podSvcAccount  string
	podUID         string
	nodeName       string
	nodeIp         string
	podLabels      map[string]string
	podAnnotations map[string]string
}

func (di *mockDownward) PodName() string                   { return di.podName }
func (di *mockDownward) PodNamespace() string              { return di.podNamespace }
func (di *mockDownward) PodIp() string                     { return di.podIp }
func (di *mockDownward) PodSvcAccount() string             { return di.podSvcAccount }
func (di *mockDownward) PodUID() string                    { return di.podUID }
func (di *mockDownward) NodeName() string                  { return di.nodeName }
func (di *mockDownward) NodeIp() string                    { return di.nodeIp }
func (di *mockDownward) PodLabels() map[string]string      { return di.podLabels }
func (di *mockDownward) PodAnnotations() map[string]string { return di.podAnnotations }

var _ = Describe("Template", func() {
	var interpolator Interpolator
	var downwardMock *mockDownward
	BeforeEach(func() {
		interpolator = NewInterpolator()
		downwardMock = &mockDownward{
			podLabels:      map[string]string{},
			podAnnotations: map[string]string{},
		}
	})

	It("should interpolate annotations", func() {
		downwardMock.podAnnotations["Test"] = "mock"
		res, err := interpolator.InterpolateString("{{.PodAnnotations.Test}}", downwardMock)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal("mock"))
	})

	It("should interpolate labels", func() {
		downwardMock.podLabels["Test"] = "mock"
		res, err := interpolator.InterpolateString("{{.PodLabels.Test}}", downwardMock)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal("mock"))
	})

	It("should be empty when no label exist", func() {
		res, err := interpolator.InterpolateString("{{.PodLabels.Test}}", downwardMock)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(BeEmpty())
	})

	It("should interpolate podname", func() {
		downwardMock.podName = "mock"
		res, err := interpolator.InterpolateString("{{.PodName}}", downwardMock)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(Equal("mock"))
	})
	It("should missing podname as empty", func() {
		res, err := interpolator.InterpolateString("{{.PodName}}", downwardMock)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(BeEmpty())
	})

	It("should error on bad template", func() {
		_, err := interpolator.InterpolateString("{{ bad template", downwardMock)
		Expect(err).To(HaveOccurred())
	})

	It("should ransform", func() {
		_, err := interpolator.InterpolateString("{{ bad template", downwardMock)
		Expect(err).To(HaveOccurred())
	})

})
