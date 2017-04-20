package proc

import (
	"debug/gosym"
	"errors"
	"fmt"
	"go/ast"
	"io"
	"sync"
)

// A SplicedMemory represents a memory space formed from multiple regions,
// each of which may override previously regions. For example, in the following
// core, the program text was loaded at 0x400000:
// Start               End                 Page Offset
// 0x0000000000400000  0x000000000044f000  0x0000000000000000
// but then it's partially overwritten with an RW mapping whose data is stored
// in the core file:
// Type           Offset             VirtAddr           PhysAddr
//                FileSiz            MemSiz              Flags  Align
// LOAD           0x0000000000004000 0x000000000049a000 0x0000000000000000
//                0x0000000000002000 0x0000000000002000  RW     1000
// This can be represented in a SplicedMemory by adding the original region,
// then putting the RW mapping on top of it.
type SplicedMemory struct {
	readers []readerEntry
}

type readerEntry struct {
	offset uintptr
	length uintptr
	reader MemoryReader
}

// Add adds a new region to the SplicedMemory, which may override existing regions.
func (r *SplicedMemory) Add(reader MemoryReader, off, length uintptr) {
	if length == 0 {
		return
	}
	end := off + length - 1
	newReaders := make([]readerEntry, 0, len(r.readers))
	add := func(e readerEntry) {
		if e.length == 0 {
			return
		}
		newReaders = append(newReaders, e)
	}
	inserted := false
	// Walk through the list of regions, fixing up any that overlap and inserting the new one.
	for _, entry := range r.readers {
		entryEnd := entry.offset + entry.length - 1
		switch {
		case entryEnd < off:
			// Entry is completely before the new region.
			add(entry)
		case end < entry.offset:
			// Entry is completely after the new region.
			if !inserted {
				add(readerEntry{off, length, reader})
				inserted = true
			}
			add(entry)
		case off <= entry.offset && entryEnd <= end:
			// Entry is completely overwritten by the new region. Drop.
		case entry.offset < off && entryEnd <= end:
			// New region overwrites the end of the entry.
			entry.length = off - entry.offset
			add(entry)
		case off <= entry.offset && end < entryEnd:
			// New reader overwrites the beginning of the entry.
			if !inserted {
				add(readerEntry{off, length, reader})
				inserted = true
			}
			overlap := entry.offset - off
			entry.offset += overlap
			entry.length -= overlap
			add(entry)
		case entry.offset < off && end < entryEnd:
			// New region punches a hole in the entry. Split it in two and put the new region in the middle.
			add(readerEntry{entry.offset, off - entry.offset, entry.reader})
			add(readerEntry{off, length, reader})
			add(readerEntry{end + 1, entryEnd - end, entry.reader})
			inserted = true
		default:
			panic(fmt.Sprintf("Unhandled case: existing entry is %v len %v, new is %v len %v", entry.offset, entry.length, off, length))
		}
	}
	if !inserted {
		newReaders = append(newReaders, readerEntry{off, length, reader})
	}
	r.readers = newReaders
}

// ReadMemory implements MemoryReader.ReadMemory.
func (r *SplicedMemory) ReadMemory(buf []byte, addr uintptr) (n int, err error) {
	started := false
	for _, entry := range r.readers {
		if entry.offset+entry.length < addr {
			if !started {
				continue
			}
			return n, fmt.Errorf("hit unmapped area at %v after %v bytes", addr, n)
		}

		// Don't go past the region.
		pb := buf
		if addr+uintptr(len(buf)) > entry.offset+entry.length {
			pb = pb[:entry.offset+entry.length-addr]
		}
		pn, err := entry.reader.ReadMemory(pb, addr)
		n += pn
		if err != nil || pn != len(pb) {
			return n, err
		}
		buf = buf[pn:]
		addr += uintptr(pn)
		if len(buf) == 0 {
			// Done, don't bother scanning the rest.
			return n, nil
		}
	}
	if n == 0 {
		return 0, fmt.Errorf("offset %v did not match any regions", addr)
	}
	return n, nil
}

// OffsetReaderAt wraps a ReaderAt into a MemoryReader, subtracting a fixed
// offset from the address. This is useful to represent a mapping in an address
// space. For example, if program text is mapped in at 0x400000, an
// OffsetReaderAt with offset 0x400000 can be wrapped around file.Open(program)
// to return the results of a read in that part of the address space.
type OffsetReaderAt struct {
	reader io.ReaderAt
	offset uintptr
}

func (r *OffsetReaderAt) ReadMemory(buf []byte, addr uintptr) (n int, err error) {
	return r.reader.ReadAt(buf, int64(addr-r.offset))
}

type CoreProcess struct {
	bi                BinaryInfo
	core              *Core
	breakpoints       map[uint64]*Breakpoint
	currentThread     *LinuxPrStatus
	selectedGoroutine *G
	allGCache         []*G
}

type CoreThread struct {
	th *LinuxPrStatus
	p  *CoreProcess
}

var ErrWriteCore = errors.New("can not to core process")
var ErrShortRead = errors.New("short read")
var ErrContinueCore = errors.New("can not continue execution of core process")

func OpenCore(corePath, exePath string) (*CoreProcess, error) {
	core, err := readCore(corePath, exePath)
	if err != nil {
		return nil, err
	}
	p := &CoreProcess{
		core:        core,
		breakpoints: make(map[uint64]*Breakpoint),
		bi:          NewBinaryInfo("linux", "amd64"),
	}

	var wg sync.WaitGroup
	p.bi.LoadBinaryInfo(exePath, &wg)
	wg.Wait()

	for _, th := range p.core.Threads {
		p.currentThread = th
		break
	}

	scope := &EvalScope{0, 0, p.CurrentThread(), nil, &p.bi}
	ver, isextld, err := scope.getGoInformation()
	if err != nil {
		return nil, err
	}

	p.bi.arch.SetGStructOffset(ver, isextld)
	p.selectedGoroutine, _ = GetG(p.CurrentThread())

	return p, nil
}

func (p *CoreProcess) BinInfo() *BinaryInfo {
	return &p.bi
}

func (thread *CoreThread) ReadMemory(data []byte, addr uintptr) (n int, err error) {
	n, err = thread.p.core.ReadMemory(data, addr)
	if err == nil && n != len(data) {
		err = ErrShortRead
	}
	return n, err
}

func (thread *CoreThread) writeMemory(addr uintptr, data []byte) (int, error) {
	return 0, ErrWriteCore
}

func (t *CoreThread) Location() (*Location, error) {
	f, l, fn := t.p.bi.PCToLine(t.th.Reg.Rip)
	return &Location{PC: t.th.Reg.Rip, File: f, Line: l, Fn: fn}, nil
}

func (t *CoreThread) Breakpoint() (*Breakpoint, bool, error) {
	return nil, false, nil
}

func (t *CoreThread) ThreadID() int {
	return int(t.th.Pid)
}

func (t *CoreThread) Registers(floatingPoint bool) (Registers, error) {
	//TODO(aarzilli): handle floating point registers
	return &t.th.Reg, nil
}

func (t *CoreThread) Arch() Arch {
	return t.p.bi.arch
}

func (t *CoreThread) BinInfo() *BinaryInfo {
	return &t.p.bi
}

func (t *CoreThread) StepInstruction() error {
	return ErrContinueCore
}

func (p *CoreProcess) Breakpoints() map[uint64]*Breakpoint {
	return p.breakpoints
}

func (p *CoreProcess) ClearBreakpoint(addr uint64) (*Breakpoint, error) {
	return nil, NoBreakpointError{addr: addr}
}

func (p *CoreProcess) ClearInternalBreakpoints() error {
	return nil
}

func (p *CoreProcess) ContinueOnce() (IThread, error) {
	return nil, ErrContinueCore
}

func (p *CoreProcess) StepInstruction() error {
	return ErrContinueCore
}

func (p *CoreProcess) RequestManualStop() error {
	return nil
}

func (p *CoreProcess) CurrentThread() IThread {
	return &CoreThread{p.currentThread, p}
}

func (p *CoreProcess) Detach(bool) error {
	return nil
}

func (p *CoreProcess) Exited() bool {
	return false
}

func (p *CoreProcess) FindFileLocation(fileName string, lineNumber int) (uint64, error) {
	return FindFileLocation(p.CurrentThread(), p.breakpoints, &p.bi, fileName, lineNumber)
}

func (p *CoreProcess) FirstPCAfterPrologue(fn *gosym.Func, sameline bool) (uint64, error) {
	return FirstPCAfterPrologue(p.CurrentThread(), p.breakpoints, &p.bi, fn, sameline)
}

func (p *CoreProcess) FindFunctionLocation(funcName string, firstLine bool, lineOffset int) (uint64, error) {
	return FindFunctionLocation(p.CurrentThread(), p.breakpoints, &p.bi, funcName, firstLine, lineOffset)
}

func (p *CoreProcess) AllGCache() *[]*G {
	return &p.allGCache
}

func (p *CoreProcess) Halt() error {
	return nil
}

func (p *CoreProcess) Kill() error {
	return nil
}

func (p *CoreProcess) Pid() int {
	return p.core.Pid
}

func (p *CoreProcess) Running() bool {
	return false
}

func (p *CoreProcess) SelectedGoroutine() *G {
	return p.selectedGoroutine
}

func (p *CoreProcess) SetBreakpoint(addr uint64, kind BreakpointKind, cond ast.Expr) (*Breakpoint, error) {
	return nil, ErrWriteCore
}

func (p *CoreProcess) SwitchGoroutine(gid int) error {
	g, err := FindGoroutine(p, gid)
	if err != nil {
		return err
	}
	if g == nil {
		// user specified -1 and selectedGoroutine is nil
		return nil
	}
	if g.thread != nil {
		return p.SwitchThread(g.thread.ThreadID())
	}
	p.selectedGoroutine = g
	return nil
}

func (p *CoreProcess) SwitchThread(tid int) error {
	if th, ok := p.core.Threads[tid]; ok {
		p.currentThread = th
		p.selectedGoroutine, _ = GetG(p.CurrentThread())
		return nil
	}
	return fmt.Errorf("thread %d does not exist", tid)
}

func (p *CoreProcess) ThreadList() []IThread {
	r := make([]IThread, 0, len(p.core.Threads))
	for _, v := range p.core.Threads {
		r = append(r, &CoreThread{v, p})
	}
	return r
}

func (p *CoreProcess) FindThread(threadID int) (IThread, bool) {
	t, ok := p.core.Threads[threadID]
	return &CoreThread{t, p}, ok
}
