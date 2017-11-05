init:
	mkdir -p .git/hooks
	ln -s scripts/pre-commit .git/hooks/pre-commit
