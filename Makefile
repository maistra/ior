default: build

EXE = ior

clean:
	rm -f ./cmd/${EXE} && rm -f container/${EXE}

GOSTATIC = -ldflags '-extldflags "-static"'
build:
	CGO_ENABLED=0 go build -o ./cmd/${EXE} ${GOSTATIC} ./cmd/...

image: build
	cp ./cmd/${EXE} container/ && \
	cd container && \
	docker build -t docker.io/maistra/ior .

push: image
	docker push docker.io/maistra/ior

pod: build image
	kubectl -n ior delete --ignore-not-found=true --now=true ns ior && \
	kubectl create ns ior && \
	kubectl -n ior apply -f container/pod.yaml
