package ui

import (
	"fmt"

	"github.com/charmbracelet/huh/spinner"
)

// RunWithSpinner executes an action while showing a spinner with the given title.
// If the action returns an error, the spinner stops and the error is returned.
func RunWithSpinner(title string, action func() error) error {
	var actionErr error
	err := spinner.New().
		Title(title).
		Action(func() {
			actionErr = action()
		}).
		Run()
	if err != nil {
		return err
	}
	return actionErr
}

// RunWithSpinnerf is like RunWithSpinner but accepts a format string.
func RunWithSpinnerf(format string, action func() error, args ...interface{}) error {
	return RunWithSpinner(fmt.Sprintf(format, args...), action)
}
