include .env
export

MIGRATE=go run ariga.io/atlas/cmd/atlas@latest migrate

migrate-status:
	$(MIGRATE) status --url "postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable"

migrate-diff:
	$(MIGRATE) diff --env gorm

migrate-apply:
	$(MIGRATE) apply --url "postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable"

migrate-down:
	$(MIGRATE) down --url "postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable" --env gorm

migrate-hash:
	$(MIGRATE) hash

swagger:
	go run github.com/swaggo/swag/cmd/swag@latest init -g main.go -o docs --parseDependency --parseInternal

lint-setup:
	python3 -m ensurepip --upgrade
	sudo pip3 install pre-commit
	pre-commit install
	pre-commit autoupdate

.PHONY: migrate-status migrate-diff migrate-apply migrate-down migrate-hash swagger lint-setup
