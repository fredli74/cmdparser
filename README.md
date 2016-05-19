# cmdparser #

Package cmdparser implements a function based command parser for golang

[![Build Status](https://semaphoreci.com/api/v1/fredli74/cmdparser/branches/master/badge.svg)](https://semaphoreci.com/fredli74/cmdparser)

## Usage

	import "github.com/fredli74/cmdparser"

### Example
```go
	// Enable -version flag by setting command title
	cmdparser.Title = fmt.Sprintf("Application %s", Version)

	// Enable -saveoptions and -showoptions by setting name of options file
	cmdparser.OptionsFile = filepath.Join(cmdparser.UserHomeFolder(), "options.json")

	// Setup a bool option
	var beVerbose bool
	cmdparse.BoolOption("verbose", "", "Show verbose output", &beVerbose, cmdparse.Preference)

	// Define the default command
	cmdparse.Command("", "", func() { // Default
		fmt.Printf("Verbose option is %v\n", beVerbose)
	})

	// 
	err = cmdparse.Parse()
	if err != nil {
		panic(err)
	}
```

### Reference

```go
const (
	Standard   = 1 << iota // Standard option
	Preference             // Preference option that is saved and loaded with options file
	Required               // Required option
	Hidden                 // Hidden option not shown in help
)
```
Flags to commandline options

```go
var Args []string
```
Args array will contain all arguments that were not parsed

```go
var OptionsFile string
```
OptionsFile sets the filename (with full path) where to load and save preference
options. Setting OptionsFile enables -showoptions and -saveoptions flags.

```go
var Title string
```
Title sets the text to be printed at the top of help. Setting title also enables
the -version flag that will display the Title string.

#### func  Parse

```go
func Parse() error
```
Parse takes the full commandline and parse it according to options and commands
that has been setup. This is the core handler that will call underlying command
functions

#### func  Usage

```go
func Usage()
```
Usage will display the full commandline help message. This function is
automatically called when the -h, -H or -? flag is specified. Help text is
automatically generated from available commands and options

#### func  UserHomeFolder

```go
func UserHomeFolder() string
```
UserHomeFolder is a simple cross-platform function to retrieve the users home
path using environment variables.

#### type CmdCommand

```go
type CmdCommand struct {
	Command  string // Name of the command
	Help     string // Help text to be displayed next to the command in Usage:
	Function func() // Underlying function to be called when command is specified on commandline
}
```

CmdCommand is returned by the Command function and holds the full definition for
a command

#### func  Command

```go
func Command(cmd string, help string, function func()) *CmdCommand
```
Command adds a command to the parser with the specified name, help text and
function pointer.

#### type CmdOption

```go
type CmdOption struct {
	Name    string      // Name of option
	Group   string      // Blank for global options or name of command for command specific options
	Format  string      // A string explaining the accepted format like "<number>" or "<ip>:<port>"
	Help    string      // Help text that describes the option
	Value   optionValue // Current value of the option
	Default string      // Defaul value if option is not specified
	Flags   int         // Special option flags
}
```

CmdOption is returned by each *Option support function and holds the full
definition of a command option

#### func  BoolOption

```go
func BoolOption(name string, cmd string, help string, variable *bool, flags int) *CmdOption
```
BoolOption adds a bool option with the specified name, command group, help text,
variable pointer and flags Boolean option uses the strconv.ParseBool function an
accepts 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False. Any other value
returns an error. Specifying a boolean option on commandline with no value is
the same as true
```go
	var beVerbose bool
	cmdparse.BoolOption("verbose", "", "Show verbose output", &beVerbose, cmdparse.Preference)
```
#### func  ByteOption

```go
func ByteOption(name string, cmd string, format string, help string, variable *[]byte, flags int) *CmdOption
```
ByteOption adds a byte option with the specified name, command group, help text,
variable pointer and flags Byte options uses base64 standard encoding when
specified on commandline and saving/loading from options file.
```go
	var accesskey []byte
	cmdparse.ByteOption("accesskey", "", "", "Client accesskey", &accesskey, cmdparse.Preference|cmdparse.Hidden)
```
#### func  FloatOption

```go
func FloatOption(name string, cmd string, format string, help string, variable *float64, flags int) *CmdOption
```
FloatOption adds a float option with the specified name, command group, help
text, variable pointer and flags Float options uses the 64 bit
strconv.ParseFloat function and accepts a well-formed floating point number that
is rounded using IEEE754 unbiased rounding.
```go
	var q float64
	cmdparse.FloatOption("q", "", "<value>", "Sets the filter q value", &q, cmdparse.Standard)
```
#### func  IntOption

```go
func IntOption(name string, cmd string, format string, help string, variable *int64, flags int) *CmdOption
```
IntOption adds an integer option with the specified name, command group, help
text, variable pointer and flags Integer options uses the 64 bit
strconv.ParseInt function and accepts "0x" prefix for base 16, "0" prefix for
base 8 and uses base 10 otherwise.
```go
	var size int64
	cmdparse.IntOption("size", "truncate", "<MiB>", "Size to truncate to", &size, cmdparse.Preference|cmdparse.Required)
```
#### func  StringListOption

```go
func StringListOption(name string, cmd string, format string, help string, variable *[]string, flags int) *CmdOption
```
StringListOption adds a string list option with the specified name, command
group, help text, variable pointer and flags Specifying a StringListOption on
commandline will add that string to the internal list. There is no way to remove
strings from the list from commandline except resetting the list by specifying
an empty value. StringList options uses json.Unmarshal to format json type
arrays when saving and loading to options file.
```go
	var IgnoreList []string
	cmdparse.StringListOption("ignore", "copy", "<pattern>", "Ignore files matching pattern", &IgnoreList,
	cmdparse.Standard|cmdparse.Preference)
```
#### func  StringOption

```go
func StringOption(name string, cmd string, format string, help string, variable *string, flags int) *CmdOption
```
StringOption adds a string option with the specified name, command group, help
text, variable pointer and flags
```go
	var serverAddr string
	cmdparse.StringOption("server", "", "<ip>:<port>", "Server address", &serverAddr, cmdparse.Preference|cmdparse.Required)
```
#### func (*CmdOption) OnChange

```go
func (c *CmdOption) OnChange(f func()) *CmdOption
```
OnChange is a hook called when an option value has been set This can be used to
convert option values
```go
	var sizeMB int64
	var byteSize int64
	cmdparse.IntOption("size", "", "<MiB>", "Set size", &sizeMB, cmdparse.Hidden|cmdparse.Preference).OnChange(func() {
		byteSize = sizeMB * 1024 * 1024
	})
```
Or set related options
```go
	var accesskey []byte
	var user string
	var password string
	cmdparse.ByteOption("accesskey", "", "", "Client accesskey", &accesskey, cmdparse.Preference|cmdparse.Hidden)
	cmdparse.StringOption("user", "", "<username>", "Username", &user, cmdparse.Preference|cmdparse.Required)
	cmdparse.StringOption("password", "", "<password>", "Password", &password, cmdparse.Standard).OnChange(func() {
	accesskey := GenerateAccessKey(user, password)
	}).OnSave(func() {
		if user == "" {
			panic(errors.New("Unable to save login unless both user and password options are specified"))
		}
	})
```
#### func (*CmdOption) OnSave

```go
func (c *CmdOption) OnSave(f func()) *CmdOption
```
OnSave is a hook called when an option value is about to be saved. See OnChange
for usage example.
