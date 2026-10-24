package engine

import (
	"encoding/binary"
	"errors"
	"uuid"
)

func putUUID(dst []byte, value uuid.UUID) []byte { return append(dst, value[:]...) }

func putU64(dst []byte, value uint64) { binary.BigEndian.PutUint64(dst, value) }

func putI64(dst []byte, value int64) { binary.BigEndian.PutUint64(dst, uint64(value)) }

func appendBytes(dst, value []byte) []byte {
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(value)))
	dst = append(dst, size[:]...)
	return append(dst, value...)
}

func appendString(dst []byte, value string) []byte { return appendBytes(dst, []byte(value)) }

// decoder reads a record field by field. A truncated read parks pos at -1 and
// keeps returning zero values, so a decode function can assign every field up
// front and check the result once with check. Folding the failure into pos
// keeps the decoder two words wide and every read small enough to inline.
type decoder struct {
	data []byte
	pos  int
}

// failed reports whether an earlier read ran past the end of the record.
func (d *decoder) failed() bool { return d.pos < 0 }

func (d *decoder) fail() { d.pos = -1 }

func (d *decoder) bytes() []byte {
	if d.failed() || d.pos+4 > len(d.data) {
		d.fail()
		return nil
	}
	size := int(binary.BigEndian.Uint32(d.data[d.pos:]))
	d.pos += 4
	if d.pos+size > len(d.data) {
		d.fail()
		return nil
	}
	value := d.data[d.pos : d.pos+size]
	d.pos += size
	return value
}

func (d *decoder) string() string { return string(d.bytes()) }

func (d *decoder) uuid() uuid.UUID {
	if d.failed() || d.pos+16 > len(d.data) {
		d.fail()
		return uuid.Nil()
	}
	var value uuid.UUID
	copy(value[:], d.data[d.pos:])
	d.pos += 16
	return value
}

func (d *decoder) u64() uint64 {
	if d.failed() || d.pos+8 > len(d.data) {
		d.fail()
		return 0
	}
	value := binary.BigEndian.Uint64(d.data[d.pos:])
	d.pos += 8
	return value
}

func (d *decoder) u32() uint32 {
	if d.failed() || d.pos+4 > len(d.data) {
		d.fail()
		return 0
	}
	value := binary.BigEndian.Uint32(d.data[d.pos:])
	d.pos += 4
	return value
}

func (d *decoder) i64() int64 { return int64(d.u64()) }

// fixed consumes exactly n bytes, aliasing them into the record buffer.
func (d *decoder) fixed(n int) []byte {
	if d.failed() || d.pos+n > len(d.data) {
		d.fail()
		return nil
	}
	value := d.data[d.pos : d.pos+n]
	d.pos += n
	return value
}

// byte consumes one trailing byte, returning 0 without failing when the record
// is exhausted; check reports the truncation.
func (d *decoder) byte() byte {
	if d.failed() || d.pos >= len(d.data) {
		return 0
	}
	value := d.data[d.pos]
	d.pos++
	return value
}

func (d *decoder) done() bool { return d.pos == len(d.data) }

// check is the single error gate at the end of a decode function: a truncated
// read, or a record that has trailing bytes.
func (d *decoder) check(name string) error {
	if d.failed() || !d.done() {
		return errors.New("invalid " + name + " record")
	}
	return nil
}
