# Issues for this repository are disabled

Issues for this repository are tracked in Red Hat Jira. Please head to <https://issues.redhat.com/browse/MAISTRA> in order to browse or open an issue.

## Obsolete repository

This repository contains code for IOR <= 1.1. Newer versions of IOR are embedded directly in Istio: <https://github.com/maistra/istio>.

# IOR
`ior` = Istio + OpenShift Routing (I'm terrible with names, sorry)

## What is it?
`ior` aims to integrate Istio Gateways with OpenShift Routes. It manages (create, edit, delete) OpenShift Routes based on Istio Gateways, thus elimitating the need of manually creating routes.

As an example, if a user creates the Gateway:
```yaml
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: bookinfo-gateway
spec:
  selector:
    istio: ingressgateway # use istio default controller
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - "www.bookinfo.com"
    - "bookinfo.example.com"
```

Then the following OpenShift routes will be automatically created:
```
$ oc -n istio-system get routes
NAME                     HOST/PORT              PATH      SERVICES               PORT      TERMINATION   WILDCARD
bookinfo-gateway-7zsdx   bookinfo.example.com             istio-ingressgateway   <all>                   None
bookinfo-gateway-n6lq7   www.bookinfo.com                 istio-ingressgateway   <all>                   None
```

`ior` keeps Routes in sync with Gateways, meaning if you change or delete a Gateway, the Routes will be changed or deleted accordingly.

## Development

### Building from source

Just clone the repository and run make:
```sh
make
```

Important Makefile variables:
- `VERSION`: Controls the program version (`ior version` command). If not specified, the default value of `"development"` is assigned. Example of usage:
```sh
make VERSION=1.0.0
```
