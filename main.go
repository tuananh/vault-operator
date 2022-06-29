package main

import (
	"flag"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/tuananh/vault-operator/pkg/controller"
	"github.com/tuananh/vault-operator/pkg/version"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	versionFlag = flag.Bool("version", false, "print version")
)

func main() {
	flag.Parse()

	fmt.Printf("Version: %s\n", version.Get())
	if *versionFlag {
		return
	}

	ctx := signals.SetupSignalHandler()
	logrus.SetLevel(logrus.DebugLevel)
	if err := controller.Start(ctx); err != nil {
		logrus.Fatal(err)
	}
	<-ctx.Done()
	logrus.Fatal(ctx.Err())
}
