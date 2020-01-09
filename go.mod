module github.com/maistra/ior

go 1.13

require (
	github.com/openshift/api v3.9.1-0.20191008181517-e4fd21196097+incompatible
	github.com/openshift/client-go v0.0.0-20200107172225-986d9a10f405
	github.com/spf13/cobra v0.0.5
	google.golang.org/grpc v1.24.0
	istio.io/api v0.0.0-20191115173247-e1a1952e5b81
	istio.io/istio v0.0.0-20200106190329-17f6bfc3d712
	istio.io/pkg v0.0.0-20191030005435-10d06b6b315e
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

// Kubernetes makes it challenging to depend on their libraries. To get around this, we need to force
// the sha to use. All of these are pinned to the tag "kubernetes-1.16"
replace k8s.io/api => k8s.io/api v0.0.0-20191003000013-35e20aa79eb8

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655

// Pinned to Kubernetes 1.15 for now, due to some issues with 1.16
// TODO(https://github.com/istio/istio/issues/17831) upgrade to 1.16
replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190918200256-06eb1244587a

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191003002041-49e3d608220c

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191003002408-6e42c232ac7d

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191003002707-f6b7b0f55cc0
