.PHONY: build test vet run dev clean frontend-build frontend-dev docker-build docker-login docker-build-kronize docker-push-kronize redeploy

BINARY   = kronize
DB       = ./data/kronize.db
SCRIPTS  = ./data/scripts

# ── Build ─────────────────────────────────────────────────
build: frontend-build
	go build -o $(BINARY) .

build-go:
	go build -o $(BINARY) .

frontend-build:
	cd frontend && npm run build

frontend-dev:
	cd frontend && npm run dev

# ── Verify ────────────────────────────────────────────────
test:
	go test ./internal/db/ -v

vet:
	go vet ./...

lint: vet
	cd frontend && npx tsc --noEmit

check: test vet frontend-build

# ── Run ───────────────────────────────────────────────────
run:
	mkdir -p $(SCRIPTS)
	./$(BINARY) -db $(DB) -scripts $(SCRIPTS) -addr :8080 -jwt-secret dev-secret

dev: build
	mkdir -p $(SCRIPTS)
	./$(BINARY) -db $(DB) -scripts $(SCRIPTS) -addr :8080 -jwt-secret dev-secret

# Dev with frontend hot-reload (requires two terminals)
dev-api:
	mkdir -p $(SCRIPTS)
	go run . -db $(DB) -scripts $(SCRIPTS) -addr :8080 -jwt-secret dev-secret

dev-ui: frontend-dev

# ── Docker ────────────────────────────────────────────────
PROXMOX_HOST     ?= 172.17.5.86
KRONIZE_CT_ID    ?= 110

docker-build:
	docker build -t kronize/python-runner docker/python-runner/

redeploy:
	ssh root@$(PROXMOX_HOST) "\
		pct exec $(KRONIZE_CT_ID) -- docker compose -f /opt/kronize/docker-compose.yml pull kronize && \
		pct exec $(KRONIZE_CT_ID) -- docker compose -f /opt/kronize/docker-compose.yml up -d kronize \
	"
	@echo "Done. kronize restarted on container $(KRONIZE_CT_ID)."

# ── Clean ─────────────────────────────────────────────────
clean:
	rm -f $(BINARY)
	rm -rf frontend/dist
	rm -rf data/
