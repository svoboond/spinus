.PHONY: build-nocache clean helm-dependency-update helm-lint hosts mrproper namespace run up

build-nocache:
	skaffold build -p build-nocache

clean:
	rm -rf .build/* */charts
	find . -type f -name Chart.lock -delete

helm-dependency-update:
	helm dependency update helm

helm-lint: helm-dependency-update
	helm lint helm

hosts:
	echo "`minikube ip` spinus.local"

mrproper: clean

namespace:
	kubectl config set-context --current --namespace=spinus-local-dev

run: helm-dependency-update
	skaffold run --tail

up: helm-dependency-update
	skaffold dev
