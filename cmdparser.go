package cmdparser

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	Standard = 1 << iota
	Preference
	Required
	Hidden
)

var Option map[string]*CmdOption
var Command map[string]*CmdCommand
var Args []string
var Title string
var OptionsFile string
var RequireCommand bool
var SaveOptions func(map[string]interface{}) error

var commandName string
var commandList []CmdCommand
var optionList []CmdOption

func init() {
	Option = make(map[string]*CmdOption)
	Command = make(map[string]*CmdCommand)
	commandName = filepath.Base(os.Args[0])
}

func AddCommand(cmd string, help string, function func()) {
	commandList = append(commandList, CmdCommand{Command: cmd, Help: help, Function: function})
	Command[cmd] = &commandList[len(commandList)-1]
}

func AddOption(name string, cmd string, help string, defaultValue interface{}, flags int) {
	optionList = append(optionList, CmdOption{Name: name, Group: cmd, Help: help, Value: defaultValue, Default: defaultValue, Flags: flags})
	Option[name] = &optionList[len(optionList)-1]
}

func Usage() {
	if Title != "" {
		fmt.Printf(Title + "\n\n")
	}
	fmt.Println("Usage:")
	for _, n := range commandList {
		fmt.Printf("  %s [options] %s %s\n", commandName, n.Command, n.Help)
	}
	fmt.Println("\nOptions:")
	for _, n := range optionList {
		if n.Flags&Hidden == 0 {
			fmt.Printf("  -%s\n        %s", n.Name, n.Help)
			if n.Flags&Preference > 0 {
				fmt.Printf(" (*)")
			}
			if n.DefaultString() != "" {
				fmt.Printf(" (default %s)", n.DefaultString())
			}
			fmt.Print("\n")
		}
	}
	if OptionsFile != "" {
		fmt.Printf("  -save-options\n        Save (*) options to %s\n", OptionsFile)
	}
}

func Parse() error {
	if OptionsFile != "" {
		loadOptions(OptionsFile)
	}

	var stopParsing bool
	var doSave bool
	var doShow bool
	Args = nil
	for i := 0; i < len(os.Args); i++ {
		if !stopParsing && (os.Args[i] == "-?" || os.Args[i] == "-h" || os.Args[i] == "-H") {
			Usage()
			return nil
		} else if !stopParsing && os.Args[i] == "--" {
			stopParsing = true
		} else if !stopParsing && os.Args[i] == "-save-options" {
			doSave = true
		} else if !stopParsing && os.Args[i] == "-show-options" {
			doShow = true
		} else if !stopParsing && os.Args[i][0] == '-' {
			pair := strings.Split(os.Args[i][1:], "=")
			option := Option[pair[0]]
			if option == nil {
				return errors.New("Invalid option -" + pair[0])
			}
			switch option.Value.(type) {
			case bool:
				if len(pair) < 2 && i < len(os.Args)-1 && os.Args[i+1][0] != '-' {
					_, err := strconv.ParseBool(os.Args[i+1])
					if err == nil {
						i++
						pair = append(pair, os.Args[i])
					}
				}
				if len(pair) == 2 {
					b, err := strconv.ParseBool(pair[1])
					if err != nil {
						return errors.New("Invalid value set for option " + pair[0] + ": \"" + pair[1] + "\" is not a boolean")
					}
					option.Value = b
				} else {
					option.Value = true
				}
			default:
				if len(pair) < 2 && i < len(os.Args)-1 && os.Args[i+1][0] != '-' {
					i++
					pair = append(pair, os.Args[i])
				}
				switch option.Value.(type) {
				case bool:
					if len(pair) == 2 {
						b, err := strconv.ParseBool(pair[1])
						if err != nil {
							return errors.New("Invalid value set for option " + pair[0] + ": \"" + pair[1] + "\" is not a boolean")
						}
						option.Value = b
					} else {
						option.Value = true
					}
				case int:
					if len(pair) == 2 {
						i, err := strconv.ParseInt(pair[1], 10, 32)
						if err != nil {
							return errors.New("Invalid value set for option " + pair[0] + ": \"" + pair[1] + "\" is not a valid integer")
						}
						option.Value = i
					} else {
						option.Value = option.Default
					}
				case string:
					if len(pair) == 2 {
						option.Value = pair[1]
					} else {
						option.Value = option.Default
					}
				case []string:
					if len(pair) == 2 {
						if option.FromPref {
							option.Value = make([]string, 0)
						}
						option.Value = append(option.Value.([]string), pair[1])
					} else {
						option.Value = option.Default
					}
				default:
					return errors.New("Option value type not supported")
				}
			}
			option.FromPref = false
		} else {
			Args = append(Args, os.Args[i])
		}
	}

	if doSave {
		_, err := saveOptions(OptionsFile)
		if err != nil {
			return err
		}
		fmt.Println("Options saved to " + OptionsFile)
	} else if doShow {
		js, err := jsonOptions()
		if err != nil {
			return err
		}
		fmt.Println(string(js))
	} else {

		if Command[""] == nil {
			if len(Args) < 2 {
				Usage()
				return errors.New("Missing required command")
			} else if Command[Args[1]] == nil {
				return errors.New(Args[1] + " is not a valid command")
			}
		}

		for _, n := range Option {
			if n.Flags&Required > 0 && !n.IsSet() {
				return errors.New("Missing required option -" + n.Name)
			}
		}

		if len(Args) > 1 && Command[Args[1]] != nil && Command[Args[1]].Function != nil {
			Command[Args[1]].Function()
		} else if Command[""] != nil && Command[""].Function != nil {
			Command[""].Function()
		}
	}
	return nil
}

func jsonOptions() ([]byte, error) {
	optionMap := make(map[string]interface{})
	for _, v := range Option {
		if (v.Flags&Preference > 0) && v.String() != v.DefaultString() {
			optionMap[v.Name] = v.Value
		}
	}

	if SaveOptions != nil {
		err := SaveOptions(optionMap)
		if err != nil {
			return nil, err
		}
	}

	jsonData, err := json.MarshalIndent(optionMap, "", "\t")
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func saveOptions(name string) (string, error) {
	jsonData, err := jsonOptions()

	err = os.MkdirAll(filepath.Dir(name), 0700)
	if err != nil {
		return "", err
	}
	file, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0700)
	if err != nil {
		return "", err
	}
	defer file.Close()
	file.Write(jsonData)
	return string(jsonData), nil
}

func loadOptions(name string) error {
	file, err := os.OpenFile(name, os.O_RDONLY, 0700)
	if err != nil {
		return err
	}
	defer file.Close()

	var optionMap map[string]interface{}
	if err := json.NewDecoder(file).Decode(&optionMap); err != nil {
		return err
	}

	for n, v := range optionMap {
		o := Option[n]
		if o != nil {
			switch t := v.(type) {
			case nil: // for JSON null
			// skip
			case map[string]interface{}: // for JSON objects
			// not implemented
			case []interface{}: // for JSON arrays
				o.Value = make([]string, 0)
				for _, s := range t {
					// TODO: add different array types
					o.Value = append(o.Value.([]string), s.(string))
				}
			case string: // for JSON strings
				o.Value = t
			case float64: // for JSON numbers
				o.Value = int(t)
			case bool: // for JSON booleans
				o.Value = t
			}
			o.FromPref = true
		}
	}
	return nil
}

type CmdCommand struct {
	Command  string
	Help     string
	Function func()
	order    int
}

type CmdOption struct {
	Name     string
	Group    string
	Help     string
	Value    interface{}
	Default  interface{}
	FromPref bool
	Flags    int

	order int
}

func (c *CmdOption) String() string {
	return fmt.Sprintf("%v", c.Value)
}
func (c *CmdOption) DefaultString() string {
	return fmt.Sprintf("%v", c.Default)
}
func (c *CmdOption) IsSet() bool {
	return c.String() != c.DefaultString()
}
func (c *CmdOption) Set(val interface{}) {
	c.Value = val
	c.FromPref = false
}
