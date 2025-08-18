package main

import (
	"context"
	"flag"
	"k8s.io/klog/v2"
	"os"
	"os/signal"

	"github.com/red-hat-storage/ocs-operator/v4/controllers/util"
	"github.com/red-hat-storage/ocs-operator/v4/services/provider/server"

	"google.golang.org/grpc"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

func main() {
	//loggerOpts := zap.Options{}
	//loggerOpts.BindFlags(flag.CommandLine)
	//flag.Parse()
	//
	//logger := zap.New(zap.UseFlagOptions(&loggerOpts))
	ctrl.SetLogger(klog.NewKlogr())

	klog.Info("Starting Provider API server")

	namespace := util.GetPodNamespace()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	providerServer, err := server.NewOCSProviderServer(ctx, namespace)
	if err != nil {
		klog.Error(err, "failed to start provider server")
		return
	}

	providerServer.Start(*port, []grpc.ServerOption{})
}
