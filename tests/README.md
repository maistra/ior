# Very initial test suite

### Requirements:

1. OpenShift cluster up and running
1. OpenShift Routes must support wildcards. See https://docs.openshift.com/container-platform/3.11/install_config/router/default_haproxy_router.html#using-wildcard-routes
1. Istio already installed in `istio-system` namespace
1. jq: https://stedolan.github.io/jq/
1. jd: https://github.com/josephburnett/jd