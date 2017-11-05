package lib

import (
	"sync"
)

func RunCheck(ws Workspace, executor Executor, check Check, err chan<- error) {
	switch check := check.(type) {
	case SingleCheck:
		err <- executor.Execute(ws, check.Command)
	case ReformatCheck:
		err <- Reformat(ws, executor, check)
	case ManyChecks:
		wg := sync.WaitGroup{}
		childErrs := make([]chan error, len(check.Checks))
		for i, childCheck := range check.Checks {
			childErrs[i] = make(chan error)
			wg.Add(1)
			go func() {
				for {
					if childErr, ok := <-childErrs[i]; !ok {
						wg.Done()
						break
					} else {
						err <- childErr // Forward errors to the parent
					}
				}
			}()
			go RunCheck(ws, executor, childCheck, childErrs[i])
			if !check.Parallel {
				wg.Wait()
			}
		}
		wg.Wait()
	}
	close(err)
}
