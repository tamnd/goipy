package vm

import (
	"bytes"
	"encoding/binary"
	"os"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildWave() *object.Module {
	m := &object.Module{Name: "wave", Dict: object.NewDict()}

	waveErrCls := &object.Class{Name: "Error", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}

	waveParamsCls := &object.Class{Name: "_wave_params", Dict: object.NewDict()}
	waveReadCls := &object.Class{Name: "Wave_read", Dict: object.NewDict()}
	waveWriteCls := &object.Class{Name: "Wave_write", Dict: object.NewDict()}

	makeWaveParams := func(nch, sw, fr, nf int, ct, cn string) *object.Instance {
		inst := &object.Instance{Class: waveParamsCls, Dict: object.NewDict()}
		inst.Dict.SetStr("nchannels", object.NewInt(int64(nch)))
		inst.Dict.SetStr("sampwidth", object.NewInt(int64(sw)))
		inst.Dict.SetStr("framerate", object.NewInt(int64(fr)))
		inst.Dict.SetStr("nframes", object.NewInt(int64(nf)))
		inst.Dict.SetStr("comptype", &object.Str{V: ct})
		inst.Dict.SetStr("compname", &object.Str{V: cn})
		return inst
	}

	// makeWaveRead creates a Wave_read instance from raw WAV file bytes.
	makeWaveRead := func(fileBytes []byte) (*object.Instance, error) {
		nch, sw, fr, nf, audioData, err := parseWAVBytes(fileBytes, waveErrCls)
		if err != nil {
			return nil, err
		}
		blockAlign := nch * sw
		if blockAlign == 0 {
			blockAlign = 1
		}

		framePos := 0
		closed := false

		inst := &object.Instance{Class: waveReadCls, Dict: object.NewDict()}

		doClose := func() {
			closed = true
		}

		inst.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			doClose()
			return object.None, nil
		}})
		inst.Dict.SetStr("getnchannels", &object.BuiltinFunc{Name: "getnchannels", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(nch)), nil
		}})
		inst.Dict.SetStr("getsampwidth", &object.BuiltinFunc{Name: "getsampwidth", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(sw)), nil
		}})
		inst.Dict.SetStr("getframerate", &object.BuiltinFunc{Name: "getframerate", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(fr)), nil
		}})
		inst.Dict.SetStr("getnframes", &object.BuiltinFunc{Name: "getnframes", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(nf)), nil
		}})
		inst.Dict.SetStr("getcomptype", &object.BuiltinFunc{Name: "getcomptype", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "NONE"}, nil
		}})
		inst.Dict.SetStr("getcompname", &object.BuiltinFunc{Name: "getcompname", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "not compressed"}, nil
		}})
		inst.Dict.SetStr("getparams", &object.BuiltinFunc{Name: "getparams", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return makeWaveParams(nch, sw, fr, nf, "NONE", "not compressed"), nil
		}})
		inst.Dict.SetStr("readframes", &object.BuiltinFunc{Name: "readframes", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if closed {
				return nil, object.Errorf(waveErrCls, "readframes() called on closed file")
			}
			n := nf
			if len(a) >= 1 {
				if v, ok := toInt64(a[0]); ok {
					n = int(v)
				}
			}
			start := framePos * blockAlign
			end := start + n*blockAlign
			if end > len(audioData) {
				end = len(audioData)
			}
			if start > len(audioData) {
				start = len(audioData)
			}
			chunk := append([]byte(nil), audioData[start:end]...)
			framePos += (end - start) / blockAlign
			return &object.Bytes{V: chunk}, nil
		}})
		inst.Dict.SetStr("tell", &object.BuiltinFunc{Name: "tell", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(framePos)), nil
		}})
		inst.Dict.SetStr("rewind", &object.BuiltinFunc{Name: "rewind", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			framePos = 0
			return object.None, nil
		}})
		inst.Dict.SetStr("setpos", &object.BuiltinFunc{Name: "setpos", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(waveErrCls, "setpos() requires 1 argument")
			}
			v, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "setpos() argument must be int")
			}
			pos := int(v)
			if pos < 0 || pos > nf {
				return nil, object.Errorf(waveErrCls, "setpos: position out of range")
			}
			framePos = pos
			return object.None, nil
		}})
		inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return inst, nil
		}})
		inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			doClose()
			return object.False, nil
		}})

		return inst, nil
	}

	// makeWaveWrite creates a Wave_write instance that buffers frames.
	makeWaveWrite := func(fileObj object.Object) *object.Instance {
		nch := 0
		sw := 0
		fr := 0
		comptype := "NONE"
		compname := "not compressed"
		var frames []byte
		closed := false

		inst := &object.Instance{Class: waveWriteCls, Dict: object.NewDict()}

		blockAlign := func() int {
			if nch > 0 && sw > 0 {
				return nch * sw
			}
			return 1
		}

		doClose := func() error {
			if closed {
				return nil
			}
			closed = true
			wavBytes := buildWAVBytes(nch, sw, fr, frames)
			return waveWriteToFile(fileObj, wavBytes)
		}

		inst.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if err := doClose(); err != nil {
				return nil, err
			}
			return object.None, nil
		}})
		inst.Dict.SetStr("setnchannels", &object.BuiltinFunc{Name: "setnchannels", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				if v, ok := toInt64(a[0]); ok {
					nch = int(v)
				}
			}
			return object.None, nil
		}})
		inst.Dict.SetStr("setsampwidth", &object.BuiltinFunc{Name: "setsampwidth", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				if v, ok := toInt64(a[0]); ok {
					sw = int(v)
				}
			}
			return object.None, nil
		}})
		inst.Dict.SetStr("setframerate", &object.BuiltinFunc{Name: "setframerate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				if v, ok := toInt64(a[0]); ok {
					fr = int(v)
				}
			}
			return object.None, nil
		}})
		inst.Dict.SetStr("setnframes", &object.BuiltinFunc{Name: "setnframes", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			// hint only; actual count derived from bytes written
			return object.None, nil
		}})
		inst.Dict.SetStr("setcomptype", &object.BuiltinFunc{Name: "setcomptype", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				if s, ok := a[0].(*object.Str); ok {
					comptype = s.V
				}
			}
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					compname = s.V
				}
			}
			_ = comptype
			_ = compname
			return object.None, nil
		}})
		inst.Dict.SetStr("setparams", &object.BuiltinFunc{Name: "setparams", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(waveErrCls, "setparams() requires a tuple")
			}
			var elems []object.Object
			switch v := a[0].(type) {
			case *object.Tuple:
				elems = v.V
			case *object.List:
				elems = v.V
			default:
				return nil, object.Errorf(waveErrCls, "setparams() argument must be a tuple")
			}
			if len(elems) < 6 {
				return nil, object.Errorf(waveErrCls, "setparams() tuple must have 6 elements")
			}
			if v, ok := toInt64(elems[0]); ok {
				nch = int(v)
			}
			if v, ok := toInt64(elems[1]); ok {
				sw = int(v)
			}
			if v, ok := toInt64(elems[2]); ok {
				fr = int(v)
			}
			if s, ok := elems[4].(*object.Str); ok {
				comptype = s.V
			}
			if s, ok := elems[5].(*object.Str); ok {
				compname = s.V
			}
			_ = comptype
			_ = compname
			return object.None, nil
		}})
		inst.Dict.SetStr("tell", &object.BuiltinFunc{Name: "tell", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ba := blockAlign()
			if ba == 0 {
				return object.NewInt(0), nil
			}
			return object.NewInt(int64(len(frames) / ba)), nil
		}})
		writeData := func(data []byte) {
			frames = append(frames, data...)
		}
		inst.Dict.SetStr("writeframes", &object.BuiltinFunc{Name: "writeframes", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			switch v := a[0].(type) {
			case *object.Bytes:
				writeData(v.V)
			case *object.Bytearray:
				writeData(v.V)
			}
			return object.None, nil
		}})
		inst.Dict.SetStr("writeframesraw", &object.BuiltinFunc{Name: "writeframesraw", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			switch v := a[0].(type) {
			case *object.Bytes:
				writeData(v.V)
			case *object.Bytearray:
				writeData(v.V)
			}
			return object.None, nil
		}})
		inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return inst, nil
		}})
		inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if err := doClose(); err != nil {
				return nil, err
			}
			return object.False, nil
		}})

		return inst
	}

	// open(file, mode=None)
	m.Dict.SetStr("open", &object.BuiltinFunc{Name: "open", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(waveErrCls, "open() requires at least 1 argument")
		}
		fileArg := a[0]

		mode := ""
		if len(a) >= 2 && a[1] != object.None {
			if s, ok := a[1].(*object.Str); ok {
				mode = s.V
			}
		}
		if kw != nil {
			if mv, ok := kw.GetStr("mode"); ok && mv != object.None {
				if s, ok2 := mv.(*object.Str); ok2 {
					mode = s.V
				}
			}
		}

		// If mode is empty, auto-detect.
		if mode == "" {
			switch fileArg.(type) {
			case *object.BytesIO:
				mode = "rb"
			case *object.File:
				mode = "rb"
			case *object.Str:
				mode = "rb"
			default:
				mode = "rb"
			}
		}

		switch mode {
		case "r", "rb":
			fileBytes, err := waveReadAllBytes(fileArg)
			if err != nil {
				return nil, object.Errorf(waveErrCls, "%v", err)
			}
			return makeWaveRead(fileBytes)
		case "w", "wb":
			// For string filename, open actual file.
			if s, ok := fileArg.(*object.Str); ok {
				f, err := os.Create(s.V)
				if err != nil {
					return nil, object.Errorf(waveErrCls, "cannot open file: %v", err)
				}
				fo := &object.File{F: f, FilePath: s.V, Mode: "wb", Binary: true}
				return makeWaveWrite(fo), nil
			}
			return makeWaveWrite(fileArg), nil
		default:
			return nil, object.Errorf(waveErrCls, "mode must be 'r', 'rb', 'w', or 'wb'")
		}
	}})

	m.Dict.SetStr("Error", waveErrCls)
	m.Dict.SetStr("Wave_read", waveReadCls)
	m.Dict.SetStr("Wave_write", waveWriteCls)

	return m
}

// waveReadAllBytes reads all bytes from a Python file-like object.
func waveReadAllBytes(fileObj object.Object) ([]byte, error) {
	switch v := fileObj.(type) {
	case *object.BytesIO:
		if v.Pos >= len(v.V) {
			return nil, nil
		}
		return append([]byte(nil), v.V[v.Pos:]...), nil
	case *object.File:
		f, ok := v.F.(*os.File)
		if !ok {
			return nil, nil
		}
		pos, _ := f.Seek(0, 1)
		data, err := os.ReadFile(f.Name())
		if err != nil {
			return nil, err
		}
		if pos > 0 && int(pos) < len(data) {
			return data[pos:], nil
		}
		return data, nil
	default:
		return nil, nil
	}
}

// waveWriteToFile writes WAV bytes to the Python file object.
func waveWriteToFile(fileObj object.Object, data []byte) error {
	switch v := fileObj.(type) {
	case *object.BytesIO:
		v.V = append(v.V[:0], data...)
		v.Pos = len(v.V)
	case *object.File:
		f, ok := v.F.(*os.File)
		if !ok {
			return nil
		}
		f.Seek(0, 0)
		f.Write(data)
	}
	return nil
}

// parseWAVBytes parses a RIFF/WAV byte slice and returns audio parameters.
func parseWAVBytes(data []byte, errCls *object.Class) (nch, sw, fr, nf int, audioData []byte, err error) {
	if len(data) < 12 {
		err = object.Errorf(errCls, "file too short")
		return
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		err = object.Errorf(errCls, "file does not start with RIFF id")
		return
	}

	pos := 12
	gotFmt := false
	for pos+8 <= len(data) {
		chunkID := string(data[pos : pos+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		dataStart := pos + 8
		dataEnd := dataStart + chunkSize
		if dataEnd > len(data) {
			dataEnd = len(data)
		}
		chunkBody := data[dataStart:dataEnd]

		switch chunkID {
		case "fmt ":
			if len(chunkBody) < 16 {
				err = object.Errorf(errCls, "fmt chunk too short")
				return
			}
			audioFormat := binary.LittleEndian.Uint16(chunkBody[0:2])
			if audioFormat != 1 {
				err = object.Errorf(errCls, "only PCM format (type 1) is supported")
				return
			}
			nch = int(binary.LittleEndian.Uint16(chunkBody[2:4]))
			fr = int(binary.LittleEndian.Uint32(chunkBody[4:8]))
			bitsPerSample := int(binary.LittleEndian.Uint16(chunkBody[14:16]))
			sw = bitsPerSample / 8
			gotFmt = true
		case "data":
			if !gotFmt {
				err = object.Errorf(errCls, "data chunk before fmt chunk")
				return
			}
			audioData = append([]byte(nil), chunkBody...)
			blockAlign := nch * sw
			if blockAlign > 0 {
				nf = len(audioData) / blockAlign
			}
		}

		pos = dataStart + chunkSize
		if chunkSize%2 != 0 {
			pos++ // RIFF pad byte
		}
	}

	if !gotFmt {
		err = object.Errorf(errCls, "missing fmt chunk")
		return
	}
	return
}

// buildWAVBytes constructs a complete RIFF/WAV file in memory.
func buildWAVBytes(nch, sw, fr int, audioData []byte) []byte {
	if nch == 0 {
		nch = 1
	}
	if sw == 0 {
		sw = 2
	}
	if fr == 0 {
		fr = 44100
	}
	dataSize := len(audioData)
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // PCM
	binary.Write(&buf, binary.LittleEndian, uint16(nch))
	binary.Write(&buf, binary.LittleEndian, uint32(fr))
	byteRate := fr * nch * sw
	binary.Write(&buf, binary.LittleEndian, uint32(byteRate))
	blockAlign := nch * sw
	binary.Write(&buf, binary.LittleEndian, uint16(blockAlign))
	binary.Write(&buf, binary.LittleEndian, uint16(sw*8))
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(dataSize))
	buf.Write(audioData)
	return buf.Bytes()
}
