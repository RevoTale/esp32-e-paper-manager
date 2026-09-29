SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
BUILD_DIR ?= $(CURDIR)/build
QUALITY_BASE_REF ?= HEAD

.PHONY: quality format lint workflows audit test coverage native native-build latency c-size interop tinygo firmware build tools
.NOTPARALLEL:
quality: format lint workflows audit test coverage native c-size interop tinygo firmware build

tools:
	go -C tools mod verify
	go -C tools build -trimpath -o bin/golangci-lint github.com/golangci/golangci-lint/v2/cmd/golangci-lint

format:
	@test -z "$$(gofmt -l $$(go list -f '{{.Dir}}' ./...))"
	@cd tools && test -z "$$(gofmt -l $$(go list -f '{{.Dir}}' ./...))"
	@cd firmware/esp32/tests/interop && test -z "$$(gofmt -l $$(go list -f '{{.Dir}}' ./...))"
	@find . -type d \( -name .git -o -name build -o -name 'build-*' -o -name bin \) -prune -o -type f \( -name '*.go' -o -name '*.c' -o -name '*.h' \) -print0 | xargs -0 awk 'FNR==1 { if (NR>1 && n>300) {print previous ": " n " lines"; bad=1} previous=FILENAME; n=0 } {n++} END {if(n>300) {print previous ": " n " lines"; bad=1} exit bad}'

lint: tools
	tools/bin/golangci-lint config verify
	tools/bin/golangci-lint run ./...
	cd tools && bin/golangci-lint run -c ../.golangci.yml ./...
	cd firmware/esp32/tests/interop && ../../../../tools/bin/golangci-lint run -c ../../../../.golangci.yml ./...

workflows:
	go -C tools build -trimpath -o bin/actionlint github.com/rhysd/actionlint/cmd/actionlint
	tools/bin/actionlint .github/workflows/quality.yml .github/workflows/publish.yml

audit:
	go -C tools build -trimpath -o bin/govulncheck golang.org/x/vuln/cmd/govulncheck
	tools/bin/govulncheck ./...

test:
	go test -race -count=1 -shuffle=off ./...
	go vet ./...
	go mod verify
	go -C tools test -count=1 -shuffle=off ./...

coverage:
	mkdir -p "$(BUILD_DIR)"
	@for trial in 1 2 3; do \
		echo "coverage run $$trial/3 (uncached, stable test order)"; \
		go test -count=1 -shuffle=off -covermode=set -coverprofile="$(BUILD_DIR)/coverage.out" $$(go list ./... | LC_ALL=C sort); \
		go -C tools test -count=1 -shuffle=off -covermode=set -coverprofile="$(BUILD_DIR)/tools-coverage.out" ./...; \
		actual=$$(go tool cover -func="$(BUILD_DIR)/coverage.out" | awk '/^total:/ {gsub(/%/, "", $$3); print $$3}'); \
		echo "total coverage: $$actual% (floor 94.6%)"; \
		awk -v actual="$$actual" 'BEGIN {if (actual == "" || actual < 94.6) exit 1}'; \
		for profile in coverage tools-coverage; do \
			LC_ALL=C sort "$(BUILD_DIR)/$$profile.out" > "$(BUILD_DIR)/$$profile-$$trial.sorted"; \
			diff -u "$(BUILD_DIR)/$$profile-1.sorted" "$(BUILD_DIR)/$$profile-$$trial.sorted"; \
		done; \
	done
	go -C tools run ./cmd/worktree-covercheck -root .. -base "$(QUALITY_BASE_REF)" -profiles "$(BUILD_DIR)/coverage.out,$(BUILD_DIR)/tools-coverage.out" -min 90

native-build:
	cmake -S firmware/esp32/tests -B "$(BUILD_DIR)/native" -G Ninja -DCMAKE_EXPORT_COMPILE_COMMANDS=ON
	cmake --build "$(BUILD_DIR)/native" -j 4
native: native-build
	ctest --test-dir "$(BUILD_DIR)/native" --output-on-failure --no-tests=error -LE latency

latency: native-build
	ctest --test-dir "$(BUILD_DIR)/native" --verbose --no-tests=error -L latency

c-size: native
	clang-tidy-19 -p "$(BUILD_DIR)/native" firmware/esp32/core/*.c firmware/esp32/tests/*.c firmware/esp32/platform/net_io.c firmware/esp32/platform/display_guard.c firmware/esp32/platform/uart.c

interop: native
	cd firmware/esp32/tests/interop && EP_CRYPTO_CLI="$(BUILD_DIR)/native/crypto_cli" EP_WIRE_CLI="$(BUILD_DIR)/native/wire_cli" EP_RECEIVER_CLI="$(BUILD_DIR)/native/receiver_cli" go test -timeout 30s ./...

tinygo:
	mkdir -p "$(BUILD_DIR)"
	@for pkg in screenwire streamrx refreshpolicy; do tinygo test -c -target=pico2-w -scheduler=tasks -o "$(BUILD_DIR)/$$pkg.elf" "./$$pkg"; done

firmware:
	. "$$IDF_PATH/export.sh" && cd firmware/esp32 && idf.py -B build-release -DSDKCONFIG=build-release/sdkconfig build

build:
	mkdir -p "$(BUILD_DIR)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "$(BUILD_DIR)/epaper-manager-amd64" ./cmd/epaper-manager
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o "$(BUILD_DIR)/epaper-manager-arm64" ./cmd/epaper-manager
