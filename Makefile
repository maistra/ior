default: build

clean:
	rm -f main

build:
	go build -o main -ldflags "-linkmode external -extldflags -static" ./cmd/...

copy: build
	kubectl -n ior cp main busybox:/main

image: build
	cp main container/ && \
	cd container && \
	docker build -t ior .

pod: build image
	kubectl -n ior delete --ignore-not-found=true --now=true ns ior && \
	kubectl create ns ior && \
	kubectl -n ior apply -f container/pod.yaml
