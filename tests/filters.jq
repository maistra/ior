.
| del(.items[].metadata.creationTimestamp)
| del(.items[].metadata.name)
| del(.items[].metadata.resourceVersion)
| del(.items[].metadata.selfLink)
| del(.items[].metadata.labels."maistra.io/gateway-namespace")
| del(.items[].metadata.uid)
| del(.items[].status.ingress[].conditions[].lastTransitionTime)
