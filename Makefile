test:
	go test -v ./...

up:
	docker compose up -d

down:
	docker compose down -v

db_logs:
	docker logs -f postgres_db_async_api

db_login:
	psql ${DATABASE_URL}

db_create_migration:
	docker run --rm -v $(PWD)/migrations:/migrations \
		migrate/migrate \
		create -ext sql -dir /migrations -seq $(name)

db_migrate_up:
	docker run --rm --network container:postgres_db_async_api -v $(PWD)/migrations:/migrations \
		migrate/migrate \
		-path=/migrations -database ${DOCKER_DATABASE_URL} up

db_migrate_down:
	docker run --rm --network container:postgres_db_async_api -v $(PWD)/migrations:/migrations \
		migrate/migrate \
		-path=/migrations -database ${DOCKER_DATABASE_URL} down 1

tf-plan:
	cd ./terraform && terraform plan

tf-apply:
	cd ./terraform && terraform apply -auto-approve

list-queues:
	aws --endpoint-url ${ENDPOINT_URL} sqs list-queues --no-cli-pager

list-buckets:
	aws --endpoint-url ${ENDPOINT_URL} s3 ls --no-cli-pager