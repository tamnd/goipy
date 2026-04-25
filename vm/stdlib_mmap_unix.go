//go:build unix

package vm

import (
	"bytes"
	"sync"

	"github.com/tamnd/goipy/object"
	"golang.org/x/sys/unix"
)

const (
	mmapAccessDefault = 0
	mmapAccessRead    = 1
	mmapAccessWrite   = 2
	mmapAccessCopy    = 3
)

// mmapState is the backing store for a Python mmap instance.
type mmapState struct {
	mu     sync.RWMutex
	data   []byte
	pos    int
	access int
	closed bool
}

func (i *Interp) buildMmap() *object.Module {
	m := &object.Module{Name: "mmap", Dict: object.NewDict()}
	errCls := i.osErr

	setInt := func(name string, val int) {
		m.Dict.SetStr(name, object.NewInt(int64(val)))
	}

	// ── Constants ─────────────────────────────────────────────────────────────
	setInt("ACCESS_DEFAULT", mmapAccessDefault)
	setInt("ACCESS_READ", mmapAccessRead)
	setInt("ACCESS_WRITE", mmapAccessWrite)
	setInt("ACCESS_COPY", mmapAccessCopy)
	setInt("PROT_READ", unix.PROT_READ)
	setInt("PROT_WRITE", unix.PROT_WRITE)
	setInt("PROT_EXEC", unix.PROT_EXEC)
	setInt("MAP_SHARED", unix.MAP_SHARED)
	setInt("MAP_PRIVATE", unix.MAP_PRIVATE)
	setInt("MAP_ANON", unix.MAP_ANON)
	setInt("MAP_ANONYMOUS", unix.MAP_ANON)
	ps := unix.Getpagesize()
	setInt("PAGESIZE", ps)
	setInt("ALLOCATIONGRANULARITY", ps)

	// MADV constants (common subset)
	setInt("MADV_NORMAL", unix.MADV_NORMAL)
	setInt("MADV_RANDOM", unix.MADV_RANDOM)
	setInt("MADV_SEQUENTIAL", unix.MADV_SEQUENTIAL)
	setInt("MADV_WILLNEED", unix.MADV_WILLNEED)
	setInt("MADV_DONTNEED", unix.MADV_DONTNEED)
	setInt("MADV_FREE", int(unix.MADV_FREE))

	// ── mmap class ────────────────────────────────────────────────────────────
	mmapCls := &object.Class{Name: "mmap", Dict: object.NewDict()}
	m.Dict.SetStr("mmap", mmapCls)

	// ── mmap() constructor ────────────────────────────────────────────────────
	mmapCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// First arg is the instance (self).
			if len(a) < 3 {
				return nil, object.Errorf(i.typeErr, "mmap() requires fileno and length")
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return nil, object.Errorf(i.typeErr, "mmap.__init__: not an instance")
			}
			fileno, ok2 := toInt64(a[1])
			if !ok2 {
				return nil, object.Errorf(i.typeErr, "mmap() fileno must be int")
			}
			length, ok3 := toInt64(a[2])
			if !ok3 {
				return nil, object.Errorf(i.typeErr, "mmap() length must be int")
			}
			access := int64(mmapAccessDefault)
			offset := int64(0)
			if len(a) > 3 {
				if n, ok4 := toInt64(a[3]); ok4 {
					access = n
				}
			}
			if len(a) > 4 {
				if n, ok5 := toInt64(a[4]); ok5 {
					offset = n
				}
			}
			if kw != nil {
				if v, ok4 := kw.GetStr("access"); ok4 {
					if n, ok5 := toInt64(v); ok5 {
						access = n
					}
				}
				if v, ok4 := kw.GetStr("offset"); ok4 {
					if n, ok5 := toInt64(v); ok5 {
						offset = n
					}
				}
			}

			fd := int(fileno)
			mapLen := int(length)
			anon := fd == -1

			// If length==0, get the file size.
			if mapLen == 0 && !anon {
				var st unix.Stat_t
				if err := unix.Fstat(fd, &st); err != nil {
					return nil, object.Errorf(errCls, "mmap: fstat: %v", err)
				}
				mapLen = int(st.Size) - int(offset)
				if mapLen <= 0 {
					return nil, object.Errorf(errCls, "mmap: file is empty or offset past end")
				}
			}
			if mapLen == 0 {
				return nil, object.Errorf(errCls, "mmap: length must be > 0 for anonymous mapping")
			}

			// Determine prot and flags from access.
			prot := unix.PROT_READ | unix.PROT_WRITE
			flags := unix.MAP_SHARED
			switch access {
			case mmapAccessRead:
				prot = unix.PROT_READ
				flags = unix.MAP_SHARED
			case mmapAccessCopy:
				prot = unix.PROT_READ | unix.PROT_WRITE
				flags = unix.MAP_PRIVATE
			}
			if anon {
				flags = unix.MAP_ANONYMOUS | unix.MAP_PRIVATE
				if access == mmapAccessRead {
					prot = unix.PROT_READ
				}
			}

			data, err := unix.Mmap(fd, offset, mapLen, prot, flags)
			if err != nil {
				return nil, object.Errorf(errCls, "mmap: %v", err)
			}

			st := &mmapState{data: data, access: int(access)}
			mmapAttach(i, inst, st, errCls)
			return object.None, nil
		}})

	// Make mmap callable as mmap.mmap(fd, length, ...) — acts as a class
	// that creates an instance and calls __init__.
	mmapCallable := &object.BuiltinFunc{Name: "mmap",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := &object.Instance{Class: mmapCls, Dict: object.NewDict()}
			// Call __init__ with inst prepended.
			initArgs := append([]object.Object{inst}, a...)
			fn, ok := mmapCls.Dict.GetStr("__init__")
			if !ok {
				return nil, object.Errorf(i.typeErr, "mmap has no __init__")
			}
			if _, err := i.callObject(fn, initArgs, kw); err != nil {
				return nil, err
			}
			return inst, nil
		}}
	m.Dict.SetStr("mmap", mmapCallable)

	return m
}

// mmapAttach wires all mmap methods onto inst using st as backing state.
func mmapAttach(i *Interp, inst *object.Instance, st *mmapState, errCls *object.Class) {
	inst.Dict.SetStr("closed", object.False)

	checkOpen := func() error {
		st.mu.RLock()
		c := st.closed
		st.mu.RUnlock()
		if c {
			return object.Errorf(errCls, "mmap closed")
		}
		return nil
	}
	checkWrite := func() error {
		if st.access == mmapAccessRead {
			return object.Errorf(i.typeErr, "mmap can't modify a readonly memory map.")
		}
		return nil
	}

	// ── read([n]) ──────────────────────────────────────────────────────────────
	inst.Dict.SetStr("read", &object.BuiltinFunc{Name: "read",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			st.mu.Lock()
			defer st.mu.Unlock()
			n := len(st.data) - st.pos
			if len(a) > 0 && a[0] != object.None {
				if x, ok := toInt64(a[0]); ok && int(x) < n {
					n = int(x)
				}
			}
			if n < 0 {
				n = 0
			}
			out := make([]byte, n)
			copy(out, st.data[st.pos:st.pos+n])
			st.pos += n
			return &object.Bytes{V: out}, nil
		}})

	// ── write(data) ────────────────────────────────────────────────────────────
	inst.Dict.SetStr("write", &object.BuiltinFunc{Name: "write",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			if err := checkWrite(); err != nil {
				return nil, err
			}
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "write() requires data")
			}
			b := toBytes(a[0])
			st.mu.Lock()
			defer st.mu.Unlock()
			if st.pos+len(b) > len(st.data) {
				return nil, object.Errorf(errCls, "data out of range")
			}
			n := copy(st.data[st.pos:], b)
			st.pos += n
			return object.NewInt(int64(n)), nil
		}})

	// ── read_byte() ────────────────────────────────────────────────────────────
	inst.Dict.SetStr("read_byte", &object.BuiltinFunc{Name: "read_byte",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			st.mu.Lock()
			defer st.mu.Unlock()
			if st.pos >= len(st.data) {
				return nil, object.Errorf(i.valueErr, "read byte out of range")
			}
			b := st.data[st.pos]
			st.pos++
			return object.NewInt(int64(b)), nil
		}})

	// ── write_byte(byte) ───────────────────────────────────────────────────────
	inst.Dict.SetStr("write_byte", &object.BuiltinFunc{Name: "write_byte",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			if err := checkWrite(); err != nil {
				return nil, err
			}
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "write_byte() requires byte")
			}
			n, ok := toInt64(a[0])
			if !ok || n < 0 || n > 255 {
				return nil, object.Errorf(i.typeErr, "write_byte() argument must be in 0..255")
			}
			st.mu.Lock()
			defer st.mu.Unlock()
			if st.pos >= len(st.data) {
				return nil, object.Errorf(i.valueErr, "write byte out of range")
			}
			st.data[st.pos] = byte(n)
			st.pos++
			return object.None, nil
		}})

	// ── readline() ─────────────────────────────────────────────────────────────
	inst.Dict.SetStr("readline", &object.BuiltinFunc{Name: "readline",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			st.mu.Lock()
			defer st.mu.Unlock()
			rest := st.data[st.pos:]
			idx := bytes.IndexByte(rest, '\n')
			var line []byte
			if idx < 0 {
				line = make([]byte, len(rest))
				copy(line, rest)
				st.pos = len(st.data)
			} else {
				line = make([]byte, idx+1)
				copy(line, rest[:idx+1])
				st.pos += idx + 1
			}
			return &object.Bytes{V: line}, nil
		}})

	// ── seek(pos[, whence]) ────────────────────────────────────────────────────
	inst.Dict.SetStr("seek", &object.BuiltinFunc{Name: "seek",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "seek() requires pos")
			}
			pos, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "seek() pos must be int")
			}
			whence := int64(0)
			if len(a) > 1 {
				whence, _ = toInt64(a[1])
			}
			st.mu.Lock()
			defer st.mu.Unlock()
			newpos := int(pos)
			switch whence {
			case 0: // SEEK_SET
				newpos = int(pos)
			case 1: // SEEK_CUR
				newpos = st.pos + int(pos)
			case 2: // SEEK_END
				newpos = len(st.data) + int(pos)
			default:
				return nil, object.Errorf(i.valueErr, "invalid whence value")
			}
			if newpos < 0 {
				newpos = 0
			}
			if newpos > len(st.data) {
				newpos = len(st.data)
			}
			st.pos = newpos
			return object.NewInt(int64(newpos)), nil
		}})

	// ── tell() ─────────────────────────────────────────────────────────────────
	inst.Dict.SetStr("tell", &object.BuiltinFunc{Name: "tell",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			st.mu.RLock()
			pos := st.pos
			st.mu.RUnlock()
			return object.NewInt(int64(pos)), nil
		}})

	// ── size() ─────────────────────────────────────────────────────────────────
	inst.Dict.SetStr("size", &object.BuiltinFunc{Name: "size",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			st.mu.RLock()
			n := len(st.data)
			st.mu.RUnlock()
			return object.NewInt(int64(n)), nil
		}})

	// ── seekable() ────────────────────────────────────────────────────────────
	inst.Dict.SetStr("seekable", &object.BuiltinFunc{Name: "seekable",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}})

	// ── find(sub[, start[, end]]) ─────────────────────────────────────────────
	inst.Dict.SetStr("find", &object.BuiltinFunc{Name: "find",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "find() requires sub")
			}
			sub := toBytes(a[0])
			st.mu.RLock()
			data := st.data
			st.mu.RUnlock()
			start, end := 0, len(data)
			if len(a) > 1 {
				if n, ok := toInt64(a[1]); ok {
					start = int(n)
					if start < 0 {
						start = 0
					}
				}
			}
			if len(a) > 2 {
				if n, ok := toInt64(a[2]); ok {
					end = int(n)
					if end > len(data) {
						end = len(data)
					}
				}
			}
			if start > end {
				return object.NewInt(-1), nil
			}
			idx := bytes.Index(data[start:end], sub)
			if idx < 0 {
				return object.NewInt(-1), nil
			}
			return object.NewInt(int64(start + idx)), nil
		}})

	// ── rfind(sub[, start[, end]]) ────────────────────────────────────────────
	inst.Dict.SetStr("rfind", &object.BuiltinFunc{Name: "rfind",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "rfind() requires sub")
			}
			sub := toBytes(a[0])
			st.mu.RLock()
			data := st.data
			st.mu.RUnlock()
			start, end := 0, len(data)
			if len(a) > 1 {
				if n, ok := toInt64(a[1]); ok {
					start = int(n)
					if start < 0 {
						start = 0
					}
				}
			}
			if len(a) > 2 {
				if n, ok := toInt64(a[2]); ok {
					end = int(n)
					if end > len(data) {
						end = len(data)
					}
				}
			}
			if start > end {
				return object.NewInt(-1), nil
			}
			idx := bytes.LastIndex(data[start:end], sub)
			if idx < 0 {
				return object.NewInt(-1), nil
			}
			return object.NewInt(int64(start + idx)), nil
		}})

	// ── flush([offset, size]) ─────────────────────────────────────────────────
	inst.Dict.SetStr("flush", &object.BuiltinFunc{Name: "flush",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			st.mu.RLock()
			data := st.data
			st.mu.RUnlock()
			region := data
			if len(a) >= 2 {
				off, _ := toInt64(a[0])
				sz, _ := toInt64(a[1])
				if int(off+sz) <= len(data) {
					region = data[off : off+sz]
				}
			}
			if err := unix.Msync(region, unix.MS_SYNC); err != nil {
				return nil, object.Errorf(errCls, "flush: %v", err)
			}
			return object.NewInt(0), nil
		}})

	// ── madvise(option[, start, length]) ─────────────────────────────────────
	inst.Dict.SetStr("madvise", &object.BuiltinFunc{Name: "madvise",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "madvise() requires option")
			}
			option, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "madvise() option must be int")
			}
			st.mu.RLock()
			data := st.data
			st.mu.RUnlock()
			region := data
			if len(a) >= 3 {
				start, _ := toInt64(a[1])
				length, _ := toInt64(a[2])
				end := int(start + length)
				if end > len(data) {
					end = len(data)
				}
				region = data[start:end]
			}
			if err := unix.Madvise(region, int(option)); err != nil {
				return nil, object.Errorf(errCls, "madvise: %v", err)
			}
			return object.None, nil
		}})

	// ── move(dest, src, count) ────────────────────────────────────────────────
	inst.Dict.SetStr("move", &object.BuiltinFunc{Name: "move",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			if err := checkWrite(); err != nil {
				return nil, err
			}
			if len(a) < 3 {
				return nil, object.Errorf(i.typeErr, "move() requires dest, src, count")
			}
			dest, _ := toInt64(a[0])
			src, _ := toInt64(a[1])
			count, _ := toInt64(a[2])
			st.mu.Lock()
			defer st.mu.Unlock()
			if int(src+count) > len(st.data) || int(dest+count) > len(st.data) {
				return nil, object.Errorf(errCls, "move: out of range")
			}
			copy(st.data[dest:], st.data[src:src+count])
			return object.None, nil
		}})

	// ── resize(newsize) ───────────────────────────────────────────────────────
	inst.Dict.SetStr("resize", &object.BuiltinFunc{Name: "resize",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "resize() requires newsize")
			}
			newsize, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "resize() newsize must be int")
			}
			st.mu.Lock()
			defer st.mu.Unlock()
			newData, err := mmapResize(st.data, int(newsize))
			if err != nil {
				return nil, object.Errorf(i.systemErr, "mmap: resizing not available--no mremap()")
			}
			st.data = newData
			if st.pos > len(st.data) {
				st.pos = len(st.data)
			}
			return object.None, nil
		}})

	// ── close() ───────────────────────────────────────────────────────────────
	doClose := func() {
		st.mu.Lock()
		defer st.mu.Unlock()
		if !st.closed {
			unix.Munmap(st.data) //nolint
			st.data = nil
			st.closed = true
			inst.Dict.SetStr("closed", object.True)
		}
	}
	inst.Dict.SetStr("close", &object.BuiltinFunc{Name: "close",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			doClose()
			return object.None, nil
		}})

	// ── __enter__ / __exit__ ──────────────────────────────────────────────────
	inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return inst, nil
		}})
	inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			doClose()
			return object.False, nil
		}})

	// ── __len__ ───────────────────────────────────────────────────────────────
	inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			st.mu.RLock()
			n := len(st.data)
			st.mu.RUnlock()
			return object.NewInt(int64(n)), nil
		}})

	// ── __getitem__ ───────────────────────────────────────────────────────────
	inst.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "__getitem__ requires key")
			}
			st.mu.RLock()
			data := st.data
			n := len(data)
			st.mu.RUnlock()
			if sl, ok := a[0].(*object.Slice); ok {
				start, stop, step, err := i.resolveSlice(sl, n)
				if err != nil {
					return nil, err
				}
				if step == 1 {
					if start > stop {
						start = stop
					}
					out := make([]byte, stop-start)
					copy(out, data[start:stop])
					return &object.Bytes{V: out}, nil
				}
				// stepped slice
				var out []byte
				for idx := start; idx < stop; idx += step {
					out = append(out, data[idx])
				}
				return &object.Bytes{V: out}, nil
			}
			idx, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "mmap indices must be integers")
			}
			if idx < 0 {
				idx += int64(n)
			}
			if idx < 0 || int(idx) >= n {
				return nil, object.Errorf(i.indexErr, "mmap index out of range")
			}
			return object.NewInt(int64(data[idx])), nil
		}})

	// ── __setitem__ ───────────────────────────────────────────────────────────
	inst.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			if err := checkWrite(); err != nil {
				return nil, err
			}
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "__setitem__ requires key and value")
			}
			st.mu.Lock()
			defer st.mu.Unlock()
			n := len(st.data)
			if sl, ok := a[0].(*object.Slice); ok {
				start, stop, _, err := i.resolveSlice(sl, n)
				if err != nil {
					return nil, err
				}
				val := toBytes(a[1])
				if len(val) != stop-start {
					return nil, object.Errorf(i.valueErr, "mmap slice assignment is wrong size")
				}
				copy(st.data[start:stop], val)
				return object.None, nil
			}
			idx, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "mmap indices must be integers")
			}
			if idx < 0 {
				idx += int64(n)
			}
			if idx < 0 || int(idx) >= n {
				return nil, object.Errorf(i.indexErr, "mmap assignment index out of range")
			}
			bval, ok2 := toInt64(a[1])
			if !ok2 || bval < 0 || bval > 255 {
				return nil, object.Errorf(i.typeErr, "mmap item value must be in 0..255")
			}
			st.data[idx] = byte(bval)
			return object.None, nil
		}})
}

// toBytes extracts a byte slice from a Python bytes/bytearray/str object.
func toBytes(obj object.Object) []byte {
	switch v := obj.(type) {
	case *object.Bytes:
		return v.V
	case *object.Bytearray:
		return v.V
	case *object.Str:
		return []byte(v.V)
	}
	return nil
}
