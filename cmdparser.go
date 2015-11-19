// golang function based command parser
package cmdparser

import (
	"encoding/base64"
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

const (
	defaultCommand = ""
)

var Args []string
var Title string
var OptionsFile string

var commandName string
var commandList []*CmdCommand
var optionList []*CmdOption

func init() {
	commandName = filepath.Base(os.Args[0])
}

/************************************* Core Functions  *************************************/

func Usage() {
	if Title != "" {
		fmt.Printf(Title + "\n\n")
	}
	fmt.Println("Usage:")
	for _, n := range commandList {
		fmt.Printf("  %s [options] %s %s\n", commandName, n.Command, n.Help)
	}

	printOption := func(n *CmdOption) {
		fmt.Printf("  -%s", n.Name)
		if n.Format != "" {
			fmt.Printf("=%s", n.Format)
		}
		fmt.Printf("\n        %s", n.Help)
		if n.Flags&Preference > 0 {
			fmt.Printf(" (*)")
		}
		switch n.Value.(type) {
		case *boolOption:
			if n.Default == "true" {
				fmt.Printf(" (default ON)")
			}
		case *stringListOption:
			// Dont show it
		default:
			if n.Default != "" {
				fmt.Printf(" (default %s)", n.Default)
			}
		}
		fmt.Print("\n")
	}
	fmt.Println("\nOptions:")
	for _, n := range optionList {
		if n.Flags&Hidden == 0 && n.Group == "" {
			printOption(n)
		}
	}
	if OptionsFile != "" {
		fmt.Printf("\n  -save-options\n        Save (*) options to %s\n", OptionsFile)
		fmt.Printf("  -show-options\n        Show saved options\n")
	}
	fmt.Println()
	for _, g := range commandList {
		if g.Command != "" {
			var printedHeader bool
			for _, n := range optionList {
				if n.Flags&Hidden == 0 && n.Group == g.Command {
					if !printedHeader {
						fmt.Printf("%s options:\n", g.Command)
						printedHeader = true
					}
					printOption(n)
				}
			}
		}
	}

}

func Parse() error {
	if OptionsFile != "" {
		if err := loadOptions(OptionsFile); err != nil {
			return err
		}
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
			var option *CmdOption
			for _, o := range optionList {
				if o.Name == pair[0] {
					option = o
					break
				}
			}
			if option == nil {
				return errors.New("Invalid option -" + pair[0])
			}

			if len(pair) < 2 {
				switch option.Value.(type) {
				case *boolOption: // Special bool handling because a bool does not need a cmd line value
					if i < len(os.Args)-1 && os.Args[i+1][0] != '-' {
						if _, err := strconv.ParseBool(os.Args[i+1]); err != nil {
							pair = append(pair, "true")
						} else {
							i++
							pair = append(pair, os.Args[i])
						}
					} else {
						pair = append(pair, "true")
					}
				case *stringListOption:
					if i < len(os.Args)-1 && os.Args[i+1][0] != '-' {
						i++
						pair = append(pair, os.Args[i])
					} else {
						option.Value.(*stringListOption).FromString(option.Default)
					}
				default:
					if i < len(os.Args)-1 && os.Args[i+1][0] != '-' {
						i++
						pair = append(pair, os.Args[i])
					} else {
						pair = append(pair, option.Default)
					}
				}
			}

			if len(pair) == 2 {
				if pair[1] == "" {
					option.Value.Reset()
				} else {
					if err := option.Value.Set(pair[1]); err != nil {
						return errors.New(fmt.Sprintf("Invalid value set for option %s: \"%s\" (%s)", pair[0], pair[1], err.Error()))
					}
				}
				option.doChange()
			}
		} else {
			Args = append(Args, os.Args[i])
		}
	}

	if doShow {
		js, err := jsonOptions()
		if err != nil {
			return err
		}
		fmt.Println(string(js))
	} else {
		/*for _, n := range optionList {
			if (*n).Function != nil {
				(*n).Function()
			}
		}*/

		if doSave {
			_, err := saveOptions(OptionsFile)
			if err != nil {
				return err
			}
			fmt.Println("Options saved to " + OptionsFile)
		} else {
			var command *CmdCommand
			for _, c := range commandList {
				if (len(Args) > 1 && c.Command == Args[1]) || c.Command == defaultCommand {
					command = c
					if c.Command != "" {
						break
					}
				}
			}
			if command != nil {
				for _, n := range optionList {
					if n.Flags&Required > 0 && n.Value.String() == n.Default {
						return errors.New("Missing required option -" + n.Name)
					}
				}

				if command.Function != nil {
					command.Function()
				}

			} else {
				if len(Args) < 2 {
					Usage()
					return errors.New("Missing required command")
				} else {
					return errors.New(Args[1] + " is not a valid command")
				}
			}
		}
	}
	return nil
}

/************************************* Preferences Functions  *************************************/

func jsonOptions() ([]byte, error) {
	optionMap := make(map[string]interface{})
	for _, v := range optionList {
		if (v.Flags&Preference > 0) && v.Value.String() != v.Default {
			v.doSave()
			optionMap[v.Name] = v.Value.Get()
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
		return nil
	}
	defer file.Close()

	var optionMap map[string]interface{}
	if err := json.NewDecoder(file).Decode(&optionMap); err != nil {
		return err
	}
	for _, o := range optionList {
		if v, ok := optionMap[o.Name]; ok {
			switch t := v.(type) {
			case nil: // for JSON null
				o.Value.Reset()
			case map[string]interface{}: // for JSON objects
			// not implemented
			case []interface{}: // for JSON arrays
				o.Value.Reset()
				for _, s := range t {
					if err := o.Value.Set(s.(string)); err != nil {
						return err
					}
				}
				o.doChange()
			default:
				if err := o.Value.Set(fmt.Sprintf("%v", t)); err != nil {
					return err
				}
				o.doChange()
			}
		}
	}
	return nil
}

/************************************* Commands *************************************/

type CmdCommand struct {
	Command  string
	Help     string
	Function func()
}

func Command(cmd string, help string, function func()) *CmdCommand {
	c := CmdCommand{Command: cmd, Help: help, Function: function}
	commandList = append(commandList, &c)
	return &c
}

/************************************* Options *************************************/

type CmdOption struct {
	Name     string
	Group    string
	Format   string
	Help     string
	Value    optionValue
	Default  string
	Flags    int
	onChange func()
	onSave   func()
}

type optionValue interface {
	String() string
	Reset()
	Get() interface{}
	Set(string) error
}

func (c *CmdOption) OnChange(f func()) *CmdOption {
	c.onChange = f
	return c
}
func (c *CmdOption) OnSave(f func()) *CmdOption {
	c.onSave = f
	return c
}
func (c *CmdOption) doChange() {
	if c.onChange != nil {
		c.onChange()
	}
}
func (c *CmdOption) doSave() {
	if c.onSave != nil {
		c.onSave()
	}
}

type boolOption bool

func (b *boolOption) String() string   { return fmt.Sprintf("%v", *b) }
func (b *boolOption) Reset()           { *b = false }
func (b *boolOption) Get() interface{} { return bool(*b) }
func (b *boolOption) Set(s string) error {
	v, err := strconv.ParseBool(s)
	*b = boolOption(v)
	return err
}

type intOption int64

func (i *intOption) String() string   { return fmt.Sprintf("%v", *i) }
func (i *intOption) Reset()           { *i = 0 }
func (i *intOption) Get() interface{} { return int64(*i) }
func (i *intOption) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, 64)
	*i = intOption(v)
	return err
}

type floatOption float64

func (f *floatOption) String() string   { return fmt.Sprintf("%v", *f) }
func (f *floatOption) Reset()           { *f = 0 }
func (f *floatOption) Get() interface{} { return float64(*f) }
func (f *floatOption) Set(s string) error {
	v, err := strconv.ParseFloat(s, 64)
	*f = floatOption(v)
	return err
}

type stringOption string

func (s *stringOption) String() string   { return fmt.Sprintf("%s", *s) }
func (s *stringOption) Reset()           { *s = "" }
func (s *stringOption) Get() interface{} { return string(*s) }
func (s *stringOption) Set(v string) error {
	*s = stringOption(v)
	return nil
}

type stringListOption []string

func (s *stringListOption) String() string {
	if *s == nil {
		return ""
	} else {
		j, _ := json.Marshal([]string(*s))
		return string(j)
	}
}
func (s *stringListOption) FromString(v string) error {
	return json.Unmarshal([]byte(v), s)
}
func (s *stringListOption) Reset()           { *s = nil }
func (s *stringListOption) Get() interface{} { return []string(*s) }
func (s *stringListOption) Set(v string) error {
	*s = append(*s, v)
	return nil
}

type byteOption []byte

func (b *byteOption) String() string   { return base64.StdEncoding.EncodeToString(*b) }
func (b *byteOption) Reset()           { *b = nil }
func (b *byteOption) Get() interface{} { return []byte(*b) }
func (b *byteOption) Set(s string) error {
	v, err := base64.StdEncoding.DecodeString(s)
	*b = byteOption(v)
	return err
}

func addOption(name string, cmd string, format string, help string, variable optionValue, flags int) *CmdOption {
	o := CmdOption{Name: name, Group: cmd, Format: format, Help: help, Value: variable, Default: variable.String(), Flags: flags}
	optionList = append(optionList, &o)
	return &o
}

func BoolOption(name string, cmd string, help string, variable *bool, flags int) *CmdOption {
	return addOption(name, cmd, "", help, (*boolOption)(variable), flags)
}
func IntOption(name string, cmd string, format string, help string, variable *int64, flags int) *CmdOption {
	return addOption(name, cmd, format, help, (*intOption)(variable), flags)
}
func FloatOption(name string, cmd string, format string, help string, variable *float64, flags int) *CmdOption {
	return addOption(name, cmd, format, help, (*floatOption)(variable), flags)
}
func StringOption(name string, cmd string, format string, help string, variable *string, flags int) *CmdOption {
	return addOption(name, cmd, format, help, (*stringOption)(variable), flags)
}
func StringListOption(name string, cmd string, format string, help string, variable *[]string, flags int) *CmdOption {
	return addOption(name, cmd, format, help, (*stringListOption)(variable), flags)
}
func ByteOption(name string, cmd string, format string, help string, variable *[]byte, flags int) *CmdOption {
	return addOption(name, cmd, format, help, (*byteOption)(variable), flags)
}
