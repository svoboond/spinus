.PHONY: browser build-nocache clean helm-dependency-update helm-lint hosts mrproper namespace run \
	sqlc-delete sqlc-run sqlc-generate up

browser:
	xdg-open http://spinus.local

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

sqlc-delete:
	skaffold --filename=tools/sqlc/skaffold.yaml delete

sqlc-run:
	skaffold --filename=tools/sqlc/skaffold.yaml run

sqlc-generate: sqlc-run
	kubectl -n spinus-local-dev exec spinus-sqlc-local-dev-0 -- ./spinus-sqlc-generate --config local-conf.yaml
	kubectl -n spinus-local-dev cp spinus-sqlc-local-dev-0:/app/internal/db/sqlc internal/db/sqlc
	$(MAKE) --no-print-directory sqlc-delete

up: helm-dependency-update
	skaffold dev
