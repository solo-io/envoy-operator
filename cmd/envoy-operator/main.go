package main

import (
	"context"
	"runtime"

	stub "github.com/solo-io/envoy-operator/pkg/stub"
	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	"github.com/sirupsen/logrus"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	printVersion()
	sdk.Watch("envoy.solo.io/v1alpha1", "Envoy", "default", 5)
	sdk.Handle(stub.NewHandler())
	sdk.Run(context.TODO())
}
