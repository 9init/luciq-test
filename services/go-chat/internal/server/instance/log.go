package instance

import (
	"errors"
	"fmt"
)

func (inst *Instance) Log(format string, a ...interface{}) {
	if inst != nil {
		inst.Logger.Info(format, a...)
	}
}

// Err log and return the error - useful for cases when we want to log a certain error message then return it
func (inst *Instance) Err(format string, a ...any) error {
	var err error
	if len(a) == 0 {
		err = errors.New(format)
	} else {
		err = fmt.Errorf(format, a...)
	}
	inst.Logger.Error("%s", err.Error())
	return err
}
