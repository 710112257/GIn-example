// Copyright (c) 2012-2018 Ugorji Nwoke. All rights reserved.
// Use of this source code is governed by a MIT license found in the LICENSE file.

package codec

import (
	"math"
	"reflect"
	"time"
)

// major
const (
	cborMajorUint byte = iota
	cborMajorNegInt
	cborMajorBytes
	cborMajorString
	cborMajorArray
	cborMajorMap
	cborMajorTag
	cborMajorSimpleOrFloat
)

// simple
const (
	cborBdFalse byte = 0xf4 + iota
	cborBdTrue
	cborBdNil
	cborBdUndefined
	cborBdExt
	cborBdFloat16
	cborBdFloat32
	cborBdFloat64
)

// indefinite
const (
	cborBdIndefiniteBytes  byte = 0x5f
	cborBdIndefiniteString byte = 0x7f
	cborBdIndefiniteArray  byte = 0x9f
	cborBdIndefiniteMap    byte = 0xbf
	cborBdBreak            byte = 0xff
)

// These define some in-stream descriptors for
// manual encoding e.g. when doing explicit indefinite-length
const (
	CborStreamBytes  byte = 0x5f
	CborStreamString byte = 0x7f
	CborStreamArray  byte = 0x9f
	CborStreamMap    byte = 0xbf
	CborStreamBreak  byte = 0xff
)

// base values
const (
	cborBaseUint   byte = 0x00
	cborBaseNegInt byte = 0x20
	cborBaseBytes  byte = 0x40
	cborBaseString byte = 0x60
	cborBaseArray  byte = 0x80
	cborBaseMap    byte = 0xa0
	cborBaseTag    byte = 0xc0
	cborBaseSimple byte = 0xe0
)

// const (
// 	cborSelfDesrTag  byte = 0xd9
// 	cborSelfDesrTag2 byte = 0xd9
// 	cborSelfDesrTag3 byte = 0xf7
// )

func cbordesc(bd byte) string {
	switch bd >> 5 {
	case cborMajorUint:
		return "(u)int"
	case cborMajorNegInt:
		return "int"
	case cborMajorBytes:
		return "bytes"
	case cborMajorString:
		return "string"
	case cborMajorArray:
		return "array"
	case cborMajorMap:
		return "map"
	case cborMajorTag:
		return "tag"
	case cborMajorSimpleOrFloat: // default
		switch bd {
		case cborBdNil:
			return "nil"
		case cborBdFalse:
			return "false"
		case cborBdTrue:
			return "true"
		case cborBdFloat16, cborBdFloat32, cborBdFloat64:
			return "float"
		case cborBdIndefiniteBytes:
			return "bytes*"
		case cborBdIndefiniteString:
			return "string*"
		case cborBdIndefiniteArray:
			return "array*"
		case cborBdIndefiniteMap:
			return "map*"
		default:
			return "unknown(simple)"
		}
	}
	return "unknown"
}

// -------------------

type cborEncDriver struct {
	noBuiltInTypes
	encDriverNoopContainerWriter
	e *Encoder
	w *encWriterSwitch
	h *CborHandle
	x [8]byte
	// _ [3]uint64 // padding
}

func (e *cborEncDriver) EncodeNil() {
	e.w.writen1(cborBdNil)
}

func (e *cborEncDriver) EncodeBool(b bool) {
	if b {
		e.w.writen1(cborBdTrue)
	} else {
		e.w.writen1(cborBdFalse)
	}
}

func (e *cborEncDriver) EncodeFloat32(f float32) {
	e.w.writen1(cborBdFloat32)
	bigenHelper{e.x[:4], e.w}.writeUint32(math.Float32bits(f))
}

func (e *cborEncDriver) EncodeFloat64(f float64) {
	e.w.writen1(cborBdFloat64)
	bigenHelper{e.x[:8], e.w}.writeUint64(math.Float64bits(f))
}

func (e *cborEncDriver) encUint(v uint64, bd byte) {
	if v <= 0x17 {
		e.w.writen1(byte(v) + bd)
	} else if v <= math.MaxUint8 {
		e.w.writen2(bd+0x18, uint8(v))
	} else if v <= math.MaxUint16 {
		e.w.writen1(bd + 0x19)
		bigenHelper{e.x[:2], e.w}.writeUint16(uint16(v))
	} else if v <= math.MaxUint32 {
		e.w.writen1(bd + 0x1a)
		bigenHelper{e.x[:4], e.w}.writeUint32(uint32(v))
	} else { // if v <= math.MaxUint64 {
		e.w.writen1(bd + 0x1b)
		bigenHelper{e.x[:8], e.w}.writeUint64(v)
	}
}

func (e *cborEncDriver) EncodeInt(v int64) {
	if v < 0 {
		e.encUint(uint64(-1-v), cborBaseNegInt)
	} else {
		e.encUint(uint64(v), cborBaseUint)
	}
}

func (e *cborEncDriver) EncodeUint(v uint64) {
	e.encUint(v, cborBaseUint)
}

func (e *cborEncDriver) encLen(bd byte, length int) {
	e.encUint(uint64(length), bd)
}

func (e *cborEncDriver) EncodeTime(t time.Time) {
	if t.IsZero() {
		e.EncodeNil()
	} else if e.h.TimeRFC3339 {
		e.encUint(0, cborBaseTag)
		e.EncodeStringEnc(cUTF8, t.Format(time.RFC3339Nano))
	} else {
		e.encUint(1, cborBaseTag)
		t = t.UTC().Round(time.Microsecond)
		sec, nsec := t.Unix(), uint64(t.Nanosecond())
		if nsec == 0 {
			e.EncodeInt(sec)
		} else {
			e.EncodeFloat64(float64(sec) + float64(nsec)/1e9)
		}
	}
}

func (e *cborEncDriver) EncodeExt(rv interface{}, xtag uint64, ext Ext) {
	// xdebugf("cbor EncodeExt: v: %T, %v, ext: %T, %v, ==Self: %v", rv, rv, ext, ext, ext == SelfExt)
	e.encUint(uint64(xtag), cborBaseTag)
	if ext == SelfExt {
		rv2 := baseRV(rv)
		e.e.encodeValue(rv2, e.h.fnNoExt(rv2.Type()))
	} else if v := ext.ConvertExt(rv); v == nil {
		e.EncodeNil()
	} else {
		e.e.encode(v)
	}
}

func (e *cborEncDriver) EncodeRawExt(re *RawExt) {
	e.encUint(uint64(re.Tag), cborBaseTag)
	// only encodes re.Value (never re.Data)
	// if false && re.Data != nil {
	// 	en.encode(re.Data)
	// } else if re.Value != nil {
	if re.Value != nil {
		e.e.encode(re.Value)
	} else {
		e.EncodeNil()
	}
}

func (e *cborEncDriver) WriteArrayStart(length int) {
	if e.h.IndefiniteLength {
		e.w.writen1(cborBdIndefiniteArray)
	} else {
		e.encLen(cborBaseArray, length)
	}
}

func (e *cborEncDriver) WriteMapStart(length int) {
	if e.h.IndefiniteLength {
		e.w.writen1(cborBdIndefiniteMap)
	} else {
		e.encLen(cborBaseMap, length)
	}
}

func (e *cborEncDriver) WriteMapEnd() {
	if e.h.IndefiniteLength {
		e.w.writen1(cborBdBreak)
	}
}

func (e *cborEncDriver) WriteArrayEnd() {
	if e.h.IndefiniteLength {
		e.w.writen1(cborBdBreak)
	}
}

func (e *cborEncDriver) EncodeStringEnc(c charEncoding, v string) {
	e.encStringBytesS(cborBaseString, v)
}

func (e *cborEncDriver) EncodeStringBytesRaw(v []byte) {
	if v == nil {
		e.EncodeNil()
	} else {
		e.encStringBytesS(cborBaseBytes, stringView(v))
	}
}

func (e *cborEncDriver) encStringBytesS(bb byte, v string) {
	if e.h.IndefiniteLength {
		if bb == cborBaseBytes {
			e.w.writen1(cborBdIndefiniteBytes)
		} else {
			e.w.writen1(cborBdIndefiniteString)
		}
		var vlen uint = uint(len(v))
		blen := vlen / 4
		if blen == 0 {
			blen = 64
		} else if blen > 1024 {
			blen = 1024
		}
		for i := uint(0); i < vlen; {
			var v2 string
			i2 := i + blen
			if i2 < vlen {
				v2 = v[i:i2]
			} else {
				v2 = v[i:]
			}
			e.encLen(bb, len(v2))
			e.w.writestr(v2)
			i = i2
		}
		e.w.writen1(cborBdBreak)
	} else {
		e.encLen(bb, len(v))
		e.w.writestr(v)
	}
}

// ----------------------

type cborDecDriver struct {
	d      *Decoder
	h      *CborHandle
	r      *decReaderSwitch
	br     bool // bytes reader
	bdRead bool
	bd     byte
	st     bool // skip tags
	noBuiltInTypes
	// decNoSeparator
	decDriverNoopContainerReader
	// _ [3]uint64 // padding
}

// func (d *cborDecDriver) readNextBdSkipTags() {
// 	d.bd = d.r.readn1()
// 	if d.h.SkipUnexpectedTags {
// 		for d.bd >= cborBaseTag && d.bd < cborBaseSimple {
// 			d.decUint()
// 			d.bd = d.r.readn1()
// 		}
// 	}
// 	d.bdRead = true
// }

// func (d *cborDecDriver) readNextBd() {
// 	d.bd = d.r.readn1()
// 	if d.handleCborSelfDesc && d.bd == cborSelfDesrTag {
// 		if x := d.readn1(); x == cborSelfDesrTag2 {
// 			if x = d.readn1(); x != cborSelfDesrTag3 {
// 				d.d.errorf("mishandled self desc: expected 0xd9d9f7, got: 0xd9d9%x", x)
// 			}
// 		} else {
// 			d.unreadn1()
// 		}
// 	}
// 	d.bdRead = true
// }

func (d *cborDecDriver) readNextBd() {
	d.bd = d.r.readn1()
	d.bdRead = true
}

// skipTags is called to skip any tags in the stream.
//
// Since any value can be tagged, then we should call skipTags
// before any value is decoded.
//
// By definition, skipTags should not be called before
// checking for break, or nil or undefined.
func (d *cborDecDriver) skipTags() {
	for d.bd>>5 == cborMajorTag {
		d.decUint()
		d.bd = d.r.readn1()
	}
}

func (d *cborDecDriver) uncacheRead() {
	if d.bdRead {
		d.r.unreadn1()
		d.bdRead = false
	}
}

func (d *cborDecDriver) ContainerType() (vt valueType) {
	if !d.bdRead {
		d.readNextBd()
	}
	if d.st {
		d.skipTags()
	}
	if d.bd == cborBdNil {
		return valueTypeNil
	} else if d.bd == cborBdIndefiniteBytes || (d.bd>>5 == cborMajorBytes) {
		return valueTypeBytes
	} else if d.bd == cborBdIndefiniteString || (d.bd>>5 == cborMajorString) {
		return valueTypeString
	} else if d.bd == cborBdIndefiniteArray || (d.bd>>5 == cborMajorArray) {
		return valueTypeArray
	} else if d.bd == cborBdIndefiniteMap || (d.bd>>5 == cborMajorMap) {
		return valueTypeMap
	}
	// else {
	// d.d.errorf("isContainerType: unsupported parameter: %v", vt)
	// }
	return valueTypeUnset
}

func (d *cborDecDriver) TryDecodeAsNil() bool {
	if !d.bdRead {
		d.readNextBd()
	}
	// treat Nil and Undefined as nil values
	if d.bd == cborBdNil || d.bd == cborBdUndefined {
		d.bdRead = false
		return true
	}
	return false
}

func (d *cborDecDriver) CheckBreak() (v bool) {
	if !d.bdRead {
		d.readNextBd()
	}
	if d.bd == cborBdBreak {
		d.bdRead = false
		v = true
	}
	return
}

func (d *cborDecDriver) decUint() (ui uint64) {
	v := d.bd & 0x1f
	if v <= 0x17 {
		ui = uint64(v)
	} else {
		if v == 0x18 {
			ui = uint64(d.r.readn1())
		} else if v == 0x19 {
			ui = uint64(bigen.Uint16(d.r.readx(2)))
		} else if v == 0x1a {
			ui = uint64(bigen.Uint32(d.r.readx(4)))
		} else if v == 0x1b {
			ui = uint64(bigen.Uint64(d.r.readx(8)))
		} else {
			d.d.errorf("invalid descriptor decoding uint: %x/%s", d.bd, cbordesc(d.bd))
			return
		}
	}
	return
}

func (d *cborDecDriver) decCheckInteger() (neg bool) {
	if !d.bdRead {
		d.readNextBd()
	}
	if d.st {
		d.skipTags()
	}
	major := d.bd >> 5
	if major == cborMajorUint {
	} else if major == cborMajorNegInt {
		neg = true
	} else {
		d.d.errorf("invalid integer; got major %v from descriptor %x/%s, expected %v or %v",
			major, d.bd, cbordesc(d.bd), cborMajorUint, cborMajorNegInt)
	}
	return
}

func (d *cborDecDriver) DecodeInt64() (i int64) {
	neg := d.decCheckInteger()
	ui := d.decUint()
	d.bdRead = false
	return cborDecInt64(ui, neg)
}

func cborDecInt64(ui uint64, neg bool) (i int64) {
	// check if this number can be converted to an int without overflow
	if neg {
		i = -(chkOvf.SignedIntV(ui + 1))
	} else {
		i = chkOvf.SignedIntV(ui)
	}
	return
}

func (d *cborDecDriver) DecodeUint64() (ui uint64) {
	if d.decCheckInteger() {
		d.d.errorf("cannot assign negative signed value to unsigned type")
	}
	ui = d.decUint()
	d.bdRead = false
	return
}

func (d *cborDecDriver) DecodeFloat64() (f float64) {
	if !d.bdRead {
		d.readNextBd()
	}
	if d.st {
		d.skipTags()
	}
	switch d.bd {
	case cborBdFloat16:
		f = float64(math.Float32frombits(halfFloatToFloatBits(bigen.Uint16(d.r.readx(2)))))
	case cborBdFloat32:
		f = float64(math.Float32frombits(bigen.Uint32(d.r.readx(4))))
	case cborBdFloat64:
		f = math.Float64frombits(bigen.Uint64(d.r.readx(8)))
	default:
		major := d.bd >> 5
		if major == cborMajorUint {
			f = float64(cborDecInt64(d.decUint(), false))
		} else if major == cborMajorNegInt {
			f = float64(cborDecInt64(d.decUint(), true))
		} else {
			d.d.errorf("float only valid from float16/32/64 - invalid descriptor %x/%s", d.bd, cbordesc(d.bd))
		}
	}
	d.bdRead = false
	return
}

// bool can be decoded from bool only (single byte).
func (d *cborDecDriver) DecodeBool() (b bool) {
	if !d.bdRead {
		d.readNextBd()
	}
	if d.st {
		d.skipTags()
	}
	if d.bd == cborBdTrue {
		b = true
	} else if d.bd == cborBdFalse {
	} else {
		d.d.errorf("not bool - %s %x/%s", msgBadDesc, d.bd, cbordesc(d.bd))
		return
	}
	d.bdRead = false
	return
}

func (d *cborDecDriver) ReadMapStart() (length int) {
	if !d.bdRead {
		d.readNextBd()
	}
	if d.st {
		d.skipTags()
	}
	d.bdRead = false
	if d.bd == cborBdIndefiniteMap {
		return -1
	}
	if d.bd>>5 != cborMajorMap {
		d.d.errorf("error reading map; got major type: %x, expected: %x/%s", d.bd>>5, cborMajorMap, cbordesc(d.bd))
	}
	return d.decLen()
}

func (d *cborDecDriver) ReadArrayStart() (length int) {
	if !d.bdRead {
		d.readNextBd()
	}
	if d.st {
		d.skipTags()
	}
	d.bdRead = false
	if d.bd == cborBdIndefiniteArray {
		return -1
	}
	if d.bd>>5 != cborMajorArray {
		d.d.errorf("error reading array; got major type: %x, expect: %x/%s", d.bd>>5, cborMajorArray, cbordesc(d.bd))
	}
	return d.decLen()
}

func (d *cborDecDriver) decLen() int {
	return int(d.decUint())
}

func (d *cborDecDriver) decAppendIndefiniteBytes(bs []byte) []byte {
	d.bdRead = false
	for !d.CheckBreak() {
		if major := d.bd >> 5; major != cborMajorBytes && major != cborMajorString {
			d.d.errorf("error reading indefinite string/bytes;"+
				" got major %v, expected %x/%s", major, d.bd, cbordesc(d.bd))
		}
		n := d.decLen()
		oldLen := len(bs)
		newLen := oldLen + n
		if newLen > cap(bs) {
			bs2 := make([]byte, newLen, 2*cap(bs)+n)
			copy(bs2, bs)
			bs = bs2
		} else {
			bs = bs[:newLen]
		}
		d.r.readb(bs[oldLen:newLen])
		// bs = append(bs, d.r.readn()...)
		d.bdRead = false
	}
	d.bdRead = false
	return bs
}

func (d *cborDecDriver) DecodeBytes(bs []byte, zerocopy bool) (bsOut []byte) {
	if !d.bdRead {
		d.readNextBd()
	}
	if d.st {
		d.skipTags()
	}
	if d.bd == cborBdNil || d.bd == cborBdUndefined {
		d.bdRead = false
		return nil
	}
	if d.bd == cborBdIndefiniteBytes || d.bd == cborBdIndefiniteString {
		d.bdRead = false
		if bs == nil {
			if zerocopy {
				return d.decAppendIndefiniteBytes(d.d.b[:0])
			}
			return d.decAppendIndefiniteBytes(zeroByteSlice)
		}
		return d.decAppendIndefiniteBytes(bs[:0])
	}
	// check if an "array" of uint8's (see ContainerType for how to infer if an array)
	// if d.bd == cborBdIndefiniteArray || (d.bd >> 5 == cborMajorArray) {
	// 	bsOut, _ = fastpathTV.DecSliceUint8V(bs, true, d.d)
	// 	return
	// }
	if d.bd == cborBdIndefiniteArray {
		d.bdRead = false
		if zerocopy && len(bs) == 0 {
			bs = d.d.b[:]
		}
		if bs == nil {
			bs = []byte{}
		} else {
			bs = bs[:0]
		}
		for !d.CheckBreak() {
			bs = append(bs, uint8(chkOvf.UintV(d.DecodeUint64(), 8)))
		}
		return bs
	}
	if d.bd>>5 == cborMajorArray {
		d.bdRead = false
		if zerocopy && len(bs) == 0 {
			bs = d.d.b[:]
		}
		slen := d.decLen()
		bs = usableByteSlice(bs, slen)
		for i := 0; i < slen; i++ {
			bs[i] = uint8(chkOvf.UintV(d.DecodeUint64(), 8))
		}
		return bs
	}
	clen := d.decLen()
	d.bdRead = false
	if zerocopy {
		if d.br {
			return d.r.readx(uint(clen))
		} else if len(bs) == 0 {
			bs = d.d.b[:]
		}
	}
	return decByteSlice(d.r, clen, d.h.MaxInitLen, bs)
}

func (d *cborDecDriver) DecodeString() (s string) {
	return string(d.DecodeBytes(d.d.b[:], true))
}

func (d *cborDecDriver) DecodeStringAsBytes() (s []byte) {
	return d.DecodeBytes(d.d.b[:], true)
}

func (d *cborDecDriver) DecodeTime() (t time.Time) {
	if !d.bdRead {
		d.readNextBd()
	}
	if d.bd == cborBdNil || d.bd == cborBdUndefined {
		d.bdRead = false
		return
	}
	if d.bd>>5 != cborMajorTag {
		d.d.errorf("error reading tag; expected major type: %x, got: %x", cborMajorTag, d.bd>>5)
	}
	xtag := d.decUint()
	d.bdRead = false
	return d.decodeTime(xtag)
}

func (d *cborDecDriver) decodeTime(xtag uint64) (t time.Time) {
	switch xtag {
	case 0:
		var err error
		if t, err = time.Parse(time.RFC3339, stringView(d.DecodeStringAsBytes())); err != nil {
			d.d.errorv(err)
		}
	case 1:
		// if !d.bdRead {
		// 	d.readNextBd()
		// }
		// // decode an int64 or a float, and infer time.Time from there.
		// // for floats, round to microseconds, as that is what is guaranteed to fit well.
		// switch {
		// case d.bd == cborBdFloat16, d.bd == cborBdFloat32:
		// 	f1, f2 := math.Modf(d.DecodeFloat64())
		// 	t = time.Unix(int64(f1), int64(f2*1e9))
		// case d.bd == cborBdFloat64:
		// 	f1, f2 := math.Modf(d.DecodeFloat64())
		// 	t = time.Unix(int64(f1), int64(f2*1e9))
		// case d.bd >= cborBaseUint && d.bd < cborBaseBytes:
		// 	t = time.Unix(d.DecodeInt64(), 0)
		// default:
		// 	d.d.errorf("time.Time can only be decoded from a number (or RFC3339 string)")
		// }
		f1, f2 := math.Modf(d.DecodeFloat64())
		t = time.Unix(int64(f1), int64(f2*1e9))
	default:
		d.d.errorf("invalid tag for time.Time - expecting 0 or 1, got 0x%x", xtag)
	}
	t = t.UTC().Round(time.Microsecond)
	return
}

func (d *cborDecDriver) DecodeExt(rv interface{}, xtag uint64, ext Ext) (realxtag uint64) {
	// xdebugf("cbor DecodeExt: v: %T, %v", rv, rv)
	if !d.bdRead {
		d.readNextBd()
	}
	if d.bd>>5 != cborMajorTag {
		d.d.errorf("error reading tag; expected major type: %x, got: %x", cborMajorTag, d.bd>>5)
	}
	u := d.decUint()
	d.bdRead = false
	realxtag = u
	if ext == nil {
		re := rv.(*RawExt)
		re.Tag = realxtag
		d.d.decode(&re.Value)
	} else if xtag != realxtag {
		d.d.errorf("Wrong extension tag. Got %b. Expecting: %v", realxtag, xtag)
		return
	} else if ext == SelfExt {
		rv2 := baseRV(rv)
		d.d.decodeValue(rv2, d.h.fnNoExt(rv2.Type()))
	} else {
		d.d.interfaceExtConvertAndDecode(rv, ext)
	}
	d.bdRead = false
	return
}

func (d *cborDecDriver) DecodeNaked() {
	if !d.bdRead {
		d.readNextBd()
	}

	n := d.d.naked()
	var decodeFurther bool

	// xdebug2f("DecodeNaked: bd: %x, bd >> 5: %x", d.bd, d.bd>>5)

	switch d.bd >> 5 {
	case cborMajorUint:
		if d.h.SignedInteger {
			n.v = valueTypeInt
			n.i = d.DecodeInt64()
		} else {
			n.v = valueTypeUint
			n.u = d.DecodeUint64()
		}
	case cborMajorNegInt:
		n.v = valueTypeInt
		n.i = d.DecodeInt64()
	case cborMajorBytes:
		decNakedReadRawBytes(d, d.d, n, d.h.RawToString)
	case cborMajorString:
		n.v = valueTypeString
		n.s = d.DecodeString()
	case cborMajorArray:
		n.v = valueTypeArray
		decodeFurther = true
	case cborMajorMap:
		n.v = valueTypeMap
		decodeFurther = true
	case cborMajorTag:
		n.v = valueTypeExt
		n.u = d.decUint()
		n.l = nil
		if n.u == 0 || n.u == 1 {
			d.bdRead = false
			n.v = valueTypeTime
			n.t = d.decodeTime(n.u)
		} else if d.st && d.h.getExtForTag(n.u) == nil {
			// d.skipTags() // no need to call this - tags already skipped
			d.bdRead = false
			d.DecodeNaked()
			return // return when done (as true recursive function)
		}
		// d.bdRead = false
		// d.d.decode(&re.Value) // handled by decode itself.
		// decodeFurther = true
	case cborMajorSimpleOrFloat:
		switch d.bd {
		case cborBdNil:
			n.v = valueTypeNil
		case cborBdFalse:
			n.v = valueTypeBool
			n.b = false
		case cborBdTrue:
			n.v = valueTypeBool
			n.b = true
		case cborBdFloat16, cborBdFloat32, cborBdFloat64:
			n.v = valueTypeFloat
			n.f = d.DecodeFloat64()
		case cborBdIndefiniteBytes:
			decNakedReadRawBytes(d, d.d, n, d.h.RawToString)
		case cborBdIndefiniteString:
			n.v = valueTypeString
			n.s = d.DecodeString()
		case cborBdIndefiniteArray:
			n.v = valueTypeArray
			decodeFurther = true
		case cborBdIndefiniteMap:
			n.v = valueTypeMap
			decodeFurther = true
		default:
			d.d.errorf("decodeNaked: Unrecognized d.bd: 0x%x", d.bd)
		}
	default: // should never happen
		d.d.errorf("decodeNaked: Unrecognized d.bd: 0x%x", d.bd)
	}
	if !decodeFurther {
		d.bdRead = false
	}
}

// -------------------------

// CborHandle is a Handle for the CBOR encoding format,
// defined at http://tools.ietf.org/html/rfc7049 and documented further at http://cbor.io .
//
// CBOR is comprehensively supported, including support for:
//   - indefinite-length arrays/maps/bytes/strings
//   - (extension) tags in range 0..0xffff (0 .. 65535)
//   - half, single and double-precision floats
//   - all numbers (1, 2, 4 and 8-byte signed and unsigned integers)
//   - nil, true, false, ...
//   - arrays and maps, bytes and text strings
//
// None of the optional extensions (with tags) defined in the spec are supported out-of-the-box.
// Users can implement them as needed (using SetExt), including spec-documented ones:
//   - timestamp, BigNum, BigFloat, Decimals,
//   - Encoded Text (e.g. URL, regexp, base64, MIME Message), etc.
type CborHandle struct {
	binaryEncodingType
	noElemSeparators
	BasicHandle

	// IndefiniteLength=true, means that we encode using indefinitelength
	IndefiniteLength bool

	// TimeRFC3339 says to encode time.Time using RFC3339 format.
	// If unset, we encode time.Time using seconds past epoch.
	TimeRFC3339 bool

	// SkipUnexpectedTags says to skip over any tags for which extensions are
	// not defined. This is in keeping with the cbor spec on "Optional Tagging of Items".
	//
	// Furthermore, this allows the skipping over of the Self Describing Tag 0xd9d9f7.
	SkipUnexpectedTags bool

	_ [1]uint64 // padding (cache-aligned)
}

// Name returns the name of the handle: cbor
func (h *CborHandle) Name() string { return "cbor" }

// SetInterfaceExt sets an extension
func (h *CborHandle) SetInterfaceExt(rt reflect.Type, tag uint64, ext InterfaceExt) (err error) {
	return h.SetExt(rt, tag, makeExt(ext))
}

func (h *CborHandle) newEncDriver(e *Encoder) encDriver {
	return &cborEncDriver{e: e, w: e.w(), h: h}
}

func (h *CborHandle) newDecDriver(d *Decoder) decDriver {
	return &cborDecDriver{d: d, h: h, r: d.r(), br: d.bytes, st: h.SkipUnexpectedTags}
}

func (e *cborEncDriver) reset() {
	e.w = e.e.w()
}

func (d *cborDecDriver) reset() {
	d.r, d.br = d.d.r(), d.d.bytes
	d.bd, d.bdRead = 0, false
	d.st = d.h.SkipUnexpectedTags
}

var _ decDriver = (*cborDecDriver)(nil)
var _ encDriver = (*cborEncDriver)(nil)
