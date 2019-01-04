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
	kubectl -n ior delete --ignore-not-found=true --now=true ns ior && \
	kubectl create ns ior && \
	sed -e "s|\$${HUB}|${HUB}|g" -e "s|\$${TAG}|${TAG}|g" -e "s|\$${NAMESPACE}|${NAMESPACE}|g" container/pod.yaml | kubectl -n ior apply -f-

test:
	./tests/test.sh
