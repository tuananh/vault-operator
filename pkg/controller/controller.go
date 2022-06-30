/*
Copyright 2022 Tuan Anh Tran <me@tuananh.org>
*/
package controller

import (
	"context"
	"os"

	"github.com/acorn-io/baaah"
	"github.com/acorn-io/baaah/pkg/merr"
	"github.com/sirupsen/logrus"
	v1alpha1 "github.com/tuananh/vault-operator/pkg/apis/v1alpha1"
	"github.com/tuananh/vault-operator/pkg/vault"
	corev1 "k8s.io/api/core/v1"
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

	// initialize shared client
	err = vault.InitSharedClient()
	if err != nil {
		logrus.Error(err, "Could not create API client for Vault")
		os.Exit(1)
	} else {
		if vault.SharedClient != nil {
			if vault.SharedClient.PerformRenewToken() {
				go vault.SharedClient.RenewToken()
			}
		} else {
			logrus.Info("Shared client wasn't initialized, each secret must be use the vaultRole property")
		}
	}

	sel := klabels.SelectorFromSet(map[string]string{
		// ManagedByVaultOperator: "true",
	})

	// everythingSel := klabels.Everything()
	// router.Type(&networkingv1.Ingress{}).Selector(sel).HandlerFunc(UpdateIngressWithAnnotation)
	router.Type(&v1alpha1.VaultSecret{}).Selector(sel).HandlerFunc(VaultSecretHandler)

	return router.Start(ctx)
}

func newScheme() (*runtime.Scheme, error) {
	var (
		errs   []error
		scheme = runtime.NewScheme()
	)

	// errs = append(errs, networkingv1.AddToScheme(scheme))
	errs = append(errs, v1alpha1.AddToScheme(scheme))
	errs = append(errs, corev1.AddToScheme(scheme))
	errs = append(errs, v1.AddToScheme(scheme))
	return scheme, merr.NewErrors(errs...)
}
