test:
	govendor test +local
	govendor build +local +program
	go build ./cmd/gogitix
	govendor test +local
	cd test && bats test.bats

init:
	mkdir -p .git/hooks
	ln -s ../../scripts/pre-commit .git/hooks/pre-commit

.PHONY: init test
