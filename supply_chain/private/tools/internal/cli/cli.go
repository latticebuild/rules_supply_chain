package cli

import (
	"fmt"
	"os"
	"strings"
)

func Main(name string, run func([]string) (int, error)) {
	code, err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, name+":", err)
		code = 1
	}
	os.Exit(code)
}
func Parse(args []string, required []string, repeated ...string) (map[string][]string, error) {
	values := map[string][]string{}
	permitted := map[string]bool{}
	repeat := map[string]bool{}
	for _, key := range required {
		permitted[key] = true
	}
	for _, key := range repeated {
		permitted[key], repeat[key] = true, true
	}
	for i := 0; i < len(args); i++ {
		key, value, inline := strings.Cut(args[i], "=")
		if !permitted[key] {
			return nil, fmt.Errorf("unexpected argument %s", args[i])
		}
		if len(values[key]) > 0 && !repeat[key] {
			return nil, fmt.Errorf("argument %s repeated", key)
		}
		if !inline {
			i++
			if i == len(args) || strings.HasPrefix(args[i], "--") {
				return nil, fmt.Errorf("missing value for %s", key)
			}
			value = args[i]
		}
		values[key] = append(values[key], value)
	}
	for _, key := range required {
		if len(values[key]) == 0 {
			return nil, fmt.Errorf("missing required argument %s", key)
		}
	}
	return values, nil
}
