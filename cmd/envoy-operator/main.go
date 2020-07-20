package main

import (
	"context"
	"runtime"

	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	stub "github.com/solo-io/envoy-operator/pkg/stub"

	"flag"
	"log"

	"github.com/sirupsen/logrus"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	namespace := flag.String("n", "default", "the namespace in which to monitor Envoy CRDs and manage "+
		"resources")
	flag.Parse()
	printVersion()
	log.Printf("Envoy Operator: using namespace %s", *namespace)
	sdk.Watch("envoy.solo.io/v1alpha1", "Envoy", *namespace, 5)
	sdk.Handle(stub.NewHandler())
	sdk.Run(context.TODO())
}
