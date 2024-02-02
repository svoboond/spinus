.PHONY: browser build-nocache clean helm-dependency-update helm-lint hosts mrproper namespace run sqlc-generate up

browser:
	xdg-open http://127.0.0.1:8213

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

sqlc-generate:
	docker run -v ./sqlc.yaml:/app/sqlc.yaml -v ./internal/db:/app/internal/db `docker build -q tools/sqlc/`

up: helm-dependency-update
	skaffold dev
