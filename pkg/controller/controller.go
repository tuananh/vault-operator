package controller

import (
	"context"

	"github.com/acorn-io/baaah"
	"github.com/acorn-io/baaah/pkg/merr"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

func Start(ctx context.Context) error {
	scheme, err := newScheme()
	if err != nil {
		return err
	}

	router, err := baaah.DefaultRouter(scheme)
	if err != nil {
		return err
	}

	sel := klabels.SelectorFromSet(map[string]string{
		ManagedByVaultOperator: "true",
	})

	router.Type(&networkingv1.Ingress{}).Selector(sel).HandlerFunc(UpdateIngressWithAnnotation)

	return router.Start(ctx)
}

func newScheme() (*runtime.Scheme, error) {
	var (
		errs   []error
		scheme = runtime.NewScheme()
	)

	errs = append(errs, networkingv1.AddToScheme(scheme))
	// errs = append(errs, v1alpha1.AddToScheme(scheme))
	errs = append(errs, v1.AddToScheme(scheme))
	return scheme, merr.NewErrors(errs...)
}
