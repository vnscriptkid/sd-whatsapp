up:
	docker compose up -d

down:
	docker compose down --volumes --remove-orphans

cli:
	docker compose exec redis redis-cli

start:
	go run ./server1
	go run ./server2