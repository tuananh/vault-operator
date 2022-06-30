/*
Copyright 2022 Tuan Anh Tran <me@tuananh.org>
*/
package controller

import (
	"fmt"

	v1alpha1 "github.com/tuananh/vault-operator/pkg/apis/v1alpha1"
)

func ValidatePKI(secret *v1alpha1.VaultSecret) error {
	if secret.Spec.SecretEngine != "pki" {
		return nil
	}

	if secret.Spec.Role == "" {
		return fmt.Errorf("`Role' must be set")
	}

	if _, ok := secret.Spec.EngineOptions["common_name"]; !ok {
		return fmt.Errorf("`engineOptions.common_name' must be set")
	}

	return nil
}
