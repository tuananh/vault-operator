package controller

import (
	"github.com/acorn-io/baaah/pkg/router"
	networkingv1 "k8s.io/api/networking/v1"
)

func UpdateIngressWithAnnotation(req router.Request, resp router.Response) error {
	ingress := req.Object.(*networkingv1.Ingress)
	if ingress.Annotations == nil {
		ingress.Annotations = map[string]string{}
	}
	ingress.Annotations["hello"] = "controller"

	return req.Client.Update(req.Ctx, ingress)
}
