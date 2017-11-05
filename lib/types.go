package lib

type Check interface{}

type ManyChecks struct {
	Checks   []Check
	Parallel bool
}

type SingleCheck struct {
	Command
}

type ReformatCheck struct {
	Check  SingleCheck
	Format SingleCheck
}
