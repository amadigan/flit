package flag

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type Flag struct {
	Name          string
	Forms         []string // x, -x, --extract
	Value         any
	ValueName     string
	ValueRequired bool
	Usage         string
	Last          bool // remaining options are arguments
}

type FlagOptions struct {
	Flags      []Flag
	Args       any
	CheckFlags bool // check arguments not starting with - as short flags (e.g. xvf)

	cached        sync.Once
	flags         map[string]*Flag
	hasRune       bool
	hasShortParam bool
	hasLongParam  bool
}

func (fo *FlagOptions) cache() {
	fo.cached.Do(func() {
		fo.flags = make(map[string]*Flag, len(fo.Flags))
		for i := range fo.Flags {

			f := &fo.Flags[i]
			for _, form := range f.Forms {
				if f.Value != nil {
					if len(form) == 1 || (len(form) == 2 && form[0] == '-') {
						fo.hasShortParam = true
					} else if len(form) > 2 && form[0:2] == "--" {
						fo.hasLongParam = true
					}
				}
				fo.flags[form] = f
			}
		}
	})
}

func (fo *FlagOptions) ParseArguments(args []string) error {
	if fo.Args == nil {
		return ExtraArgumentError{Args: args}
	}

	argsValue := reflect.ValueOf(fo.Args)

	if argsValue.Kind() == reflect.Pointer {
		// make a slice of the underlying type and fill it with the arguments
		elemType := argsValue.Elem().Type()
		sliceValue := reflect.MakeSlice(reflect.SliceOf(elemType), len(args), len(args))
		for i, arg := range args {
			if err := resolve(arg, sliceValue.Index(i).Addr().Interface()); err != nil {
				return err
			}
		}
		argsValue.Elem().Set(sliceValue)
	} else if argsValue.Kind() == reflect.Slice {
		// the slice must already exist, so we just fill it with the arguments
		if argsValue.Len() < len(args) {
			return ExtraArgumentError{Args: args[argsValue.Len():]}
		}

		for i, arg := range args {
			if err := resolve(arg, argsValue.Index(i).Addr().Interface()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (fo *FlagOptions) Parse(args []string) ([]*Flag, error) {
	fo.cache()

	if len(fo.Flags) == 0 {
		return nil, fo.ParseArguments(args)
	}

	hasArgs := fo.Args != nil

	var remaining []string
	var flags []*Flag

	for i := 0; i < len(args); i++ {
		if args[i] == "" {
			if hasArgs {
				remaining = append(remaining, args[i])

				continue
			} else {
				return flags, ExtraArgumentError{Args: []string{args[i]}}
			}
		}

		if args[i] == "--" {
			remaining = append(remaining, args[i+1:]...)
			break
		}

		var param string

		flag := fo.flags[args[i]] // handles 'x', '-x', '--extract'
		if flag == nil && fo.hasLongParam && strings.HasPrefix(args[i], "--") {
			// long parameter
			if idx := strings.Index(args[i], "="); idx > 0 {
				flag = fo.flags[args[i][:idx]]
				param = args[i][idx+1:]
			}
		}

		if flag == nil && strings.HasPrefix(args[i], "-") && len(args[i]) > 1 {
			// check a string starting with - as a list of flags (-xvf)
			var subflags []*Flag
			var valid bool

			if flag, subflags, param, valid = fo.readShortFlags(args[i][1:]); valid {
				flags = append(flags, subflags...)

				if flag == nil {
					continue
				}
			}
		}

		if flag == nil && (!hasArgs || fo.CheckFlags) {
			// check for 'xvf' style flags

			var subflags []*Flag
			var valid bool

			if flag, subflags, param, valid = fo.readShortFlags(args[i]); valid {
				flags = append(flags, subflags...)

				if flag == nil {
					continue
				}
			}
		}

		if flag == nil {
			if hasArgs {
				remaining = append(remaining, args[i])
				continue
			}

			return flags, InvalidParameterError{Flag: nil, Form: args[i], Arg: args[i], Err: fmt.Errorf("unrecognized flag")}
		}

		flags = append(flags, flag)

		if flag.Value != nil {
			if param != "" {
				if err := resolve(param, flag.Value); err != nil {
					return flags, InvalidParameterError{Flag: flag, Form: args[i], Arg: param, Err: err}
				}
			} else if len(args) > i+1 {
				if err := resolve(args[i+1], flag.Value); err != nil {
					return flags, InvalidParameterError{Flag: flag, Form: args[i], Arg: args[i+1], Err: err}
				}
				i++
			} else if flag.ValueRequired {
				return flags, MissingParameterError{Flag: flag, Form: args[i]}
			}
		}

		if flag.Last {
			remaining = append(remaining, args[i+1:]...)
			break
		}
	}

	return flags, fo.ParseArguments(remaining)
}

func (fo *FlagOptions) readShortFlags(flstr string) (*Flag, []*Flag, string, bool) {
	var flags []*Flag

	for j := 0; j < len(flstr); j++ {
		subflag := fo.flags[flstr[j:j+1]]
		if subflag == nil {
			return nil, nil, "", false
		}

		if subflag.Value != nil {
			return subflag, flags, flstr[j+1:], true
		}

		flags = append(flags, subflag)
	}

	return nil, flags, "", true
}

func resolve(arg string, value any) error {
	if value == nil {
		panic("cannot resolve to a nil value")
	}

	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Pointer {
		return fmt.Errorf("value must be a pointer")
	}

	elem := v.Elem()
	if !elem.CanSet() {
		return fmt.Errorf("value must be a settable pointer")
	}

	switch elem.Kind() {
	case reflect.String:
		elem.SetString(arg)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer: %v", err)
		}
		elem.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer: %v", err)
		}
		elem.SetUint(u)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(arg, 64)
		if err != nil {
			return fmt.Errorf("invalid float: %v", err)
		}
		elem.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(arg)
		if err != nil {
			return fmt.Errorf("invalid boolean: %v", err)
		}
		elem.SetBool(b)
	default:
		return fmt.Errorf("unsupported value type: %s", elem.Kind())
	}

	return nil
}
