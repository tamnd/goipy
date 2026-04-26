import wave
import io


def test_write_and_read_basic():
    buf = io.BytesIO()
    w = wave.open(buf, 'wb')
    w.setnchannels(1)
    w.setsampwidth(2)
    w.setframerate(44100)
    w.writeframes(b'\x00\x01' * 100)
    w.close()

    buf.seek(0)
    r = wave.open(buf, 'rb')
    print(r.getnchannels())
    print(r.getsampwidth())
    print(r.getframerate())
    print(r.getnframes())
    r.close()
    print('test_write_and_read_basic ok')


def test_params():
    buf = io.BytesIO()
    w = wave.open(buf, 'wb')
    w.setnchannels(2)
    w.setsampwidth(2)
    w.setframerate(22050)
    w.writeframes(b'\x00' * 400)
    w.close()

    buf.seek(0)
    r = wave.open(buf)
    params = r.getparams()
    print(params.nchannels)
    print(params.sampwidth)
    print(params.framerate)
    print(params.nframes)
    print(params.comptype)
    print(params.compname)
    r.close()
    print('test_params ok')


def test_readframes():
    buf = io.BytesIO()
    w = wave.open(buf, 'wb')
    w.setnchannels(1)
    w.setsampwidth(1)
    w.setframerate(8000)
    data = bytes(range(10))
    w.writeframes(data)
    w.close()

    buf.seek(0)
    r = wave.open(buf, 'rb')
    frames = r.readframes(5)
    print(frames)
    print(r.tell())
    r.rewind()
    print(r.tell())
    all_frames = r.readframes(r.getnframes())
    print(all_frames)
    r.close()
    print('test_readframes ok')


def test_tell_setpos():
    buf = io.BytesIO()
    w = wave.open(buf, 'wb')
    w.setnchannels(1)
    w.setsampwidth(2)
    w.setframerate(8000)
    w.writeframes(b'\x00\x01' * 20)
    w.close()

    buf.seek(0)
    r = wave.open(buf, 'rb')
    print(r.tell())
    r.setpos(10)
    print(r.tell())
    r.setpos(0)
    print(r.tell())
    r.close()
    print('test_tell_setpos ok')


def test_comptype():
    buf = io.BytesIO()
    w = wave.open(buf, 'wb')
    w.setnchannels(1)
    w.setsampwidth(2)
    w.setframerate(8000)
    w.setcomptype('NONE', 'not compressed')
    w.writeframes(b'\x00\x01' * 5)
    w.close()

    buf.seek(0)
    r = wave.open(buf, 'rb')
    print(r.getcomptype())
    print(r.getcompname())
    r.close()
    print('test_comptype ok')


def test_setparams():
    buf = io.BytesIO()
    w = wave.open(buf, 'wb')
    w.setparams((1, 2, 44100, 0, 'NONE', 'not compressed'))
    w.writeframes(b'\x00\x01' * 10)
    w.close()

    buf.seek(0)
    r = wave.open(buf, 'rb')
    print(r.getnchannels())
    print(r.getsampwidth())
    print(r.getframerate())
    print(r.getnframes())
    r.close()
    print('test_setparams ok')


def test_write_tell():
    buf = io.BytesIO()
    w = wave.open(buf, 'wb')
    w.setnchannels(1)
    w.setsampwidth(2)
    w.setframerate(8000)
    print(w.tell())
    w.writeframes(b'\x00\x01' * 10)
    print(w.tell())
    w.close()
    print('test_write_tell ok')


def test_context_manager():
    buf = io.BytesIO()
    with wave.open(buf, 'wb') as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(8000)
        w.writeframes(b'\x00\x01' * 5)

    buf.seek(0)
    with wave.open(buf, 'rb') as r:
        print(r.getnframes())
        print(r.getsampwidth())
    print('test_context_manager ok')


def test_error_invalid():
    try:
        wave.open(io.BytesIO(b'not a wav file'), 'rb')
        print('no error')
    except wave.Error:
        print('wave.Error raised')
    print('test_error_invalid ok')


def test_setnframes():
    buf = io.BytesIO()
    w = wave.open(buf, 'wb')
    w.setnchannels(1)
    w.setsampwidth(2)
    w.setframerate(8000)
    w.setnframes(10)
    w.writeframes(b'\x00\x01' * 10)
    w.close()

    buf.seek(0)
    r = wave.open(buf, 'rb')
    print(r.getnframes())
    r.close()
    print('test_setnframes ok')


def test_writeframesraw():
    buf = io.BytesIO()
    w = wave.open(buf, 'wb')
    w.setnchannels(1)
    w.setsampwidth(1)
    w.setframerate(8000)
    w.writeframesraw(b'\x01\x02\x03')
    w.writeframesraw(b'\x04\x05')
    w.close()

    buf.seek(0)
    r = wave.open(buf, 'rb')
    print(r.getnframes())
    print(r.readframes(5))
    r.close()
    print('test_writeframesraw ok')


def test_multichannel():
    buf = io.BytesIO()
    w = wave.open(buf, 'wb')
    w.setnchannels(2)
    w.setsampwidth(2)
    w.setframerate(44100)
    w.writeframes(bytes(range(40)))
    w.close()

    buf.seek(0)
    r = wave.open(buf, 'rb')
    print(r.getnchannels())
    print(r.getnframes())
    frame = r.readframes(1)
    print(len(frame))
    r.close()
    print('test_multichannel ok')


test_write_and_read_basic()
test_params()
test_readframes()
test_tell_setpos()
test_comptype()
test_setparams()
test_write_tell()
test_context_manager()
test_error_invalid()
test_setnframes()
test_writeframesraw()
test_multichannel()
