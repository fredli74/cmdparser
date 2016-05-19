// Copyright 2015-2016 Fredrik Lidström. All rights reserved.
// Use of this source code is governed by the standard MIT License (MIT)
// that can be found in the LICENSE file.

// Package cmdparser implements a function based command parser for golang
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

// Flags to commandline options
const (
	Standard   = 1 << iota // Standard option
	Preference             // Preference option that is saved and loaded with options file
	Required               // Required option
	Hidden                 // Hidden option not shown in help
)

// Args array will contain all arguments that were not parsed
var Args []string

// Title sets the text to be printed at the top of help. Setting title also enables the -version flag that will display the Title string.
var Title string

// OptionsFile sets the filename (with full path) where to load and save preference options. Setting OptionsFile enables -showoptions and -saveoptions flags.
var OptionsFile string

var commandName string        // Name of command to use in Usage instructions
var commandList []*CmdCommand // Internal list of all commands
var optionList []*CmdOption   // Internal list of all options

func init() {
	commandName = filepath.Base(os.Args[0])
}

/************************************* Core Functions  *************************************/

// Usage will display the full commandline help message. This function is automatically called when the -h, -H or -? flag is specified.
// Help text is automatically generated from available commands and options
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
		fmt.Printf("\n  -saveoptions\n        Save (*) options to %s\n", OptionsFile)
		fmt.Printf("  -showoptions\n        Show saved options\n")
	}
	if Title != "" {
		fmt.Println("  -version\n        Show current version")
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

// Parse takes the full commandline and parse it according to options and commands that has been setup.
// This is the core handler that will call underlying command functions
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
		} else if !stopParsing && (os.Args[i] == "-version") {
			if Title != "" {
				fmt.Println(Title)
			} else {
				fmt.Println("No title has been set")
			}
			return nil
		} else if !stopParsing && os.Args[i] == "--" {
			stopParsing = true
		} else if !stopParsing && OptionsFile != "" && os.Args[i] == "-saveoptions" {
			doSave = true
		} else if !stopParsing && OptionsFile != "" && os.Args[i] == "-showoptions" {
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
				if (len(Args) > 1 && c.Command == Args[1]) || c.Command == "" {
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

// CmdCommand is returned by the Command function and holds the full definition for a command 
type CmdCommand struct {
	Command  string 	// Name of the command
	Help     string 	// Help text to be displayed next to the command in Usage:
	Function func() 	// Underlying function to be called when command is specified on commandline
}

// Command adds a command to the parser with the specified name, help text and function pointer.
func Command(cmd string, help string, function func()) *CmdCommand {
	c := CmdCommand{Command: cmd, Help: help, Function: function}
	commandList = append(commandList, &c)
	return &c
}

/************************************* Options *************************************/

// CmdOption is returned by each *Option support function and holds the full definition of a command option
type CmdOption struct {
	Name     string 	// Name of option
	Group    string 	// Blank for global options or name of command for command specific options
	Format   string 	// A string explaining the accepted format like "<number>" or "<ip>:<port>"
	Help     string 	// Help text that describes the option
	Value    optionValue 	// Current value of the option
	Default  string // Defaul value if option is not specified
	Flags    int // Special option flags 
	onChange func() 	// function hook called when value changes
	onSave   func() 	// function hook called before saving (encrypting passwords for example)
}

type optionValue interface {
	String() string 	// Get current option in text format
	Reset()				// Reset the option to default
	Get() interface{} 	// Get the native value
	Set(string) error 	// Set the native value from string
}

// OnChange is a hook called when an option value has been set
// This can be used to convert option values
//   var sizeMB int64
//   var byteSize int64
//   cmdparse.IntOption("size", "", "<MiB>", "Set size", &sizeMB, cmdparse.Hidden|cmdparse.Preference).OnChange(func() {
//     byteSize = sizeMB * 1024 * 1024
//   })
//
// Or set related options
//
//   var accesskey []byte
//   var user string
//   var password string
//   cmdparse.ByteOption("accesskey", "", "", "Client accesskey", &accesskey, cmdparse.Preference|cmdparse.Hidden)
//   cmdparse.StringOption("user", "", "<username>", "Username", &user, cmdparse.Preference|cmdparse.Required)
//   cmdparse.StringOption("password", "", "<password>", "Password", &password, cmdparse.Standard).OnChange(func() {
//     accesskey := GenerateAccessKey(user, password)
//   }).OnSave(func() {
//     if user == "" {
//       panic(errors.New("Unable to save login unless both user and password options are specified"))
//     }
//   })
func (c *CmdOption) OnChange(f func()) *CmdOption {
	c.onChange = f
	return c
}

// OnSave is a hook called when an option value is about to be saved.
// See OnChange for usage example.
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

// BoolOption adds a bool option with the specified name, command group, help text, variable pointer and flags
// Boolean option uses the strconv.ParseBool function an accepts 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False. Any other value returns an error.
// Specifying a boolean option on commandline with no value is the same as true
//   var beVerbose bool
//   cmdparse.BoolOption("verbose", "", "Show verbose output", &beVerbose, cmdparse.Preference)
func BoolOption(name string, cmd string, help string, variable *bool, flags int) *CmdOption {
	return addOption(name, cmd, "", help, (*boolOption)(variable), flags)
}

// IntOption adds an integer option with the specified name, command group, help text, variable pointer and flags
// Integer options uses the 64 bit strconv.ParseInt function and accepts "0x" prefix for base 16, "0" prefix for base 8
// and uses base 10 otherwise. 
//   var size int64
//   cmdparse.IntOption("size", "truncate", "<MiB>", "Size to truncate to", &size, cmdparse.Preference|cmdparse.Required)
func IntOption(name string, cmd string, format string, help string, variable *int64, flags int) *CmdOption {
	return addOption(name, cmd, format, help, (*intOption)(variable), flags)
}

// FloatOption adds a float option with the specified name, command group, help text, variable pointer and flags
// Float options uses the 64 bit strconv.ParseFloat function and accepts a well-formed floating point number that is rounded using IEEE754 unbiased rounding.
//   var q float64
//   cmdparse.FloatOption("q", "", "<value>", "Sets the filter q value", &q, cmdparse.Standard)
func FloatOption(name string, cmd string, format string, help string, variable *float64, flags int) *CmdOption {
	return addOption(name, cmd, format, help, (*floatOption)(variable), flags)
}

// StringOption adds a string option with the specified name, command group, help text, variable pointer and flags
//   var serverAddr string
//   cmdparse.StringOption("server", "", "<ip>:<port>", "Server address", &serverAddr, cmdparse.Preference|cmdparse.Required)
func StringOption(name string, cmd string, format string, help string, variable *string, flags int) *CmdOption {
	return addOption(name, cmd, format, help, (*stringOption)(variable), flags)
}

// StringListOption adds a string list option with the specified name, command group, help text, variable pointer and flags
// Specifying a StringListOption on commandline will add that string to the internal list. There is no way to remove strings from the list from commandline except resetting the list by specifying an empty value.
// StringList options uses json.Unmarshal to format json type arrays when saving and loading to options file.
//   var IgnoreList []string
//   cmdparse.StringListOption("ignore", "copy", "<pattern>", "Ignore files matching pattern", &IgnoreList, cmdparse.Standard|cmdparse.Preference)
func StringListOption(name string, cmd string, format string, help string, variable *[]string, flags int) *CmdOption {
	return addOption(name, cmd, format, help, (*stringListOption)(variable), flags)
}

// ByteOption adds a byte option with the specified name, command group, help text, variable pointer and flags
// Byte options uses base64 standard encoding when specified on commandline and saving/loading from options file.
//   var accesskey []byte
//	 cmdparse.ByteOption("accesskey", "", "", "Client accesskey", &accesskey, cmdparse.Preference|cmdparse.Hidden)
func ByteOption(name string, cmd string, format string, help string, variable *[]byte, flags int) *CmdOption {
	return addOption(name, cmd, format, help, (*byteOption)(variable), flags)
}
