package target

import (
	"debug/gosym"
	"go/ast"

	"github.com/derekparker/delve/pkg/proc"
)

// Target represents the target of the debugger. This
// target could be a system process, core file, etc.
type Interface interface {
	Info
	ProcessManipulation
	BreakpointManipulation
}

// Info is an interface that provides general information on the target.
type Info interface {
	Pid() int
	Exited() bool
	Running() bool
	BinInfo() *proc.BinaryInfo

	ThreadInfo
	GoroutineInfo

	// FindFileLocation returns the address of the first instruction belonging
	// to line lineNumber in file fileName.
	FindFileLocation(fileName string, lineNumber int) (uint64, error)

	// FirstPCAfterPrologue returns the first instruction address after fn's
	// prologue.
	// If sameline is true and the first instruction after the prologue belongs
	// to a different source line the entry point will be returned instead.
	FirstPCAfterPrologue(fn *gosym.Func, sameline bool) (uint64, error)

	// FindFunctionLocation finds address of a function's line
	//
	// If firstLine == true is passed FindFunctionLocation will attempt to find
	// the first line of the function.
	//
	// If lineOffset is passed FindFunctionLocation will return the address of
	// that line.
	//
	// Pass lineOffset == 0 and firstLine == false if you want the address for
	// the function's entry point. Note that setting breakpoints at that
	// address will cause surprising behavior:
	// https://github.com/derekparker/delve/issues/170
	FindFunctionLocation(funcName string, firstLine bool, lineOffset int) (uint64, error)
}

// ThreadInfo is an interface for getting information on active threads
// in the process.
type ThreadInfo interface {
	FindThread(threadID int) (proc.IThread, bool)
	ThreadList() []proc.IThread
	CurrentThread() proc.IThread
}

// GoroutineInfo is an interface for getting information on running goroutines.
type GoroutineInfo interface {
	SelectedGoroutine() *proc.G
}

// ProcessManipulation is an interface for changing the execution state of a process.
type ProcessManipulation interface {
	ContinueOnce() (trapthread proc.IThread, err error)
	StepInstruction() error
	SwitchThread(int) error
	SwitchGoroutine(int) error
	RequestManualStop() error
	Halt() error
	Kill() error
	Detach(bool) error
}

// BreakpointManipulation is an interface for managing breakpoints.
type BreakpointManipulation interface {
	Breakpoints() map[uint64]*proc.Breakpoint
	SetBreakpoint(addr uint64, kind proc.BreakpointKind, cond ast.Expr) (*proc.Breakpoint, error)
	ClearBreakpoint(addr uint64) (*proc.Breakpoint, error)
	ClearInternalBreakpoints() error
}

// VariableEval is an interface for dealing with eval scopes.
type VariableEval interface {
	FrameToScope(proc.Stackframe) *proc.EvalScope
	ConvertEvalScope(gid, frame int) (*proc.EvalScope, error)
}

var _ Interface = &proc.Process{}
var _ Interface = &proc.CoreProcess{}
var _ Interface = &proc.GdbserverProcess{}
