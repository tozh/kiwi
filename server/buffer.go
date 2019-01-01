package server

import (
	"errors"
	"io"
	"unicode/utf8"
)

// this file makes some little change to the bytes.Buffer, change it to 2 sizes and more suitable for kiwi.

const maxInt = int(^uint(0) >> 1)

const bootLen = 256
const largeBufferBootLen = 4096 // 4k

const minRead = 1024                 // 1k
const largeMinRead = PROTO_IOBUF_LEN // 16k

var ErrTooLarge = errors.New("bytes.Buffer: too large")
var errNegativeRead = errors.New("bytes.Buffer: reader returned negative count from Read")

/* -------------- function begin --------------- */

func IndexByte(s []byte, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func makeSlice(n int) []byte {
	// If the make fails, give a known error.
	defer func() {
		if recover() != nil {
			panic(ErrTooLarge)
		}
	}()
	return make([]byte, n)
}

func NewBuffer(buf []byte) *Buffer { return &Buffer{buf: buf} }

func NewBufferString(s string) *Buffer {
	return &Buffer{buf: []byte(s)}
}

func NewLargeBuffer(buf []byte) *LargeBuffer { return &LargeBuffer{buf: buf} }

func NewLargeBufferString(s string) *LargeBuffer {
	return &LargeBuffer{buf: []byte(s)}
}

/* ============== function end  ============== */

/* -------------- small buffer begin --------------- */
type Buffer struct {
	buf       []byte         // contents are the bytes buf[off : len(buf)]
	off       int            // read at &buf[off], write at &buf[len(buf)]
	bootstrap [bootLen] byte // memory to hold first slice; helps small buffers avoid allocation.
}

func (b *Buffer) Bytes() []byte { return b.buf[b.off:] }

func (b *Buffer) String() string {
	if b == nil {
		// Special case, useful in debugging.
		return "<nil>"
	}
	return string(b.buf[b.off:])
}

// empty returns whether the unread portion of the buffer is empty.
func (b *Buffer) empty() bool { return len(b.buf) <= b.off }

// b.Len() == len(b.Bytes()).
func (b *Buffer) Len() int { return len(b.buf) - b.off }

func (b *Buffer) Cap() int { return cap(b.buf) }

func (b *Buffer) Truncate(n int) {
	if n == 0 {
		b.Reset()
		return
	}
	if n < 0 || n > b.Len() {
		panic("bytes.Buffer: truncation out of range")
	}
	b.buf = b.buf[:b.off+n]
}

// Reset resets the buffer to be empty,
// but it retains the underlying storage for use by future writes.
// Reset is the same as Truncate(0).
func (b *Buffer) Reset() {
	b.buf = b.buf[:0]
	b.off = 0
}

func (b *Buffer) tryGrowByReslice(n int) (int, bool) {
	if l := len(b.buf); n <= cap(b.buf)-l {
		b.buf = b.buf[:l+n]
		return l, true
	}
	return 0, false
}

func (b *Buffer) grow(n int) int {
	m := b.Len()
	// If buffer is empty, reset to recover space.
	if m == 0 && b.off != 0 {
		b.Reset()
	}
	// Try to grow by means of a reslice.
	if i, ok := b.tryGrowByReslice(n); ok {
		return i
	}
	// Check if we can make use of bootstrap array.
	if b.buf == nil && n <= len(b.bootstrap) {
		b.buf = b.bootstrap[:n]
		return 0
	}
	c := cap(b.buf)
	if n <= c/2-m {
		// We can slide things down instead of allocating a new
		// slice. We only need m+n <= c to slide, but
		// we instead let capacity get twice as large so we
		// don't spend all our time copying.
		copy(b.buf, b.buf[b.off:])
	} else if c > maxInt-c-n {
		panic(ErrTooLarge)
	} else {
		// Not enough space anywhere, we need to allocate.
		buf := makeSlice(2*c + n)
		copy(buf, b.buf[b.off:])
		b.buf = buf
	}
	// Restore b.off and len(b.buf).
	b.off = 0
	b.buf = b.buf[:m+n]
	return m
}

func (b *Buffer) Grow(n int) {
	if n < 0 {
		panic("bytes.Buffer.Grow: negative count")
	}
	m := b.grow(n)
	b.buf = b.buf[:m]
}

func (b *Buffer) Write(p []byte) (n int, err error) {
	m, ok := b.tryGrowByReslice(len(p))
	if !ok {
		m = b.grow(len(p))
	}
	return copy(b.buf[m:], p), nil
}

func (b *Buffer) WriteString(s string) (n int, err error) {
	m, ok := b.tryGrowByReslice(len(s))
	if !ok {
		m = b.grow(len(s))
	}
	return copy(b.buf[m:], s), nil
}

func (b *Buffer) ReadFrom(r io.Reader) (n int64, err error) {
	for {
		i := b.grow(minRead)
		m, e := r.Read(b.buf[i:cap(b.buf)])
		if m < 0 {
			panic(errNegativeRead)
		}

		b.buf = b.buf[:i+m]
		n += int64(m)
		if e == io.EOF {
			return n, nil // e is EOF, so return nil explicitly
		}
		if e != nil {
			return n, e
		}
	}
}

func (b *Buffer) WriteTo(w io.Writer) (n int64, err error) {
	if nBytes := b.Len(); nBytes > 0 {
		m, e := w.Write(b.buf[b.off:])
		if m > nBytes {
			panic("bytes.Buffer.WriteTo: invalid Write count")
		}
		b.off += m
		n = int64(m)
		if e != nil {
			return n, e
		}
		// all bytes should have been written, by definition of
		// Write method in io.Writer
		if m != nBytes {
			return n, io.ErrShortWrite
		}
	}
	// Buffer is now empty; reset.
	b.Reset()
	return n, nil
}

func (b *Buffer) WriteByte(c byte) error {
	m, ok := b.tryGrowByReslice(1)
	if !ok {
		m = b.grow(1)
	}
	b.buf[m] = c
	return nil
}

func (b *Buffer) WriteRune(r rune) (n int, err error) {
	if r < utf8.RuneSelf {
		b.WriteByte(byte(r))
		return 1, nil
	}
	m, ok := b.tryGrowByReslice(utf8.UTFMax)
	if !ok {
		m = b.grow(utf8.UTFMax)
	}
	n = utf8.EncodeRune(b.buf[m:m+utf8.UTFMax], r)
	b.buf = b.buf[:m+n]
	return n, nil
}

func (b *Buffer) Read(p []byte) (n int, err error) {
	if b.empty() {
		// Buffer is empty, reset to recover space.
		b.Reset()
		if len(p) == 0 {
			return 0, nil
		}
		return 0, io.EOF
	}
	n = copy(p, b.buf[b.off:])
	b.off += n
	return n, nil
}

func (b *Buffer) Next(n int) []byte {
	m := b.Len()
	if n > m {
		n = m
	}
	data := b.buf[b.off : b.off+n]
	b.off += n
	if n > 0 {
	}
	return data
}

func (b *Buffer) ReadByte() (byte, error) {
	if b.empty() {
		// Buffer is empty, reset to recover space.
		b.Reset()
		return 0, io.EOF
	}
	c := b.buf[b.off]
	b.off++
	return c, nil
}

func (b *Buffer) ReadRune() (r rune, size int, err error) {
	if b.empty() {
		// Buffer is empty, reset to recover space.
		b.Reset()
		return 0, 0, io.EOF
	}
	c := b.buf[b.off]
	if c < utf8.RuneSelf {
		b.off++
		return rune(c), 1, nil
	}
	r, n := utf8.DecodeRune(b.buf[b.off:])
	b.off += n
	return r, n, nil
}

func (b *Buffer) ReadBytes(delim byte) (line []byte, err error) {
	slice, err := b.readSlice(delim)
	// return a copy of slice. The buffer'kiwiS backing array may
	// be overwritten by later calls.
	line = append(line, slice...)
	return line, err
}

// readSlice is like ReadBytes but returns a reference to internal buffer data.
func (b *Buffer) readSlice(delim byte) (line []byte, err error) {
	i := IndexByte(b.buf[b.off:], delim)
	end := b.off + i + 1
	if i < 0 {
		end = len(b.buf)
		err = io.EOF
	}
	line = b.buf[b.off:end]
	b.off = end
	return line, err
}

func (b *Buffer) ReadString(delim byte) (line string, err error) {
	slice, err := b.readSlice(delim)
	return string(slice), err
}

func (b *Buffer) ReadByteNotGoForward() (byte, error) {
	if b.empty() {
		// Buffer is empty, reset to recover space.
		b.Reset()
		return 0, io.EOF
	}
	c := b.buf[b.off]
	return c, nil
}

func (b *Buffer) ReadSliceExclude(delim byte) (line []byte, err error) {
	// return the slice.
	return b.readSliceExclude(delim)
}

func (b *Buffer) ReadBytesExclude(delim byte) (line []byte, err error) {
	slice, err := b.readSliceExclude(delim)
	// return a copy of slice. The buffer'kiwiS backing array may
	// be overwritten by later calls.
	line = append(line, slice...)
	return slice, err
}

func (b *Buffer) ReadStringExclude(delim byte) (line string, err error) {
	slice, err := b.readSliceExclude(delim)
	// return a copy of slice. The buffer'kiwiS backing array may
	// be overwritten by later calls.
	return string(slice), err
}

// readSlice is like ReadBytes but returns a reference to internal buffer data.
func (b *Buffer) readSliceExclude(delim byte) (line []byte, err error) {
	i := IndexByte(b.buf[b.off:], delim)
	end := b.off + i + 1
	if i < 0 {
		end = len(b.buf)
		err = io.EOF
	}
	line = b.buf[b.off : end-1]
	b.off = end
	return line, err
}

func (b *Buffer) SearchByte(delim byte) (index int) {
	i := IndexByte(b.buf[b.off:], delim)
	if i < 0 {
		return -1
	}
	return b.off + i + 1
}

/* ============== small buffer end ============== */

/* -------------- large buffer start--------------- */

type LargeBuffer struct {
	buf       []byte                    // contents are the bytes buf[off : len(buf)]
	off       int                       // read at &buf[off], write at &buf[len(buf)]
	bootstrap [largeBufferBootLen] byte // memory to hold first slice; helps small buffers avoid allocation.
}

func (b *LargeBuffer) Bytes() []byte { return b.buf[b.off:] }

func (b *LargeBuffer) String() string {
	if b == nil {
		// Special case, useful in debugging.
		return "<nil>"
	}
	return string(b.buf[b.off:])
}

// empty returns whether the unread portion of the buffer is empty.
func (b *LargeBuffer) empty() bool { return len(b.buf) <= b.off }

// b.Len() == len(b.Bytes()).
func (b *LargeBuffer) Len() int { return len(b.buf) - b.off }

func (b *LargeBuffer) Cap() int { return cap(b.buf) }

func (b *LargeBuffer) Truncate(n int) {
	if n == 0 {
		b.Reset()
		return
	}
	if n < 0 || n > b.Len() {
		panic("bytes.LargeBuffer: truncation out of range")
	}
	b.buf = b.buf[:b.off+n]
}

// Reset resets the buffer to be empty,
// but it retains the underlying storage for use by future writes.
// Reset is the same as Truncate(0).
func (b *LargeBuffer) Reset() {
	b.buf = b.buf[:0]
	b.off = 0
}

func (b *LargeBuffer) tryGrowByReslice(n int) (int, bool) {
	if l := len(b.buf); n <= cap(b.buf)-l {
		b.buf = b.buf[:l+n]
		return l, true
	}
	return 0, false
}

func (b *LargeBuffer) grow(n int) int {
	m := b.Len()
	// If buffer is empty, reset to recover space.
	if m == 0 && b.off != 0 {
		b.Reset()
	}
	// Try to grow by means of a reslice.
	if i, ok := b.tryGrowByReslice(n); ok {
		return i
	}
	// Check if we can make use of bootstrap array.
	if b.buf == nil && n <= len(b.bootstrap) {
		b.buf = b.bootstrap[:n]
		return 0
	}
	c := cap(b.buf)
	if n <= c/2-m {
		// We can slide things down instead of allocating a new
		// slice. We only need m+n <= c to slide, but
		// we instead let capacity get twice as large so we
		// don't spend all our time copying.
		copy(b.buf, b.buf[b.off:])
	} else if c > maxInt-c-n {
		panic(ErrTooLarge)
	} else {
		// Not enough space anywhere, we need to allocate.
		buf := makeSlice(2*c + n)
		copy(buf, b.buf[b.off:])
		b.buf = buf
	}
	// Restore b.off and len(b.buf).
	b.off = 0
	b.buf = b.buf[:m+n]
	return m
}

func (b *LargeBuffer) Grow(n int) {
	if n < 0 {
		panic("bytes.LargeBuffer.Grow: negative count")
	}
	m := b.grow(n)
	b.buf = b.buf[:m]
}

func (b *LargeBuffer) Write(p []byte) (n int, err error) {
	m, ok := b.tryGrowByReslice(len(p))
	if !ok {
		m = b.grow(len(p))
	}
	return copy(b.buf[m:], p), nil
}

func (b *LargeBuffer) WriteString(s string) (n int, err error) {
	m, ok := b.tryGrowByReslice(len(s))
	if !ok {
		m = b.grow(len(s))
	}
	return copy(b.buf[m:], s), nil
}

func (b *LargeBuffer) ReadFrom(r io.Reader) (n int64, err error) {
	for {
		i := b.grow(largeMinRead)
		m, e := r.Read(b.buf[i:cap(b.buf)])
		if m < 0 {
			panic(errNegativeRead)
		}

		b.buf = b.buf[:i+m]
		n += int64(m)
		if e == io.EOF {
			return n, nil // e is EOF, so return nil explicitly
		}
		if e != nil {
			return n, e
		}
	}
}

func (b *LargeBuffer) WriteTo(w io.Writer) (n int64, err error) {
	if nBytes := b.Len(); nBytes > 0 {
		m, e := w.Write(b.buf[b.off:])
		if m > nBytes {
			panic("bytes.LargeBuffer.WriteTo: invalid Write count")
		}
		b.off += m
		n = int64(m)
		if e != nil {
			return n, e
		}
		// all bytes should have been written, by definition of
		// Write method in io.Writer
		if m != nBytes {
			return n, io.ErrShortWrite
		}
	}
	// LargeBuffer is now empty; reset.
	b.Reset()
	return n, nil
}

func (b *LargeBuffer) WriteByte(c byte) error {
	m, ok := b.tryGrowByReslice(1)
	if !ok {
		m = b.grow(1)
	}
	b.buf[m] = c
	return nil
}

func (b *LargeBuffer) WriteRune(r rune) (n int, err error) {
	if r < utf8.RuneSelf {
		b.WriteByte(byte(r))
		return 1, nil
	}
	m, ok := b.tryGrowByReslice(utf8.UTFMax)
	if !ok {
		m = b.grow(utf8.UTFMax)
	}
	n = utf8.EncodeRune(b.buf[m:m+utf8.UTFMax], r)
	b.buf = b.buf[:m+n]
	return n, nil
}

func (b *LargeBuffer) Read(p []byte) (n int, err error) {
	if b.empty() {
		// LargeBuffer is empty, reset to recover space.
		b.Reset()
		if len(p) == 0 {
			return 0, nil
		}
		return 0, io.EOF
	}
	n = copy(p, b.buf[b.off:])
	b.off += n
	return n, nil
}

func (b *LargeBuffer) Next(n int) []byte {
	m := b.Len()
	if n > m {
		n = m
	}
	data := b.buf[b.off : b.off+n]
	b.off += n
	if n > 0 {
	}
	return data
}

func (b *LargeBuffer) ReadByte() (byte, error) {
	if b.empty() {
		// LargeBuffer is empty, reset to recover space.
		b.Reset()
		return 0, io.EOF
	}
	c := b.buf[b.off]
	b.off++
	return c, nil
}

func (b *LargeBuffer) ReadRune() (r rune, size int, err error) {
	if b.empty() {
		// LargeBuffer is empty, reset to recover space.
		b.Reset()
		return 0, 0, io.EOF
	}
	c := b.buf[b.off]
	if c < utf8.RuneSelf {
		b.off++
		return rune(c), 1, nil
	}
	r, n := utf8.DecodeRune(b.buf[b.off:])
	b.off += n
	return r, n, nil
}

func (b *LargeBuffer) ReadBytes(delim byte) (line []byte, err error) {
	slice, err := b.readSlice(delim)
	// return a copy of slice. The buffer'kiwiS backing array may
	// be overwritten by later calls.
	line = append(line, slice...)
	return line, err
}

// readSlice is like ReadBytes but returns a reference to internal buffer data.
func (b *LargeBuffer) readSlice(delim byte) (line []byte, err error) {
	i := IndexByte(b.buf[b.off:], delim)
	end := b.off + i + 1
	if i < 0 {
		end = len(b.buf)
		err = io.EOF
	}
	line = b.buf[b.off:end]
	b.off = end
	return line, err
}

func (b *LargeBuffer) ReadString(delim byte) (line string, err error) {
	slice, err := b.readSlice(delim)
	return string(slice), err
}

func (b *LargeBuffer) ReadByteNotGoForward() (byte, error) {
	if b.empty() {
		// LargeBuffer is empty, reset to recover space.
		b.Reset()
		return 0, io.EOF
	}
	c := b.buf[b.off]
	return c, nil
}

func (b *LargeBuffer) ReadSliceExclude(delim byte) (line []byte, err error) {
	// return the slice.
	return b.readSliceExclude(delim)
}

func (b *LargeBuffer) ReadBytesExclude(delim byte) (line []byte, err error) {
	slice, err := b.readSliceExclude(delim)
	// return a copy of slice. The buffer'kiwiS backing array may
	// be overwritten by later calls.
	line = append(line, slice...)
	return slice, err
}

func (b *LargeBuffer) ReadStringExclude(delim byte) (line string, err error) {
	slice, err := b.readSliceExclude(delim)
	// return a copy of slice. The buffer'kiwiS backing array may
	// be overwritten by later calls.
	return string(slice), err
}

// readSlice is like ReadBytes but returns a reference to internal buffer data.
func (b *LargeBuffer) readSliceExclude(delim byte) (line []byte, err error) {
	i := IndexByte(b.buf[b.off:], delim)
	end := b.off + i + 1
	if i < 0 {
		end = len(b.buf)
		err = io.EOF
	}
	line = b.buf[b.off : end-1]
	b.off = end
	return line, err
}

func (b *LargeBuffer) SearchByte(delim byte) (index int) {
	i := IndexByte(b.buf[b.off:], delim)
	if i < 0 {
		return -1
	}
	return b.off + i + 1
}

/* ============== large buffer end ============== */
