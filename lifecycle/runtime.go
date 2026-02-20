package lifecycle

import (
	"os"

	"github.com/northconn/go-lib/telemetry/log"
)

type Stopper func()
type Starter func() (Stopper, error)
type Setup func() (Starter, error)

type Runtime struct {
	starters []Starter
	stoppers []Stopper
}

func NewRuntime(setups ...Setup) *Runtime {
	r := &Runtime{
		starters: make([]Starter, 0),
		stoppers: make([]Stopper, 0),
	}
	for _, setup := range setups {
		starter, err := setup()
		if err != nil {
			log.Logger().Error("failed to set up component", "error", err)
			os.Exit(1)
		}
		if starter != nil {
			r.starters = append(r.starters, starter)
		}
	}
	return r
}

func (r *Runtime) Start() error {
	for _, starter := range r.starters {
		stopper, err := starter()
		if err != nil {
			return err
		}
		if stopper != nil {
			r.stoppers = append(r.stoppers, stopper)
		}
	}
	return nil
}

func (r *Runtime) Stop() {
	for i := len(r.stoppers) - 1; i >= 0; i-- {
		stopper := r.stoppers[i]
		stopper()
	}
}
