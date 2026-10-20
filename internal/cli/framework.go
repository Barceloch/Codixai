// Package cli implements the command line interface of Codixai.
//
// The package holds a small command framework and the command tree of the
// tool. The framework is intentionally minimal: it parses the arguments
// with the standard library, it renders a stable help text and it maps the
// errors to the exit codes of the specification.
//
// The command tree is defined in a single place (NewRootCommand) and every
// command declares its flags and its handler in one constructor. Replacing
// the framework with a third party one later is therefore a mechanical
// change: the tree, the flags and the handlers stay where they are.
//
// The package is the composition root of the CLI: it wires the use cases
// of the application layer with their providers and it renders the results.
// It never implements an analysis rule.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/barceloch/codixai/internal/cli/ui"
)

// The exit codes of the CLI. They are part of the contract of the tool.
//
//   - 0: the operation succeeded and no finding has to fail the command.
//   - 1: the command failed to run, or the configuration is invalid.
//   - 2: the arguments are invalid.
//   - 3: the audit completed and its findings reach the configured failure
//     threshold.
const (
	// ExitOK reports a successful operation without failing findings.
	ExitOK = 0
	// ExitError reports an execution or configuration error.
	ExitError = 1
	// ExitInvalidArgs reports invalid arguments.
	ExitInvalidArgs = 2
	// ExitFindings reports a completed audit whose findings reach the
	// configured failure threshold.
	ExitFindings = 3
)

// UsageError is the error of an invalid invocation of a command.
//
// The entry point maps it to the exit code 2.
type UsageError struct {
	// Message explains what is wrong and how to recover from it.
	Message string
	// Command is the command that received the invalid arguments.
	Command *Command
	// Err is the underlying cause, when there is one.
	Err error
}

// Error implements the error interface.
func (e *UsageError) Error() string {
	switch {
	case e.Message != "" && e.Err != nil:
		return e.Message + ": " + e.Err.Error()
	case e.Message != "":
		return e.Message
	case e.Err != nil:
		return e.Err.Error()
	default:
		return "invalid arguments"
	}
}

// Unwrap returns the underlying cause of the error.
func (e *UsageError) Unwrap() error { return e.Err }

// Usagef builds a UsageError with a formatted message.
func Usagef(format string, args ...any) *UsageError {
	return &UsageError{Message: fmt.Sprintf(format, args...)}
}

// FindingsError reports a completed audit whose findings reach the
// configured failure threshold.
//
// The entry point maps it to the exit code 3. The document of the analysis
// has already been written when the error is returned.
type FindingsError struct {
	// Count is the number of findings that reach the threshold.
	Count int
	// Threshold is the severity that makes the audit fail.
	Threshold string
}

// Error implements the error interface.
func (e *FindingsError) Error() string {
	return fmt.Sprintf("%d finding(s) reach the failure threshold (fail_on: %s)", e.Count, e.Threshold)
}

// Command is a node of the command tree.
//
// A command declares its metadata, its flags and its handler. The handler
// receives the arguments that are left after the flags have been parsed.
type Command struct {
	// Use is the usage line of the command. Its first word is the name of
	// the command.
	Use string
	// Short is the one line description of the command.
	Short string
	// Long is the full description of the command.
	Long string
	// Example holds the example invocations of the command.
	Example string
	// RunE is the handler of the command. A command without a handler
	// prints its help.
	RunE func(ctx context.Context, args []string) error
	// Flags registers the flags of the command.
	Flags func(fs *flag.FlagSet)
	// PersistentFlags names the flags that every command of the tree
	// accepts, wherever they appear in the invocation. The root command
	// declares the global flags of the tool with it.
	PersistentFlags []string
	// Hidden keeps the command out of the help of its parent.
	Hidden bool

	commands []*Command
	parent   *Command
	fs       *flag.FlagSet
}

// AddCommand adds a subcommand and returns the receiver.
func (c *Command) AddCommand(command *Command) *Command {
	command.parent = c
	c.commands = append(c.commands, command)
	return c
}

// Commands returns the subcommands of the command, in declaration order.
func (c *Command) Commands() []*Command { return c.commands }

// Name returns the name of the command, the first word of its usage line.
func (c *Command) Name() string {
	fields := strings.Fields(c.Use)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// Path returns the full invocation of the command, from the root.
func (c *Command) Path() string {
	if c.parent == nil {
		return c.Name()
	}
	return c.parent.Path() + " " + c.Name()
}

// Root returns the root of the command tree.
func (c *Command) Root() *Command {
	if c.parent == nil {
		return c
	}
	return c.parent.Root()
}

// lookup returns the subcommand with the given name.
func (c *Command) lookup(name string) *Command {
	for _, command := range c.commands {
		if command.Name() == name {
			return command
		}
	}
	return nil
}

// flagSet returns the flag set of the command.
//
// The flag set is built once and it always holds the help flag, so that
// every command answers to --help and -h.
func (c *Command) flagSet() *flag.FlagSet {
	if c.fs != nil {
		return c.fs
	}
	fs := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	if c.Flags != nil {
		c.Flags(fs)
	}
	if fs.Lookup("help") == nil {
		fs.Bool("help", false, "Print this help and exit.")
	}
	c.fs = fs
	return fs
}

// splitFlags splits the arguments into the flags of the command and the
// arguments that follow them.
//
// A flag that needs a value consumes the next argument, and the arguments
// that follow the first positional argument are left untouched: they
// belong to a subcommand or to the handler.
func (c *Command) splitFlags(args []string) (flagArgs []string, rest []string, err error) {
	fs := c.flagSet()
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if argument == "--" {
			return flagArgs, append(rest, args[index+1:]...), nil
		}
		if !isFlag(argument) {
			return flagArgs, append(rest, args[index:]...), nil
		}
		flagArgs = append(flagArgs, argument)
		if needsValue(fs, argument) {
			if index+1 >= len(args) {
				return nil, nil, &UsageError{
					Message: fmt.Sprintf("the flag %s needs a value", argument),
					Command: c,
				}
			}
			index++
			flagArgs = append(flagArgs, args[index])
		}
	}
	return flagArgs, rest, nil
}

// isFlag reports whether an argument is a flag.
func isFlag(argument string) bool {
	return len(argument) > 1 && strings.HasPrefix(argument, "-")
}

// extractPersistent separates the global flags from the other arguments.
//
// The global flags are accepted wherever they appear in the invocation.
// The arguments that are not global keep their order, so that the command
// tree still sees them in the order the user wrote them.
func (c *Command) extractPersistent(args []string) (persistent []string, rest []string, err error) {
	fs := c.flagSet()
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if argument == "--" {
			rest = append(rest, args[index:]...)
			return persistent, rest, nil
		}
		if !isFlag(argument) {
			rest = append(rest, argument)
			continue
		}
		name := strings.TrimLeft(argument, "-")
		if position := strings.Index(name, "="); position >= 0 {
			name = name[:position]
		}
		if !c.isPersistent(name) {
			rest = append(rest, argument)
			if needsValue(fs, argument) && index+1 < len(args) {
				index++
				rest = append(rest, args[index])
			}
			continue
		}
		persistent = append(persistent, argument)
		if needsValue(fs, argument) {
			if index+1 >= len(args) {
				return nil, nil, &UsageError{
					Message: fmt.Sprintf("the flag %s needs a value", argument),
					Command: c,
				}
			}
			index++
			persistent = append(persistent, args[index])
		}
	}
	return persistent, rest, nil
}

// isPersistent reports whether a flag is a global flag of the tool.
func (c *Command) isPersistent(name string) bool {
	for _, persistent := range c.PersistentFlags {
		if persistent == name {
			return true
		}
	}
	return false
}

// needsValue reports whether a flag consumes the next argument.
//
// A boolean flag never consumes a value, and a flag written with an equal
// sign carries its own value.
func needsValue(fs *flag.FlagSet, argument string) bool {
	name := strings.TrimLeft(argument, "-")
	if strings.Contains(name, "=") {
		return false
	}
	registered := fs.Lookup(name)
	if registered == nil {
		// The flag is unknown: the parser reports it with a better
		// message.
		return false
	}
	if boolean, ok := registered.Value.(interface{ IsBoolFlag() bool }); ok && boolean.IsBoolFlag() {
		return false
	}
	return true
}

// dispatch parses the arguments of the command and runs it.
func (c *Command) dispatch(ctx context.Context, args []string) error {
	if c.parent == nil && len(c.PersistentFlags) > 0 {
		// The global flags are accepted before and after the name of the
		// command, so that "codixai audit --format json" and
		// "codixai --format json audit" behave the same way.
		persistent, remaining, err := c.extractPersistent(args)
		if err != nil {
			return err
		}
		if err := c.flagSet().Parse(persistent); err != nil {
			return &UsageError{
				Message: fmt.Sprintf("invalid flag for %q", c.Path()),
				Command: c,
				Err:     err,
			}
		}
		args = remaining
	}
	flagArgs, rest, err := c.splitFlags(args)
	if err != nil {
		return err
	}
	fs := c.flagSet()
	if err := fs.Parse(flagArgs); err != nil {
		return &UsageError{
			Message: fmt.Sprintf("invalid flag for %q", c.Path()),
			Command: c,
			Err:     err,
		}
	}
	if help := fs.Lookup("help"); help != nil && help.Value.String() == "true" {
		c.printHelp(ctx)
		return nil
	}
	if c.parent == nil {
		// The global options are validated once, as soon as the flags of
		// the root command are parsed.
		if env := envFrom(ctx); env != nil {
			if err := env.validate(); err != nil {
				return err
			}
		}
	}
	if len(rest) > 0 {
		if subcommand := c.lookup(rest[0]); subcommand != nil {
			return subcommand.dispatch(ctx, rest[1:])
		}
		if len(c.commands) > 0 {
			return &UsageError{
				Message: fmt.Sprintf("unknown command %q", rest[0]),
				Command: c,
			}
		}
	}
	if c.RunE == nil {
		c.printHelp(ctx)
		return nil
	}
	return c.RunE(ctx, rest)
}

// printHelp writes the help of the command to the standard output.
func (c *Command) printHelp(ctx context.Context) {
	env := envFrom(ctx)
	if env == nil {
		return
	}
	c.Help(env.Stdout, env.Renderer())
}

// Help writes the help of the command.
//
// The layout is stable: the same invocation always produces the same
// sections, in the same order, so that an agent can parse it.
func (c *Command) Help(w io.Writer, r *ui.Renderer) {
	title := c.Path()
	if c.Short != "" {
		title = title + " - " + c.Short
	}
	fmt.Fprintln(w, r.Title(title))
	fmt.Fprintln(w)
	if c.Long != "" {
		for _, line := range strings.Split(strings.TrimRight(c.Long, "\n"), "\n") {
			fmt.Fprintln(w, line)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "Usage:")
	if len(c.commands) > 0 {
		fmt.Fprintf(w, "  %s <command> [flags]\n", c.Path())
	} else {
		fmt.Fprintf(w, "  %s [flags]\n", c.Path())
	}
	fmt.Fprintln(w)

	if visible := c.visibleCommands(); len(visible) > 0 {
		fmt.Fprintln(w, "Commands:")
		for _, command := range visible {
			fmt.Fprintf(w, "  %-10s %s\n", command.Name(), command.Short)
		}
		fmt.Fprintln(w)
	}

	if own := ownFlags(c); len(own) > 0 {
		fmt.Fprintln(w, "Flags:")
		for _, entry := range own {
			fmt.Fprintf(w, "  %-24s %s\n", entry.name, entry.usage)
		}
		fmt.Fprintln(w)
	}

	if global := globalFlags(c); len(global) > 0 {
		fmt.Fprintln(w, "Global flags:")
		for _, entry := range global {
			fmt.Fprintf(w, "  %-24s %s\n", entry.name, entry.usage)
		}
		fmt.Fprintln(w)
	}

	if c.Example != "" {
		fmt.Fprintln(w, "Examples:")
		for _, line := range strings.Split(strings.TrimRight(c.Example, "\n"), "\n") {
			fmt.Fprintln(w, "  "+line)
		}
		fmt.Fprintln(w)
	}
}

// visibleCommands returns the subcommands that appear in the help.
func (c *Command) visibleCommands() []*Command {
	visible := make([]*Command, 0, len(c.commands))
	for _, command := range c.commands {
		if !command.Hidden {
			visible = append(visible, command)
		}
	}
	return visible
}

// flagEntry is a flag of the help.
type flagEntry struct {
	// name is the invocation of the flag, with its placeholder.
	name string
	// usage is the description of the flag.
	usage string
}

// ownFlags returns the flags declared by the command itself.
func ownFlags(c *Command) []flagEntry {
	var entries []flagEntry
	c.flagSet().VisitAll(func(f *flag.Flag) {
		entries = append(entries, flagEntry{name: flagName(f), usage: f.Usage})
	})
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	return entries
}

// globalFlags returns the flags inherited from the ancestors of the
// command.
func globalFlags(c *Command) []flagEntry {
	var entries []flagEntry
	for ancestor := c.parent; ancestor != nil; ancestor = ancestor.parent {
		ancestor.flagSet().VisitAll(func(f *flag.Flag) {
			if f.Name == "help" || f.Name == "version" {
				return
			}
			entries = append(entries, flagEntry{name: flagName(f), usage: f.Usage})
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	return entries
}

// flagName renders the invocation of a flag.
func flagName(f *flag.Flag) string {
	if f.Name == "help" {
		return "-h, --help"
	}
	if boolean, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && boolean.IsBoolFlag() {
		return "--" + f.Name
	}
	return "--" + f.Name + " <value>"
}

// Execute runs the command line interface and returns its exit code.
//
// The arguments are the ones of the command line, without the name of the
// program. The standard output receives the document that the command
// produces and the standard error receives the diagnostics, the warnings
// and the errors.
func Execute(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	env := &Env{Stdout: stdout, Stderr: stderr}
	ctx = withEnv(ctx, env)

	root := NewRootCommand(env)
	err := root.dispatch(ctx, args)
	if err == nil {
		return ExitOK
	}

	var usage *UsageError
	if errors.As(err, &usage) {
		fmt.Fprintf(stderr, "codixai: %s\n", usage.Error())
		if usage.Command != nil {
			fmt.Fprintf(stderr, "Run '%s --help' for the usage of the command.\n", usage.Command.Path())
		}
		return ExitInvalidArgs
	}

	var findings *FindingsError
	if errors.As(err, &findings) {
		fmt.Fprintf(stderr, "codixai: %s\n", findings.Error())
		return ExitFindings
	}

	fmt.Fprintf(stderr, "codixai: error: %s\n", err.Error())
	return ExitError
}
