vault-operator
--------------

An example of writing Vault Operator for Kubernetes.

- Just an opinionated, simple way of bootstraping a new operator.
- Most Vault-related code is copied from [`vault-secret-operator`](https://github.com/ricoberger/vault-secrets-operator). This serves like a `Hello World` template for writing a K8s operator.
- I don't want to use Operator SDK since it's too complicated in my opinion.
- `kubebuilder`, `controller-gen` is used for generating CRD.
- `ko` is used for building multi-arch container image. Dockerfile is not used. SBOM generated by default so you can use `cosign` maybe to verify.

## Usage

```shell
# install the CRD: kubectl create -f config/crd
make run
```

## License

See [LICENSE](./LICENSE).