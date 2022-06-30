/*
Copyright 2022 Tuan Anh Tran <me@tuananh.org>
*/
package controller

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/acorn-io/baaah/pkg/router"
	sprig "github.com/go-task/slim-sprig"
	"github.com/sirupsen/logrus"
	"github.com/tuananh/vault-operator/pkg/apis/v1alpha1"
	"github.com/tuananh/vault-operator/pkg/vault"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	conditionTypeSecretCreated  = "SecretCreated"
	conditionReasonFetchFailed  = "FetchFailed"
	conditionReasonCreated      = "Created"
	conditionReasonCreateFailed = "CreateFailed"
	conditionReasonUpdated      = "Updated"
	conditionReasonUpdateFailed = "UpdateFailed"
	conditionReasonMergeFailed  = "MergeFailed"
	conditionInvalidResource    = "InvalidResource"
)

const (
	kvEngine  = "kv"
	pkiEngine = "pki"
)

func UpdateIngressWithAnnotation(req router.Request, resp router.Response) error {
	ingress := req.Object.(*networkingv1.Ingress)
	if ingress.Annotations == nil {
		ingress.Annotations = map[string]string{}
	}
	ingress.Annotations["hello"] = "controller"

	return req.Client.Update(req.Ctx, ingress)
}

func VaultSecretHandler(req router.Request, resp router.Response) error {
	vaultsecret := req.Object.(*v1alpha1.VaultSecret)
	logrus.Debugf("VaultSecretHandler %v", vaultsecret)

	var data map[string][]byte

	var err error
	var vaultClient *vault.Client

	logrus.Debugf("vault role %v", vaultsecret.Spec)
	if vaultsecret.Spec.VaultRole != "" {
		// logrus.WithValues("vaultRole", instance.Spec.VaultRole).Info("Create client to get secret from Vault")
		logrus.Info("Create client to get secret from Vault")
		vaultClient, err = vault.CreateClient(vaultsecret.Spec.VaultRole)
		if err != nil {
			// Error creating the Vault client - requeue the request.
			updateConditions(req.Ctx, vaultsecret, conditionReasonFetchFailed, err.Error(), metav1.ConditionFalse)
			return err
		}
	} else {
		logrus.Info("Use shared client to get secret from Vault")
		if vault.SharedClient == nil {
			err = fmt.Errorf("shared client not initialized and vaultRole property missing")
			logrus.Error(err, "Could not get secret from Vault")
			updateConditions(req.Ctx, vaultsecret, conditionReasonFetchFailed, err.Error(), metav1.ConditionFalse)
			return err
		} else {
			vaultClient = vault.SharedClient
		}
	}

	if vaultsecret.Spec.SecretEngine == "" || vaultsecret.Spec.SecretEngine == kvEngine {
		data, err = vaultClient.GetSecret(vaultsecret.Spec.SecretEngine, vaultsecret.Spec.Path, vaultsecret.Spec.Keys, vaultsecret.Spec.Version, vaultsecret.Spec.IsBinary, vaultsecret.Spec.VaultNamespace)
		if err != nil {
			// Error while getting the secret from Vault - requeue the request.
			logrus.Error(err, "Could not get secret from vault")
			updateConditions(req.Ctx, vaultsecret, conditionReasonFetchFailed, err.Error(), metav1.ConditionFalse)
			return err
		}
	} else if vaultsecret.Spec.SecretEngine == pkiEngine {
		if err := ValidatePKI(vaultsecret); err != nil {
			logrus.Error(err, "Resource validation failed")
			updateConditions(req.Ctx, vaultsecret, conditionInvalidResource, err.Error(), metav1.ConditionFalse)
			return err
		}

		var expiration *time.Time
		data, expiration, err = vaultClient.GetCertificate(vaultsecret.Spec.Path, vaultsecret.Spec.Role, vaultsecret.Spec.EngineOptions)
		if err != nil {
			logrus.Error(err, "Could not get certificate from vault")
			updateConditions(req.Ctx, vaultsecret, conditionReasonFetchFailed, err.Error(), metav1.ConditionFalse)
			return err
		}

		// Requeue before expiration
		logrus.Infof("Certificate will expire on %s", expiration.String())
		ra := time.Until(*expiration) - vaultClient.GetPKIRenew()
		if ra <= 0 {
			// TODO: figure out what to do with this
			// reconcileResult.Requeue = true
		} else {
			// reconcileResult.RequeueAfter = ra
			logrus.Infof("Certificate will be renewed on %s", time.Now().Add(ra).String())
		}
	}

	// Define a new Secret object
	secret, err := newSecretForCR(vaultsecret, data)
	if err != nil {
		// Error while creating the Kubernetes secret - requeue the request.
		logrus.Error(err, "Could not create Kubernetes secret")
		updateConditions(req.Ctx, vaultsecret, conditionReasonCreateFailed, err.Error(), metav1.ConditionFalse)
		return err
	}

	// Set VaultSecret instance as the owner and controller
	// err = ctrl.SetControllerReference(instance, secret, r.Scheme)
	// if err != nil {
	// return ctrl.Result{}, err
	// }

	// Check if this Secret already exists
	found := &corev1.Secret{}
	err = req.Client.Get(req.Ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		logrus.Info("Creating a new Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = req.Client.Create(req.Ctx, secret)
		if err != nil {
			logrus.Error(err, "Could not create secret")
			updateConditions(req.Ctx, vaultsecret, conditionReasonCreateFailed, err.Error(), metav1.ConditionFalse)
			return err
		}

		// Secret created successfully - requeue only if no version is specified
		updateConditions(req.Ctx, vaultsecret, conditionReasonCreated, "Secret was created", metav1.ConditionTrue)
		return nil
	} else if err != nil {
		logrus.Error(err, "Could not create secret")
		updateConditions(req.Ctx, vaultsecret, conditionReasonCreateFailed, err.Error(), metav1.ConditionFalse)
		return err
	}

	// Secret already exists, update the secret
	// Merge -> Checks the existing data keys and merge them into the updated secret
	// Replace -> Do not check the data keys and replace the secret
	if vaultsecret.Spec.ReconcileStrategy == "Merge" {
		secret = mergeSecretData(secret, found)

		logrus.Info("Updating a Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = req.Client.Update(req.Ctx, secret)
		if err != nil {
			logrus.Error(err, "Could not update secret")
			updateConditions(req.Ctx, vaultsecret, conditionReasonMergeFailed, err.Error(), metav1.ConditionFalse)
			return err
		}
		// r.updateConditions(req.Ctx, instance, conditionReasonUpdated, "Secret was updated", metav1.ConditionTrue)
	} else {
		logrus.Info("Updating a Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = req.Client.Update(req.Ctx, secret)
		if err != nil {
			logrus.Error(err, "Could not update secret")
			updateConditions(req.Ctx, vaultsecret, conditionReasonUpdateFailed, err.Error(), metav1.ConditionFalse)
			return err
		}
		updateConditions(req.Ctx, vaultsecret, conditionReasonUpdated, "Secret was updated", metav1.ConditionTrue)
	}

	// Secret updated successfully - requeue only if no version is specified
	return nil
}

// newSecretForCR returns a secret with the same name/namespace as the CR. The secret will include all labels and
// annotations from the CR.
func newSecretForCR(cr *v1alpha1.VaultSecret, data map[string][]byte) (*corev1.Secret, error) {
	labels := map[string]string{}
	for k, v := range cr.ObjectMeta.Labels {
		labels[k] = v
	}

	annotations := map[string]string{}
	for k, v := range cr.ObjectMeta.Annotations {
		annotations[k] = v
	}

	if cr.Spec.Templates != nil {
		newdata := make(map[string][]byte)
		for k, v := range cr.Spec.Templates {
			templated, err := runTemplate(cr, v, data)
			if err != nil {
				return nil, fmt.Errorf("template ERROR: %w", err)
			}
			newdata[k] = templated
		}
		data = newdata
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        cr.Name,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: data,
		Type: cr.Spec.Type,
	}, nil
}

type templateVaultContext struct {
	Path    string
	Address string
}

type templateContext struct {
	Secrets     map[string]string
	Vault       templateVaultContext
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
}

// runTemplate executes a template with the given secrets map, filled with the Vault secrets
func runTemplate(cr *v1alpha1.VaultSecret, tmpl string, secrets map[string][]byte) ([]byte, error) {
	// Set up the context
	sd := templateContext{
		Secrets: make(map[string]string, len(secrets)),
		Vault: templateVaultContext{
			Path:    cr.Spec.Path,
			Address: os.Getenv("VAULT_ADDRESS"),
		},
		Namespace:   cr.Namespace,
		Labels:      cr.Labels,
		Annotations: cr.Annotations,
	}

	// For templating, these should all be strings, convert
	for k, v := range secrets {
		sd.Secrets[k] = string(v)
	}

	// We need to exclude some functions for security reasons and proper working of the operator, don't use TxtFuncMap:
	// - no environment-variable related functions to prevent secrets from accessing the VAULT environment variables
	// - no filesystem functions? Directory functions don't actually allow access to the FS, so they're OK.
	// - no other non-idempotent functions like random and crypto functions
	funcmap := sprig.HermeticTxtFuncMap()
	delete(funcmap, "genPrivateKey")
	delete(funcmap, "genCA")
	delete(funcmap, "genSelfSignedCert")
	delete(funcmap, "genSignedCert")
	delete(funcmap, "htpasswd") // bcrypt strings contain salt

	tmplParser := template.New("data").Funcs(funcmap)

	// use other delimiters to prevent clashing with Helm templates
	tmplParser.Delims("{%", "%}")

	t, err := tmplParser.Parse(tmpl)
	if err != nil {
		return nil, err
	}

	var bout bytes.Buffer
	err = t.Execute(&bout, sd)
	if err != nil {
		return nil, err
	}

	return bout.Bytes(), nil
}

func mergeSecretData(new, found *corev1.Secret) *corev1.Secret {
	for key, value := range found.Data {
		if _, ok := new.Data[key]; !ok {
			new.Data[key] = value
		}
	}

	return new
}

func updateConditions(ctx context.Context, instance *v1alpha1.VaultSecret, reason, message string, status metav1.ConditionStatus) {
	instance.Status.Conditions = []metav1.Condition{{
		Type:               conditionTypeSecretCreated,
		Status:             status,
		ObservedGeneration: instance.GetGeneration(),
		LastTransitionTime: metav1.NewTime(time.Now()),
		Reason:             reason,
		Message:            message,
	}}

	// err := r.Status().Update(ctx, instance)
	// if err != nil {
	// 	log.Error(err, "Could not update status")
	// }
}
