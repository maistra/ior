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

GOBINARY ?= go
EXE = ior
HUB ?= docker.io/maistra
TAG ?= latest
NAMESPACE ?= ior

LD_EXTRAFLAGS =

VERSION ?= development
LD_EXTRAFLAGS += -X github.com/maistra/ior/pkg/version.buildVersion=${VERSION}

GITREVISION ?= $(shell git rev-parse --verify HEAD 2> /dev/null)
ifeq ($(GITREVISION),)
  GITREVISION = unknown
endif
LD_EXTRAFLAGS += -X github.com/maistra/ior/pkg/version.buildGitRevision=${GITREVISION}


GITSTATUS ?= $(shell git diff-index --quiet HEAD --  2> /dev/null; echo $$?)
ifeq ($(GITSTATUS),0)
  GITSTATUS = Clean
else ifeq ($(GITSTATUS),1)
  GITSTATUS = Modified
else
  GITSTATUS = unknown
endif
LD_EXTRAFLAGS += -X github.com/maistra/ior/pkg/version.buildStatus=${GITSTATUS}

GITTAG ?= $(shell git describe 2> /dev/null)
ifeq ($(GITTAG),)
  GITTAG = unknown
endif
LD_EXTRAFLAGS += -X github.com/maistra/ior/pkg/version.buildTag=${GITTAG}

clean:
	rm -f ./cmd/${EXE} && rm -f container/${EXE}

LDFLAGS = '-extldflags -static ${LD_EXTRAFLAGS}'
build:
	CGO_ENABLED=0 ${GOBINARY} build -o ./cmd/${EXE} -ldflags ${LDFLAGS} ./cmd/...

image: build
	cp ./cmd/${EXE} container/ && \
	cd container && \
	docker build -t ${HUB}/ior:${TAG} .

push: image
	docker push ${HUB}/ior:${TAG}

pod: build image
	kubectl delete --ignore-not-found=true --now=true ns ${NAMESPACE} && \
	sed -e "s|\$${HUB}|${HUB}|g" -e "s|\$${TAG}|${TAG}|g" -e "s|\$${NAMESPACE}|${NAMESPACE}|g" container/pod.yaml | kubectl -n ${NAMESPACE} create -f-

cleanPod:
	sed -e "s|\$${HUB}|${HUB}|g" -e "s|\$${TAG}|${TAG}|g" -e "s|\$${NAMESPACE}|${NAMESPACE}|g" container/pod.yaml | kubectl -n ${NAMESPACE} delete -f-
test:
	./tests/test.sh
