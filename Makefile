.PHONY: browser build-nocache clean \
	helm-dependency-update helm-lint \
	hosts mrproper namespace run \
	spinus-tools-delete spinus-tools-run sqlc-clean sqlc-generate templ-generate \
	up

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

spinus-tools-delete:
	skaffold --filename=tools/spinus-tools/skaffold.yaml delete

spinus-tools-run:
	skaffold --filename=tools/spinus-tools/skaffold.yaml run

sqlc-clean:
	rm -rf internal/db/sqlc/*

sqlc-generate: sqlc-clean spinus-tools-run
	kubectl -n spinus-local-dev exec spinus-tools-local-dev-0 -- ./spinus-sqlc-generate --config local-conf.yaml
	kubectl -n spinus-local-dev cp spinus-tools-local-dev-0:/app/internal/db/sqlc internal/db/sqlc
	$(MAKE) --no-print-directory spinus-tools-delete

templ-generate: spinus-tools-run
	kubectl -n spinus-local-dev exec spinus-tools-local-dev-0 -- ./templ generate
	kubectl -n spinus-local-dev cp spinus-tools-local-dev-0:/app/internal/ui internal/ui
	$(MAKE) --no-print-directory spinus-tools-delete

up: helm-dependency-update
	skaffold dev
