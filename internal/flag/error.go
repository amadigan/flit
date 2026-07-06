package flag

import "fmt"

type ExtraArgumentError struct {
	Args []string
}

func (e ExtraArgumentError) Error() string {
	return fmt.Sprintf("extra arguments: %v", e.Args)
}

type MissingParameterError struct {
	Flag *Flag
	Form string
}

func (e MissingParameterError) Error() string {
	return fmt.Sprintf("missing %s for flag %s", e.Form, e.Flag.ValueName)
}

type InvalidParameterError struct {
	Flag *Flag
	Form string
	Arg  string
	Err  error
}

func (e InvalidParameterError) Error() string {
	return fmt.Sprintf("invalid %s for flag %s: %s: %v", e.Form, e.Flag.ValueName, e.Arg, e.Err)
}

func (e InvalidParameterError) Unwrap() error {
	return e.Err
}
