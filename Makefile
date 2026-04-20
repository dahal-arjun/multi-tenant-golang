include .env
export

MIGRATE=atlas migrate

migrate-status:
	$(MIGRATE) status --url "mysql://$(DB_USER):$(DB_PASS)@:$(DB_FORWARD_PORT)/$(DB_NAME)"

migrate-diff:
	$(MIGRATE) diff --env gorm

migrate-apply:
	$(MIGRATE) apply --url "mysql://$(DB_USER):$(DB_PASS)@:$(DB_FORWARD_PORT)/$(DB_NAME)"

migrate-down:
	$(MIGRATE) down --url "mysql://$(DB_USER):$(DB_PASS)@:$(DB_FORWARD_PORT)/$(DB_NAME)" --env gorm

migrate-hash:
	$(MIGRATE) hash

lint-setup:
	python3 -m ensurepip --upgrade
	sudo pip3 install pre-commit
	pre-commit install
	pre-commit autoupdate

.PHONY: migrate-status migrate-diff migrate-apply migrate-down migrate-hash lint-setup
