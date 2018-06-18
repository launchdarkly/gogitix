package lib

import (
	"sync"
)

func RunCheck(ws Workspace, executor Executor, check Check, skipReformat bool, err chan<- error) {
	defer close(err)

	switch check := check.(type) {
	case SingleCheck:
		err <- executor.Execute(ws, check.Command)
	case ReformatCheck:
		if !skipReformat {
			err <- Reformat(ws, executor, check, skipReformat)
		}
	case ManyChecks:
		wg := sync.WaitGroup{}
		childErrs := make([]chan error, len(check.Checks))
		stopEarly := make(chan error, 1)
	OUTER:
		for i, childCheck := range check.Checks {

			// Stop if we've had a failure already
			select {
			case <-stopEarly:
				break OUTER
			default:
			}

			childErr := make(chan error)
			childErrs[i] = childErr
			wg.Add(1)

			go func() {
				for {
					if childErr, ok := <-childErr; ok {
						err <- childErr // Forward errors to the parent
						if childErr != nil {
							stopEarly <- childErr
						}
					} else {
						wg.Done()
						break
					}
				}
			}()
			go RunCheck(ws, executor, childCheck, skipReformat, childErrs[i])
			if !check.Parallel {
				wg.Wait()
			}
		}
		wg.Wait()
	}
}
