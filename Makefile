test:
	go test ./lib
	go build ./cmd/gogitix
	cd test && bats test.bats

init:
	mkdir -p .git/hooks
	ln -s ../../scripts/pre-commit .git/hooks/pre-commit

.PHONY: init test
