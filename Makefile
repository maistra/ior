## Copyright 2019 Red Hat, Inc.
##
## Licensed under the Apache License, Version 2.0 (the "License");
## you may not use this file except in compliance with the License.
## You may obtain a copy of the License at
##
##     http://www.apache.org/licenses/LICENSE-2.0
##
## Unless required by applicable law or agreed to in writing, software
## distributed under the License is distributed on an "AS IS" BASIS,
## WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
## See the License for the specific language governing permissions and
## limitations under the License.

default: build

EXE = ior
HUB ?= docker.io/maistra
TAG ?= latest
NAMESPACE ?= ior

clean:
	rm -f ./cmd/${EXE} && rm -f container/${EXE}

GOSTATIC = -ldflags '-extldflags "-static"'
build:
	CGO_ENABLED=0 go build -o ./cmd/${EXE} ${GOSTATIC} ./cmd/...

image: build
	cp ./cmd/${EXE} container/ && \
	cd container && \
	docker build -t ${HUB}/ior:${TAG} .

push: image
	docker push ${HUB}/ior:${TAG}

pod: build image
	kubectl -n ior delete --ignore-not-found=true --now=true ns ${NAMESPACE} && \
	sed -e "s|\$${HUB}|${HUB}|g" -e "s|\$${TAG}|${TAG}|g" -e "s|\$${NAMESPACE}|${NAMESPACE}|g" container/pod.yaml | kubectl -n ior create -f-

cleanPod:
	sed -e "s|\$${HUB}|${HUB}|g" -e "s|\$${TAG}|${TAG}|g" -e "s|\$${NAMESPACE}|${NAMESPACE}|g" container/pod.yaml | kubectl -n ior delete -f-
test:
	./tests/test.sh
