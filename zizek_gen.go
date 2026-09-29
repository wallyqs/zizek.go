package zizek

import (
	"math"
	"math/bits"
)

type optionalBytes struct {
	value []byte
	ok bool
}

type optionalBool struct {
	value bool
	ok bool
}

type optionalF64 struct {
	value float64
	ok bool
}

type optionalU64 struct {
	value uint64
	ok bool
}

type optionalI64 struct {
	value int64
	ok bool
}

func encodeExponent(biased_exp uint64) uint64 {
	var c2047 uint64
	var is2047 bool
	var c1023 uint64
	var ge bool
	var r uint64
	var c2046 uint64
	var r2 uint64
	c2047 = uint64(2047)
	is2047 = (biased_exp == c2047)
	if is2047 {
		return c2047
	}
	c1023 = uint64(1023)
	ge = (biased_exp >= c1023)
	if ge {
		r = (biased_exp - c1023)
		return r
	}
	c2046 = uint64(2046)
	r2 = (c2046 - biased_exp)
	return r2
}

func decodeExponent(enc uint64) uint64 {
	var c2047 uint64
	var is2047 bool
	var c1023 uint64
	var le bool
	var r uint64
	var c2046 uint64
	var r2 uint64
	c2047 = uint64(2047)
	is2047 = (enc == c2047)
	if is2047 {
		return c2047
	}
	c1023 = uint64(1023)
	le = (enc <= c1023)
	if le {
		r = (enc + c1023)
		return r
	}
	c2046 = uint64(2046)
	r2 = (c2046 - enc)
	return r2
}

func reverseBitsN(val uint64, n uint64) uint64 {
	var c0 uint64
	var isz bool
	var c64 uint64
	var rev uint64
	var sh uint64
	var r uint64
	c0 = uint64(0)
	isz = (n == c0)
	if isz {
		return c0
	}
	c64 = uint64(64)
	rev = bits.Reverse64(val)
	sh = (c64 - n)
	r = (rev >> sh)
	return r
}

func updateMantissa(unbiased_exp int64, mantissa uint64) uint64 {
	var zero_i int64
	var le0 bool
	var c52 uint64
	var r uint64
	var c51i int64
	var le51 bool
	var c52i int64
	var nfi int64
	var n_frac uint64
	var one uint64
	var shifted uint64
	var frac_mask uint64
	var frac uint64
	var xored uint64
	var revfrac uint64
	zero_i = int64(0)
	le0 = (unbiased_exp <= zero_i)
	if le0 {
		c52 = uint64(52)
		r = reverseBitsN(mantissa, c52)
		return r
	}
	c51i = int64(51)
	le51 = (unbiased_exp <= c51i)
	if le51 {
		c52i = int64(52)
		nfi = (c52i - unbiased_exp)
		n_frac = uint64(nfi)
		one = uint64(1)
		shifted = (one << n_frac)
		frac_mask = (shifted - one)
		frac = (mantissa & frac_mask)
		xored = (mantissa ^ frac)
		revfrac = reverseBitsN(frac, n_frac)
		r = (xored | revfrac)
		return r
	}
	return mantissa
}

func isSimpleFloat(x float64) bool {
	var sb bool
	var inf bool
	var nan bool
	var sb_or_inf bool
	var bad bool
	var false_ bool
	var pow56f float64
	var ge bool
	var i uint64
	var back float64
	var eqv bool
	var pow56 uint64
	var lt bool
	var res bool
	sb = math.Signbit(x)
	inf = math.IsInf(x, 0)
	nan = math.IsNaN(x)
	sb_or_inf = (sb || inf)
	bad = (sb_or_inf || nan)
	false_ = false
	if bad {
		return false_
	}
	pow56f = float64(72057594037927936)
	ge = (x >= pow56f)
	if ge {
		return false_
	}
	i = uint64(x)
	back = float64(i)
	eqv = (back == x)
	pow56 = uint64(72057594037927936)
	lt = (i < pow56)
	res = (eqv && lt)
	return res
}

func floatToIndex(x float64) uint64 {
	var simple bool
	var si uint64
	var bits uint64
	var c52 uint64
	var shr uint64
	var c2047 uint64
	var biased_exp uint64
	var mmask uint64
	var mantissa uint64
	var be_i int64
	var c1023i int64
	var unbiased_exp int64
	var mantissa_enc uint64
	var exp_enc uint64
	var bit63 uint64
	var expsh uint64
	var hi uint64
	var r uint64
	simple = isSimpleFloat(x)
	if simple {
		si = uint64(x)
		return si
	}
	bits = math.Float64bits(x)
	c52 = uint64(52)
	shr = (bits >> c52)
	c2047 = uint64(2047)
	biased_exp = (shr & c2047)
	mmask = uint64(4503599627370495)
	mantissa = (bits & mmask)
	be_i = int64(biased_exp)
	c1023i = int64(1023)
	unbiased_exp = (be_i - c1023i)
	mantissa_enc = updateMantissa(unbiased_exp, mantissa)
	exp_enc = encodeExponent(biased_exp)
	bit63 = uint64(9223372036854775808)
	expsh = (exp_enc << c52)
	hi = (bit63 | expsh)
	r = (hi | mantissa_enc)
	return r
}

func indexToFloat(i uint64) float64 {
	var c63 uint64
	var top uint64
	var c0 uint64
	var isint bool
	var m56 uint64
	var integral uint64
	var f float64
	var c52 uint64
	var shr uint64
	var c2047 uint64
	var exp_enc uint64
	var biased_exp uint64
	var mmask uint64
	var mantissa_enc uint64
	var be_i int64
	var c1023i int64
	var unbiased_exp int64
	var mantissa uint64
	var besh uint64
	var bitsr uint64
	var f2 float64
	c63 = uint64(63)
	top = (i >> c63)
	c0 = uint64(0)
	isint = (top == c0)
	if isint {
		m56 = uint64(72057594037927935)
		integral = (i & m56)
		f = float64(integral)
		return f
	}
	c52 = uint64(52)
	shr = (i >> c52)
	c2047 = uint64(2047)
	exp_enc = (shr & c2047)
	biased_exp = decodeExponent(exp_enc)
	mmask = uint64(4503599627370495)
	mantissa_enc = (i & mmask)
	be_i = int64(biased_exp)
	c1023i = int64(1023)
	unbiased_exp = (be_i - c1023i)
	mantissa = updateMantissa(unbiased_exp, mantissa_enc)
	besh = (biased_exp << c52)
	bitsr = (besh | mantissa)
	f2 = math.Float64frombits(bitsr)
	return f2
}

func minReversedInRange(lo uint64, hi uint64, n int64) uint64 {
	var zero_i int64
	var isz bool
	var c0 uint64
	var one uint64
	var lop1 uint64
	var k_lo uint64
	var k_hi uint64
	var le bool
	var onei int64
	var nm1 int64
	var rec uint64
	var two uint64
	var r uint64
	zero_i = int64(0)
	isz = (n == zero_i)
	c0 = uint64(0)
	if isz {
		return c0
	}
	one = uint64(1)
	lop1 = (lo + one)
	k_lo = (lop1 >> one)
	k_hi = (hi >> one)
	le = (k_lo <= k_hi)
	if le {
		onei = int64(1)
		nm1 = (n - onei)
		rec = minReversedInRange(k_lo, k_hi, nm1)
		two = uint64(2)
		r = (rec * two)
		return r
	}
	return lo
}

func simplestInRange(lo float64, hi float64) float64 {
	var c float64
	var cle bool
	var pow56f float64
	var clt bool
	var inrange bool
	var lo_bits uint64
	var hi_bits uint64
	var c52 uint64
	var e_lo uint64
	var e_hi uint64
	var mmask uint64
	var m_lo uint64
	var m_hi uint64
	var c1023 uint64
	var cond bool
	var ncond bool
	var e uint64
	var m_min uint64
	var m_max uint64
	var eq_hilo bool
	var ne_hilo bool
	var eq_lohi bool
	var ne_lohi bool
	var zero uint64
	var c1023i int64
	var e_i int64
	var unbiased int64
	var c52i int64
	var ge52 bool
	var esh0 uint64
	var bitsr0 uint64
	var f0 float64
	var zero_i int64
	var le0 bool
	var n_frac int64
	var nle0 bool
	var sub int64
	var one uint64
	var shl1 uint64
	var low_mask uint64
	var h uint64
	var l_lo uint64
	var l_hi uint64
	var minrev uint64
	var hsh uint64
	var m_best uint64
	var esh uint64
	var bitsr uint64
	var f float64
	c = math.Ceil(lo)
	cle = (c <= hi)
	pow56f = float64(72057594037927936)
	clt = (c < pow56f)
	inrange = (cle && clt)
	if inrange {
		return c
	}
	lo_bits = math.Float64bits(lo)
	hi_bits = math.Float64bits(hi)
	c52 = uint64(52)
	e_lo = (lo_bits >> c52)
	e_hi = (hi_bits >> c52)
	mmask = uint64(4503599627370495)
	m_lo = (lo_bits & mmask)
	m_hi = (hi_bits & mmask)
	c1023 = uint64(1023)
	cond = (e_lo >= c1023)
	ncond = (!cond)
	e = e_lo
	m_min = m_lo
	m_max = m_hi
	eq_hilo = (e_hi == e_lo)
	ne_hilo = (!eq_hilo)
	if ne_hilo {
		m_max = mmask
	}
	if ncond {
		e = e_hi
		m_max = m_hi
		eq_lohi = (e_lo == e_hi)
		ne_lohi = (!eq_lohi)
		if ne_lohi {
			zero = uint64(0)
			m_min = zero
		}
	}
	c1023i = int64(1023)
	e_i = int64(e)
	unbiased = (e_i - c1023i)
	c52i = int64(52)
	ge52 = (unbiased >= c52i)
	if ge52 {
		esh0 = (e << c52)
		bitsr0 = (esh0 | m_min)
		f0 = math.Float64frombits(bitsr0)
		return f0
	}
	zero_i = int64(0)
	le0 = (unbiased <= zero_i)
	n_frac = c52i
	nle0 = (!le0)
	if nle0 {
		sub = (c52i - unbiased)
		n_frac = sub
	}
	one = uint64(1)
	shl1 = (one << n_frac)
	low_mask = (shl1 - one)
	h = (m_min >> n_frac)
	l_lo = (m_min & low_mask)
	l_hi = (m_max & low_mask)
	minrev = minReversedInRange(l_lo, l_hi, n_frac)
	hsh = (h << n_frac)
	m_best = (hsh | minrev)
	esh = (e << c52)
	bitsr = (esh | m_best)
	f = math.Float64frombits(bitsr)
	return f
}

func base64Encode(data []byte) string {
	var alphabet []byte
	var out []byte
	var n int64
	var i0 int64
	var i1 int64
	var i2 int64
	var i3 int64
	var u0 uint64
	var c6 uint64
	var c8 uint64
	var c12 uint64
	var c16 uint64
	var c18 uint64
	var c3f uint64
	var eq uint64
	var pos int64
	var done bool
	var rem int64
	var b0 uint64
	var b1 uint64
	var b2 uint64
	var has1 bool
	var p1 int64
	var has2 bool
	var p2 int64
	var s16 uint64
	var s8 uint64
	var acc0 uint64
	var acc uint64
	var t0 uint64
	var k0 uint64
	var d0 uint64
	var t1 uint64
	var k1 uint64
	var d1 uint64
	var d2 uint64
	var t2 uint64
	var k2 uint64
	var d3 uint64
	var k3 uint64
	alphabet = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/")
	out = []byte{}
	n = int64(len(data))
	i0 = int64(0)
	i1 = int64(1)
	i2 = int64(2)
	i3 = int64(3)
	u0 = uint64(0)
	c6 = uint64(6)
	c8 = uint64(8)
	c12 = uint64(12)
	c16 = uint64(16)
	c18 = uint64(18)
	c3f = uint64(63)
	eq = uint64(61)
	pos = i0
	for {
		done = (pos >= n)
		if done {
			break
		}
		rem = (n - pos)
		b0 = uint64(data[pos])
		b1 = u0
		b2 = u0
		has1 = (rem > i1)
		if has1 {
			p1 = (pos + i1)
			b1 = uint64(data[p1])
		}
		has2 = (rem > i2)
		if has2 {
			p2 = (pos + i2)
			b2 = uint64(data[p2])
		}
		s16 = (b0 << c16)
		s8 = (b1 << c8)
		acc0 = (s16 | s8)
		acc = (acc0 | b2)
		t0 = (acc >> c18)
		k0 = (t0 & c3f)
		d0 = uint64(alphabet[k0])
		out = append(out, byte(d0))
		t1 = (acc >> c12)
		k1 = (t1 & c3f)
		d1 = uint64(alphabet[k1])
		out = append(out, byte(d1))
		d2 = eq
		if has1 {
			t2 = (acc >> c6)
			k2 = (t2 & c3f)
			d2 = uint64(alphabet[k2])
		}
		out = append(out, byte(d2))
		d3 = eq
		if has2 {
			k3 = (acc & c3f)
			d3 = uint64(alphabet[k3])
		}
		out = append(out, byte(d3))
		pos = (pos + i3)
	}
	return string(out)
}

func base64Value(c uint64) uint64 {
	var a_upper uint64
	var z_upper uint64
	var a_lower uint64
	var z_lower uint64
	var zero uint64
	var nine uint64
	var plus uint64
	var slash uint64
	var c26 uint64
	var c52 uint64
	var c62 uint64
	var c63 uint64
	var invalid uint64
	var value uint64
	var upper_lo bool
	var upper_hi bool
	var is_upper bool
	var lower_lo bool
	var lower_hi bool
	var is_lower bool
	var lower_offset uint64
	var digit_lo bool
	var digit_hi bool
	var is_digit bool
	var digit_offset uint64
	var is_plus bool
	var is_slash bool
	a_upper = uint64(65)
	z_upper = uint64(90)
	a_lower = uint64(97)
	z_lower = uint64(122)
	zero = uint64(48)
	nine = uint64(57)
	plus = uint64(43)
	slash = uint64(47)
	c26 = uint64(26)
	c52 = uint64(52)
	c62 = uint64(62)
	c63 = uint64(63)
	invalid = uint64(64)
	value = invalid
	upper_lo = (c >= a_upper)
	upper_hi = (c <= z_upper)
	is_upper = (upper_lo && upper_hi)
	if is_upper {
		value = (c - a_upper)
	}
	lower_lo = (c >= a_lower)
	lower_hi = (c <= z_lower)
	is_lower = (lower_lo && lower_hi)
	if is_lower {
		lower_offset = (c - a_lower)
		value = (lower_offset + c26)
	}
	digit_lo = (c >= zero)
	digit_hi = (c <= nine)
	is_digit = (digit_lo && digit_hi)
	if is_digit {
		digit_offset = (c - zero)
		value = (digit_offset + c52)
	}
	is_plus = (c == plus)
	if is_plus {
		value = c62
	}
	is_slash = (c == slash)
	if is_slash {
		value = c63
	}
	return value
}

func base64Decode(data []byte) ([]byte, bool) {
	var out []byte
	var none optionalBytes
	var n int64
	var i0 int64
	var i1 int64
	var i2 int64
	var i3 int64
	var i4 int64
	var u0 uint64
	var u6 uint64
	var u8 uint64
	var u12 uint64
	var u16 uint64
	var u18 uint64
	var invalid_value uint64
	var padding uint64
	var rem int64
	var short_ bool
	var bad_length bool
	var pos int64
	var done bool
	var p1 int64
	var p2 int64
	var p3 int64
	var next_ int64
	var last bool
	var b0 uint64
	var b1 uint64
	var b2 uint64
	var b3 uint64
	var c0 uint64
	var c1 uint64
	var bad_c0 bool
	var bad_c1 bool
	var pad2 bool
	var pad3 bool
	var any_padding bool
	var not_last bool
	var padding_before_tail bool
	var not_pad3 bool
	var bad_padding bool
	var c2 uint64
	var not_pad2 bool
	var bad_c2 bool
	var c3 uint64
	var bad_c3 bool
	var s18 uint64
	var s12 uint64
	var s6 uint64
	var acc0 uint64
	var acc1 uint64
	var acc uint64
	var out0 uint64
	var out1 uint64
	var some optionalBytes
	out = []byte{}
	none = optionalBytes{}
	n = int64(len(data))
	i0 = int64(0)
	i1 = int64(1)
	i2 = int64(2)
	i3 = int64(3)
	i4 = int64(4)
	u0 = uint64(0)
	u6 = uint64(6)
	u8 = uint64(8)
	u12 = uint64(12)
	u16 = uint64(16)
	u18 = uint64(18)
	invalid_value = uint64(64)
	padding = uint64(61)
	rem = n
	for {
		short_ = (rem < i4)
		if short_ {
			break
		}
		rem = (rem - i4)
	}
	bad_length = (rem != i0)
	if bad_length {
		return none.value, none.ok
	}
	pos = i0
	for {
		done = (pos >= n)
		if done {
			break
		}
		p1 = (pos + i1)
		p2 = (pos + i2)
		p3 = (pos + i3)
		next_ = (pos + i4)
		last = (next_ == n)
		b0 = uint64(data[pos])
		b1 = uint64(data[p1])
		b2 = uint64(data[p2])
		b3 = uint64(data[p3])
		c0 = base64Value(b0)
		c1 = base64Value(b1)
		bad_c0 = (c0 >= invalid_value)
		if bad_c0 {
			return none.value, none.ok
		}
		bad_c1 = (c1 >= invalid_value)
		if bad_c1 {
			return none.value, none.ok
		}
		pad2 = (b2 == padding)
		pad3 = (b3 == padding)
		any_padding = (pad2 || pad3)
		not_last = (!last)
		padding_before_tail = (any_padding && not_last)
		if padding_before_tail {
			return none.value, none.ok
		}
		not_pad3 = (!pad3)
		bad_padding = (pad2 && not_pad3)
		if bad_padding {
			return none.value, none.ok
		}
		c2 = u0
		not_pad2 = (!pad2)
		if not_pad2 {
			c2 = base64Value(b2)
			bad_c2 = (c2 >= invalid_value)
			if bad_c2 {
				return none.value, none.ok
			}
		}
		c3 = u0
		if not_pad3 {
			c3 = base64Value(b3)
			bad_c3 = (c3 >= invalid_value)
			if bad_c3 {
				return none.value, none.ok
			}
		}
		s18 = (c0 << u18)
		s12 = (c1 << u12)
		s6 = (c2 << u6)
		acc0 = (s18 | s12)
		acc1 = (acc0 | s6)
		acc = (acc1 | c3)
		out0 = (acc >> u16)
		out = append(out, byte(out0))
		if not_pad2 {
			out1 = (acc >> u8)
			out = append(out, byte(out1))
		}
		if not_pad3 {
			out = append(out, byte(acc))
		}
		pos = next_
	}
	some = optionalBytes{value: out, ok: true}
	return some.value, some.ok
}

func u64AsI64(value uint64) int64 {
	var signed_ int64
	signed_ = int64(value)
	return signed_
}

func twosComplementLe(data []byte) []byte {
	var out []byte
	var n int64
	var i0 int64
	var i1 int64
	var u1 uint64
	var u255 uint64
	var carry bool
	var pos int64
	var done bool
	var byte_ uint64
	var inverted uint64
	var sum uint64
	out = []byte{}
	n = int64(len(data))
	i0 = int64(0)
	i1 = int64(1)
	u1 = uint64(1)
	u255 = uint64(255)
	carry = true
	pos = i0
	for {
		done = (pos >= n)
		if done {
			break
		}
		byte_ = uint64(data[pos])
		inverted = (byte_ ^ u255)
		sum = inverted
		if carry {
			sum = (inverted + u1)
			carry = (sum > u255)
		}
		out = append(out, byte(sum))
		pos = (pos + i1)
	}
	return out
}

func canonicalizeSignedLe(data []byte) []byte {
	var out []byte
	var n int64
	var i0 int64
	var i1 int64
	var i2 int64
	var u0 uint64
	var u128 uint64
	var u255 uint64
	var empty bool
	var keep int64
	var one_left bool
	var last_pos int64
	var prev_pos int64
	var last uint64
	var prev uint64
	var prev_sign uint64
	var prev_negative bool
	var prev_positive bool
	var last_zero bool
	var last_ff bool
	var positive_extension bool
	var negative_extension bool
	var redundant bool
	var needed bool
	var pos int64
	var done bool
	var byte_ uint64
	out = []byte{}
	n = int64(len(data))
	i0 = int64(0)
	i1 = int64(1)
	i2 = int64(2)
	u0 = uint64(0)
	u128 = uint64(128)
	u255 = uint64(255)
	empty = (n == i0)
	if empty {
		out = append(out, byte(u0))
		return out
	}
	keep = n
	for {
		one_left = (keep <= i1)
		if one_left {
			break
		}
		last_pos = (keep - i1)
		prev_pos = (keep - i2)
		last = uint64(data[last_pos])
		prev = uint64(data[prev_pos])
		prev_sign = (prev & u128)
		prev_negative = (prev_sign != u0)
		prev_positive = (!prev_negative)
		last_zero = (last == u0)
		last_ff = (last == u255)
		positive_extension = (last_zero && prev_positive)
		negative_extension = (last_ff && prev_negative)
		redundant = (positive_extension || negative_extension)
		needed = (!redundant)
		if needed {
			break
		}
		keep = (keep - i1)
	}
	pos = i0
	for {
		done = (pos >= keep)
		if done {
			break
		}
		byte_ = uint64(data[pos])
		out = append(out, byte(byte_))
		pos = (pos + i1)
	}
	return out
}

func signedAbsLe(data []byte) []byte {
	var value []byte
	var n int64
	var i0 int64
	var i1 int64
	var u0 uint64
	var u128 uint64
	var last_pos int64
	var last uint64
	var sign uint64
	var negative bool
	var positive bool
	var magnitude []byte
	var out []byte
	var pos int64
	var done bool
	var byte_ uint64
	var mag_last uint64
	var mag_sign uint64
	var needs_zero bool
	value = canonicalizeSignedLe(data)
	n = int64(len(value))
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u128 = uint64(128)
	last_pos = (n - i1)
	last = uint64(value[last_pos])
	sign = (last & u128)
	negative = (sign != u0)
	positive = (!negative)
	if positive {
		return value
	}
	magnitude = twosComplementLe(value)
	out = []byte{}
	pos = i0
	for {
		done = (pos >= n)
		if done {
			break
		}
		byte_ = uint64(magnitude[pos])
		out = append(out, byte(byte_))
		pos = (pos + i1)
	}
	mag_last = uint64(magnitude[last_pos])
	mag_sign = (mag_last & u128)
	needs_zero = (mag_sign != u0)
	if needs_zero {
		out = append(out, byte(u0))
	}
	return out
}

func signedNegateLe(data []byte) []byte {
	var value []byte
	var out []byte
	var n int64
	var i0 int64
	var i1 int64
	var u0 uint64
	var u128 uint64
	var u255 uint64
	var last_pos int64
	var last uint64
	var sign uint64
	var negative bool
	var extension uint64
	var pos int64
	var done bool
	var byte_ uint64
	var negated []byte
	var canonical []byte
	value = canonicalizeSignedLe(data)
	out = []byte{}
	n = int64(len(value))
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u128 = uint64(128)
	u255 = uint64(255)
	last_pos = (n - i1)
	last = uint64(value[last_pos])
	sign = (last & u128)
	negative = (sign != u0)
	extension = u0
	if negative {
		extension = u255
	}
	pos = i0
	for {
		done = (pos >= n)
		if done {
			break
		}
		byte_ = uint64(value[pos])
		out = append(out, byte(byte_))
		pos = (pos + i1)
	}
	out = append(out, byte(extension))
	negated = twosComplementLe(out)
	canonical = canonicalizeSignedLe(negated)
	return canonical
}

func signedAddLe(left_raw []byte, right_raw []byte) []byte {
	var left []byte
	var right []byte
	var out []byte
	var left_n int64
	var right_n int64
	var i0 int64
	var i1 int64
	var u0 uint64
	var u8 uint64
	var u128 uint64
	var u255 uint64
	var left_last_pos int64
	var right_last_pos int64
	var left_last uint64
	var right_last uint64
	var left_sign uint64
	var right_sign uint64
	var left_negative bool
	var right_negative bool
	var left_extension uint64
	var right_extension uint64
	var width int64
	var right_longer bool
	var carry uint64
	var pos int64
	var done bool
	var left_byte uint64
	var left_present bool
	var right_byte uint64
	var right_present bool
	var partial uint64
	var sum uint64
	var out_byte uint64
	var canonical []byte
	left = canonicalizeSignedLe(left_raw)
	right = canonicalizeSignedLe(right_raw)
	out = []byte{}
	left_n = int64(len(left))
	right_n = int64(len(right))
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u8 = uint64(8)
	u128 = uint64(128)
	u255 = uint64(255)
	left_last_pos = (left_n - i1)
	right_last_pos = (right_n - i1)
	left_last = uint64(left[left_last_pos])
	right_last = uint64(right[right_last_pos])
	left_sign = (left_last & u128)
	right_sign = (right_last & u128)
	left_negative = (left_sign != u0)
	right_negative = (right_sign != u0)
	left_extension = u0
	if left_negative {
		left_extension = u255
	}
	right_extension = u0
	if right_negative {
		right_extension = u255
	}
	width = left_n
	right_longer = (right_n > left_n)
	if right_longer {
		width = right_n
	}
	width = (width + i1)
	carry = u0
	pos = i0
	for {
		done = (pos >= width)
		if done {
			break
		}
		left_byte = left_extension
		left_present = (pos < left_n)
		if left_present {
			left_byte = uint64(left[pos])
		}
		right_byte = right_extension
		right_present = (pos < right_n)
		if right_present {
			right_byte = uint64(right[pos])
		}
		partial = (left_byte + right_byte)
		sum = (partial + carry)
		out_byte = (sum & u255)
		out = append(out, byte(out_byte))
		carry = (sum >> u8)
		pos = (pos + i1)
	}
	canonical = canonicalizeSignedLe(out)
	return canonical
}

func signedSubLe(left []byte, right []byte) []byte {
	var negative_right []byte
	var result []byte
	negative_right = signedNegateLe(right)
	result = signedAddLe(left, negative_right)
	return result
}

func positiveShr1Le(data []byte) []byte {
	var value []byte
	var out []byte
	var n int64
	var i0 int64
	var i1 int64
	var u1 uint64
	var u7 uint64
	var pos int64
	var done bool
	var current uint64
	var shifted uint64
	var out_byte uint64
	var next_pos int64
	var has_next bool
	var next_ uint64
	var next_low uint64
	var incoming uint64
	var canonical []byte
	value = canonicalizeSignedLe(data)
	out = []byte{}
	n = int64(len(value))
	i0 = int64(0)
	i1 = int64(1)
	u1 = uint64(1)
	u7 = uint64(7)
	pos = i0
	for {
		done = (pos >= n)
		if done {
			break
		}
		current = uint64(value[pos])
		shifted = (current >> u1)
		out_byte = shifted
		next_pos = (pos + i1)
		has_next = (next_pos < n)
		if has_next {
			next_ = uint64(value[next_pos])
			next_low = (next_ & u1)
			incoming = (next_low << u7)
			out_byte = (shifted | incoming)
		}
		out = append(out, byte(out_byte))
		pos = next_pos
	}
	canonical = canonicalizeSignedLe(out)
	return canonical
}

func signedMulLe(left_raw []byte, right_raw []byte) []byte {
	var left_value []byte
	var right_value []byte
	var left_n int64
	var right_n int64
	var i1 int64
	var u0 uint64
	var u1 uint64
	var u128 uint64
	var left_last_pos int64
	var right_last_pos int64
	var left_last uint64
	var right_last uint64
	var left_sign uint64
	var right_sign uint64
	var left_negative bool
	var right_negative bool
	var negative_result bool
	var multiplicand []byte
	var multiplier []byte
	var product []byte
	var low uint64
	var nonzero bool
	var multiplier_n int64
	var more_bytes bool
	var has_value bool
	var done bool
	var low_bit uint64
	var odd bool
	left_value = canonicalizeSignedLe(left_raw)
	right_value = canonicalizeSignedLe(right_raw)
	left_n = int64(len(left_value))
	right_n = int64(len(right_value))
	i1 = int64(1)
	u0 = uint64(0)
	u1 = uint64(1)
	u128 = uint64(128)
	left_last_pos = (left_n - i1)
	right_last_pos = (right_n - i1)
	left_last = uint64(left_value[left_last_pos])
	right_last = uint64(right_value[right_last_pos])
	left_sign = (left_last & u128)
	right_sign = (right_last & u128)
	left_negative = (left_sign != u0)
	right_negative = (right_sign != u0)
	negative_result = (left_negative != right_negative)
	multiplicand = signedAbsLe(left_value)
	multiplier = signedAbsLe(right_value)
	product = []byte{}
	product = append(product, byte(u0))
	for {
		low = uint64(multiplier[u0])
		nonzero = (low != u0)
		multiplier_n = int64(len(multiplier))
		more_bytes = (multiplier_n > i1)
		has_value = (nonzero || more_bytes)
		done = (!has_value)
		if done {
			break
		}
		low_bit = (low & u1)
		odd = (low_bit != u0)
		if odd {
			product = signedAddLe(product, multiplicand)
		}
		multiplicand = signedAddLe(multiplicand, multiplicand)
		multiplier = positiveShr1Le(multiplier)
	}
	if negative_result {
		product = signedNegateLe(product)
	}
	return product
}

func positiveDivLeNonzero(dividend_raw []byte, divisor_raw []byte) []byte {
	var remainder []byte
	var scaled_divisor []byte
	var i0 int64
	var i1 int64
	var u0 uint64
	var u1 uint64
	var quotient []byte
	var bit []byte
	var scale_cmp int64
	var can_scale bool
	var stop_scaling bool
	var bit_low uint64
	var bit_n int64
	var bit_nonzero bool
	var bit_more bool
	var has_bit bool
	var done bool
	var cmp int64
	var can_subtract bool
	remainder = signedAbsLe(dividend_raw)
	scaled_divisor = signedAbsLe(divisor_raw)
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u1 = uint64(1)
	quotient = []byte{}
	quotient = append(quotient, byte(u0))
	bit = []byte{}
	bit = append(bit, byte(u1))
	for {
		scale_cmp = signedCompareLe(scaled_divisor, remainder)
		can_scale = (scale_cmp <= i0)
		stop_scaling = (!can_scale)
		if stop_scaling {
			break
		}
		scaled_divisor = signedAddLe(scaled_divisor, scaled_divisor)
		bit = signedAddLe(bit, bit)
	}
	for {
		scaled_divisor = positiveShr1Le(scaled_divisor)
		bit = positiveShr1Le(bit)
		bit_low = uint64(bit[u0])
		bit_n = int64(len(bit))
		bit_nonzero = (bit_low != u0)
		bit_more = (bit_n > i1)
		has_bit = (bit_nonzero || bit_more)
		done = (!has_bit)
		if done {
			break
		}
		cmp = signedCompareLe(scaled_divisor, remainder)
		can_subtract = (cmp <= i0)
		if can_subtract {
			remainder = signedSubLe(remainder, scaled_divisor)
			quotient = signedAddLe(quotient, bit)
		}
	}
	return quotient
}

func positiveRemLeNonzero(dividend []byte, divisor []byte) []byte {
	var positive_dividend []byte
	var positive_divisor []byte
	var quotient []byte
	var product []byte
	var remainder []byte
	positive_dividend = signedAbsLe(dividend)
	positive_divisor = signedAbsLe(divisor)
	quotient = positiveDivLeNonzero(positive_dividend, positive_divisor)
	product = signedMulLe(quotient, positive_divisor)
	remainder = signedSubLe(positive_dividend, product)
	return remainder
}

func positivePowLe(base_raw []byte, exponent uint64) []byte {
	var u0 uint64
	var u1 uint64
	var result []byte
	var factor []byte
	var power uint64
	var done bool
	var low_bit uint64
	var odd bool
	u0 = uint64(0)
	u1 = uint64(1)
	result = []byte{}
	result = append(result, byte(u1))
	factor = signedAbsLe(base_raw)
	power = exponent
	for {
		done = (power == u0)
		if done {
			break
		}
		low_bit = (power & u1)
		odd = (low_bit != u0)
		if odd {
			result = signedMulLe(result, factor)
		}
		factor = signedMulLe(factor, factor)
		power = (power >> u1)
	}
	return result
}

func signedCompareLe(left_raw []byte, right_raw []byte) int64 {
	var left []byte
	var right []byte
	var i0 int64
	var i1 int64
	var u0 uint64
	var u128 uint64
	var negative_one int64
	var left_n int64
	var right_n int64
	var left_last_pos int64
	var right_last_pos int64
	var left_last uint64
	var right_last uint64
	var left_sign uint64
	var right_sign uint64
	var left_negative bool
	var right_negative bool
	var right_positive bool
	var left_only_negative bool
	var left_positive bool
	var right_only_negative bool
	var left_shorter bool
	var short_result int64
	var left_longer bool
	var long_result int64
	var pos int64
	var done bool
	var left_byte uint64
	var right_byte uint64
	var byte_less bool
	var byte_greater bool
	left = canonicalizeSignedLe(left_raw)
	right = canonicalizeSignedLe(right_raw)
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u128 = uint64(128)
	negative_one = (i0 - i1)
	left_n = int64(len(left))
	right_n = int64(len(right))
	left_last_pos = (left_n - i1)
	right_last_pos = (right_n - i1)
	left_last = uint64(left[left_last_pos])
	right_last = uint64(right[right_last_pos])
	left_sign = (left_last & u128)
	right_sign = (right_last & u128)
	left_negative = (left_sign != u0)
	right_negative = (right_sign != u0)
	right_positive = (!right_negative)
	left_only_negative = (left_negative && right_positive)
	if left_only_negative {
		return negative_one
	}
	left_positive = (!left_negative)
	right_only_negative = (left_positive && right_negative)
	if right_only_negative {
		return i1
	}
	left_shorter = (left_n < right_n)
	if left_shorter {
		short_result = negative_one
		if left_negative {
			short_result = i1
		}
		return short_result
	}
	left_longer = (left_n > right_n)
	if left_longer {
		long_result = i1
		if left_negative {
			long_result = negative_one
		}
		return long_result
	}
	pos = left_n
	for {
		done = (pos == i0)
		if done {
			break
		}
		pos = (pos - i1)
		left_byte = uint64(left[pos])
		right_byte = uint64(right[pos])
		byte_less = (left_byte < right_byte)
		if byte_less {
			return negative_one
		}
		byte_greater = (left_byte > right_byte)
		if byte_greater {
			return i1
		}
	}
	return i0
}

func bigintZero() []byte {
	var out []byte
	var zero uint64
	out = []byte{}
	zero = uint64(0)
	out = append(out, byte(zero))
	return out
}

func bigintOne() []byte {
	var out []byte
	var one uint64
	out = []byte{}
	one = uint64(1)
	out = append(out, byte(one))
	return out
}

func bigintTwo() []byte {
	var out []byte
	var two uint64
	out = []byte{}
	two = uint64(2)
	out = append(out, byte(two))
	return out
}

func bigintBase256() []byte {
	var out []byte
	var zero uint64
	var one uint64
	out = []byte{}
	zero = uint64(0)
	one = uint64(1)
	out = append(out, byte(zero))
	out = append(out, byte(one))
	return out
}

func positiveU8Le(value uint64) []byte {
	var out []byte
	var u0 uint64
	var u128 uint64
	var needs_sign_byte bool
	out = []byte{}
	u0 = uint64(0)
	u128 = uint64(128)
	out = append(out, byte(value))
	needs_sign_byte = (value >= u128)
	if needs_sign_byte {
		out = append(out, byte(u0))
	}
	return out
}

func bigintMin(left []byte, right []byte) []byte {
	var cmp int64
	var i0 int64
	var left_is_min bool
	var out_left []byte
	var out_right []byte
	cmp = signedCompareLe(left, right)
	i0 = int64(0)
	left_is_min = (cmp <= i0)
	if left_is_min {
		out_left = canonicalizeSignedLe(left)
		return out_left
	}
	out_right = canonicalizeSignedLe(right)
	return out_right
}

func integerSimplest(min_value []byte, max_value []byte, shrink_towards []byte) []byte {
	var i0 int64
	var below_cmp int64
	var below bool
	var min_out []byte
	var above_cmp int64
	var above bool
	var max_out []byte
	var target_out []byte
	i0 = int64(0)
	below_cmp = signedCompareLe(shrink_towards, min_value)
	below = (below_cmp < i0)
	if below {
		min_out = canonicalizeSignedLe(min_value)
		return min_out
	}
	above_cmp = signedCompareLe(shrink_towards, max_value)
	above = (above_cmp > i0)
	if above {
		max_out = canonicalizeSignedLe(max_value)
		return max_out
	}
	target_out = canonicalizeSignedLe(shrink_towards)
	return target_out
}

func integerValidate(min_value []byte, max_value []byte, value []byte) bool {
	var i0 int64
	var min_cmp int64
	var max_cmp int64
	var above_min bool
	var below_max bool
	var valid bool
	i0 = int64(0)
	min_cmp = signedCompareLe(min_value, value)
	max_cmp = signedCompareLe(value, max_value)
	above_min = (min_cmp <= i0)
	below_max = (max_cmp <= i0)
	valid = (above_min && below_max)
	return valid
}

func integerUnit(min_value []byte, max_value []byte, shrink_towards []byte) []byte {
	var simplest []byte
	var one []byte
	var successor []byte
	var successor_valid bool
	var predecessor []byte
	var predecessor_valid bool
	simplest = integerSimplest(min_value, max_value, shrink_towards)
	one = bigintOne()
	successor = signedAddLe(simplest, one)
	successor_valid = integerValidate(min_value, max_value, successor)
	if successor_valid {
		return successor
	}
	predecessor = signedSubLe(simplest, one)
	predecessor_valid = integerValidate(min_value, max_value, predecessor)
	if predecessor_valid {
		return predecessor
	}
	return simplest
}

func integerMaxIndex(min_value []byte, max_value []byte) []byte {
	var difference []byte
	var out []byte
	difference = signedSubLe(max_value, min_value)
	out = signedAbsLe(difference)
	return out
}

func integerToIndex(min_value []byte, max_value []byte, shrink_towards []byte, value []byte) []byte {
	var i0 int64
	var simplest []byte
	var value_cmp int64
	var at_simplest bool
	var zero []byte
	var above_delta []byte
	var above []byte
	var below_delta []byte
	var below []byte
	var value_delta []byte
	var distance []byte
	var one []byte
	var distance_minus_one []byte
	var count_above []byte
	var count_below []byte
	var count []byte
	var value_above bool
	var up_index []byte
	var distance_cmp int64
	var up_slot_exists bool
	var down_index []byte
	i0 = int64(0)
	simplest = integerSimplest(min_value, max_value, shrink_towards)
	value_cmp = signedCompareLe(value, simplest)
	at_simplest = (value_cmp == i0)
	if at_simplest {
		zero = bigintZero()
		return zero
	}
	above_delta = signedSubLe(max_value, simplest)
	above = signedAbsLe(above_delta)
	below_delta = signedSubLe(simplest, min_value)
	below = signedAbsLe(below_delta)
	value_delta = signedSubLe(value, simplest)
	distance = signedAbsLe(value_delta)
	one = bigintOne()
	distance_minus_one = signedSubLe(distance, one)
	count_above = bigintMin(distance_minus_one, above)
	count_below = bigintMin(distance_minus_one, below)
	count = signedAddLe(count_above, count_below)
	value_above = (value_cmp > i0)
	if value_above {
		up_index = signedAddLe(count, one)
		return up_index
	}
	distance_cmp = signedCompareLe(distance, above)
	up_slot_exists = (distance_cmp <= i0)
	if up_slot_exists {
		count = signedAddLe(count, one)
	}
	down_index = signedAddLe(count, one)
	return down_index
}

func integerFromIndex(min_value []byte, max_value []byte, shrink_towards []byte, index_raw []byte) ([]byte, bool) {
	var i0 int64
	var simplest []byte
	var index []byte
	var zero []byte
	var index_zero_cmp int64
	var at_zero bool
	var some_simplest optionalBytes
	var above_delta []byte
	var above []byte
	var below_delta []byte
	var below []byte
	var max_index []byte
	var past_cmp int64
	var past_end bool
	var none optionalBytes
	var a []byte
	var two_a []byte
	var two_a_cmp int64
	var interleaved bool
	var distance []byte
	var up bool
	var one []byte
	var two []byte
	var index_plus_one []byte
	var parity []byte
	var parity_cmp int64
	var tail bool
	var range_cmp int64
	var result []byte
	var some optionalBytes
	i0 = int64(0)
	simplest = integerSimplest(min_value, max_value, shrink_towards)
	index = signedAbsLe(index_raw)
	zero = bigintZero()
	index_zero_cmp = signedCompareLe(index, zero)
	at_zero = (index_zero_cmp == i0)
	if at_zero {
		some_simplest = optionalBytes{value: simplest, ok: true}
		return some_simplest.value, some_simplest.ok
	}
	above_delta = signedSubLe(max_value, simplest)
	above = signedAbsLe(above_delta)
	below_delta = signedSubLe(simplest, min_value)
	below = signedAbsLe(below_delta)
	max_index = signedAddLe(above, below)
	past_cmp = signedCompareLe(index, max_index)
	past_end = (past_cmp > i0)
	if past_end {
		none = optionalBytes{}
		return none.value, none.ok
	}
	a = bigintMin(above, below)
	two_a = signedAddLe(a, a)
	two_a_cmp = signedCompareLe(index, two_a)
	interleaved = (two_a_cmp <= i0)
	distance = append([]byte(nil), index...)
	up = false
	if interleaved {
		one = bigintOne()
		two = bigintTwo()
		index_plus_one = signedAddLe(index, one)
		distance = positiveDivLeNonzero(index_plus_one, two)
		parity = positiveRemLeNonzero(index, two)
		parity_cmp = signedCompareLe(parity, zero)
		up = (parity_cmp != i0)
	}
	tail = (!interleaved)
	if tail {
		distance = signedSubLe(index, a)
		range_cmp = signedCompareLe(above, below)
		up = (range_cmp > i0)
	}
	result = signedSubLe(simplest, distance)
	if up {
		result = signedAddLe(simplest, distance)
	}
	some = optionalBytes{value: result, ok: true}
	return some.value, some.ok
}

func integerShrinkExtraDivisor(slot uint64) []byte {
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var u5 uint64
	var u7 uint64
	var u10 uint64
	var divisor uint64
	var is_three bool
	var is_five bool
	var is_seven bool
	var out []byte
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	u5 = uint64(5)
	u7 = uint64(7)
	u10 = uint64(10)
	divisor = u10
	is_three = (slot == u0)
	if is_three {
		divisor = u3
	}
	is_five = (slot == u1)
	if is_five {
		divisor = u5
	}
	is_seven = (slot == u2)
	if is_seven {
		divisor = u7
	}
	out = positiveU8Le(divisor)
	return out
}

func integerTrailingPower(distance []byte, radix uint64) []byte {
	var i0 int64
	var base []byte
	var zero []byte
	var power []byte
	var rest []byte
	var rest_cmp int64
	var exhausted bool
	var remainder []byte
	var remainder_cmp int64
	var divides bool
	var stops bool
	i0 = int64(0)
	base = positiveU8Le(radix)
	zero = bigintZero()
	power = bigintOne()
	rest = signedAbsLe(distance)
	for {
		rest_cmp = signedCompareLe(rest, zero)
		exhausted = (rest_cmp == i0)
		if exhausted {
			break
		}
		remainder = positiveRemLeNonzero(rest, base)
		remainder_cmp = signedCompareLe(remainder, zero)
		divides = (remainder_cmp == i0)
		stops = (!divides)
		if stops {
			break
		}
		rest = positiveDivLeNonzero(rest, base)
		power = signedMulLe(power, base)
	}
	return power
}

func integerShrinkExtraDistance(distance []byte, slot uint64) []byte {
	var i0 int64
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u4 uint64
	var u10 uint64
	var zero []byte
	var one []byte
	var divides bool
	var divisor []byte
	var quotient []byte
	var radix uint64
	var decimal bool
	var power []byte
	var one_cmp int64
	var trivial bool
	var whole_cmp int64
	var whole bool
	var earlier uint64
	var checked bool
	var earlier_divisor []byte
	var earlier_quotient []byte
	var same_cmp int64
	var same bool
	i0 = int64(0)
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u4 = uint64(4)
	u10 = uint64(10)
	zero = bigintZero()
	one = bigintOne()
	divides = (slot < u4)
	if divides {
		divisor = integerShrinkExtraDivisor(slot)
		quotient = positiveDivLeNonzero(distance, divisor)
		return quotient
	}
	radix = u2
	decimal = (slot == u4)
	if decimal {
		radix = u10
	}
	power = integerTrailingPower(distance, radix)
	one_cmp = signedCompareLe(power, one)
	trivial = (one_cmp <= i0)
	if trivial {
		return zero
	}
	whole_cmp = signedCompareLe(power, distance)
	whole = (whole_cmp >= i0)
	if whole {
		return zero
	}
	earlier = u0
	for {
		checked = (earlier >= u4)
		if checked {
			break
		}
		earlier_divisor = integerShrinkExtraDivisor(earlier)
		earlier_quotient = positiveDivLeNonzero(distance, earlier_divisor)
		same_cmp = signedCompareLe(power, earlier_quotient)
		same = (same_cmp == i0)
		if same {
			return zero
		}
		earlier = (earlier + u1)
	}
	return power
}

func integerShrinkExtra(min_value []byte, max_value []byte, shrink_towards []byte, value []byte, ordinal uint64) ([]byte, bool) {
	var i0 int64
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u5 uint64
	var u6 uint64
	var zero []byte
	var prefix []byte
	var simplest []byte
	var offset []byte
	var distance []byte
	var seen uint64
	var slot uint64
	var slots_done bool
	var moved []byte
	var moved_cmp int64
	var moves bool
	var side uint64
	var sides_done bool
	var candidate []byte
	var below bool
	var valid bool
	var index []byte
	var index_cmp int64
	var unprobed bool
	var wanted bool
	var some optionalBytes
	var none optionalBytes
	i0 = int64(0)
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u5 = uint64(5)
	u6 = uint64(6)
	zero = bigintZero()
	prefix = positiveU8Le(u5)
	simplest = integerSimplest(min_value, max_value, shrink_towards)
	offset = signedSubLe(value, simplest)
	distance = signedAbsLe(offset)
	seen = u0
	slot = u0
	for {
		slots_done = (slot >= u6)
		if slots_done {
			break
		}
		moved = integerShrinkExtraDistance(distance, slot)
		moved_cmp = signedCompareLe(moved, zero)
		moves = (moved_cmp > i0)
		if moves {
			side = u0
			for {
				sides_done = (side >= u2)
				if sides_done {
					break
				}
				candidate = signedAddLe(simplest, moved)
				below = (side == u1)
				if below {
					candidate = signedSubLe(simplest, moved)
				}
				valid = integerValidate(min_value, max_value, candidate)
				if valid {
					index = integerToIndex(min_value, max_value, shrink_towards, candidate)
					index_cmp = signedCompareLe(index, prefix)
					unprobed = (index_cmp >= i0)
					if unprobed {
						wanted = (seen == ordinal)
						if wanted {
							some = optionalBytes{value: candidate, ok: true}
							return some.value, some.ok
						}
						seen = (seen + u1)
					}
				}
				side = (side + u1)
			}
		}
		slot = (slot + u1)
	}
	none = optionalBytes{}
	return none.value, none.ok
}

func booleanMaxIndex() []byte {
	var one []byte
	one = bigintOne()
	return one
}

func booleanToIndex(value bool) []byte {
	var one []byte
	var zero []byte
	if value {
		one = bigintOne()
		return one
	}
	zero = bigintZero()
	return zero
}

func booleanFromIndex(index []byte) (bool, bool) {
	var i0 int64
	var zero []byte
	var zero_cmp int64
	var is_zero bool
	var false_ bool
	var some_false optionalBool
	var one []byte
	var one_cmp int64
	var is_one bool
	var true_ bool
	var some_true optionalBool
	var none optionalBool
	i0 = int64(0)
	zero = bigintZero()
	zero_cmp = signedCompareLe(index, zero)
	is_zero = (zero_cmp == i0)
	if is_zero {
		false_ = false
		some_false = optionalBool{value: false_, ok: true}
		return some_false.value, some_false.ok
	}
	one = bigintOne()
	one_cmp = signedCompareLe(index, one)
	is_one = (one_cmp == i0)
	if is_one {
		true_ = true
		some_true = optionalBool{value: true_, ok: true}
		return some_true.value, some_true.ok
	}
	none = optionalBool{}
	return none.value, none.ok
}

func bytesSimplest(min_size int64) []byte {
	var out []byte
	var i0 int64
	var i1 int64
	var u0 uint64
	var pos int64
	var done bool
	out = []byte{}
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	pos = i0
	for {
		done = (pos >= min_size)
		if done {
			break
		}
		out = append(out, byte(u0))
		pos = (pos + i1)
	}
	return out
}

func bytesUnit(min_size int64, max_size int64) []byte {
	var out []byte
	var i0 int64
	var i1 int64
	var u0 uint64
	var u1 uint64
	var has_minimum bool
	var last_pos int64
	var pos int64
	var done bool
	var is_last bool
	var byte_ uint64
	var has_values bool
	out = []byte{}
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u1 = uint64(1)
	has_minimum = (min_size > i0)
	if has_minimum {
		last_pos = (min_size - i1)
		pos = i0
		for {
			done = (pos >= min_size)
			if done {
				break
			}
			is_last = (pos == last_pos)
			byte_ = u0
			if is_last {
				byte_ = u1
			}
			out = append(out, byte(byte_))
			pos = (pos + i1)
		}
		return out
	}
	has_values = (max_size > i0)
	if has_values {
		out = append(out, byte(u1))
	}
	return out
}

func bytesValidate(min_size int64, max_size int64, value []byte) bool {
	var length int64
	var above_min bool
	var below_max bool
	var valid bool
	length = int64(len(value))
	above_min = (length >= min_size)
	below_max = (length <= max_size)
	valid = (above_min && below_max)
	return valid
}

func bytesToIndex(min_size int64, value []byte) []byte {
	var i0 int64
	var i1 int64
	var base []byte
	var offset []byte
	var length int64
	var value_length int64
	var done bool
	var exponent uint64
	var bucket []byte
	var position []byte
	var pos int64
	var byte_ uint64
	var digit []byte
	var shifted []byte
	var index []byte
	i0 = int64(0)
	i1 = int64(1)
	base = bigintBase256()
	offset = bigintZero()
	length = min_size
	value_length = int64(len(value))
	for {
		done = (length >= value_length)
		if done {
			break
		}
		exponent = uint64(length)
		bucket = positivePowLe(base, exponent)
		offset = signedAddLe(offset, bucket)
		length = (length + i1)
	}
	position = bigintZero()
	pos = i0
	for {
		done = (pos >= value_length)
		if done {
			break
		}
		byte_ = uint64(value[pos])
		digit = positiveU8Le(byte_)
		shifted = signedMulLe(position, base)
		position = signedAddLe(shifted, digit)
		pos = (pos + i1)
	}
	index = signedAddLe(offset, position)
	return index
}

func bytesMaxIndex(min_size int64, max_size int64) []byte {
	var last []byte
	var i0 int64
	var i1 int64
	var u255 uint64
	var pos int64
	var done bool
	var index []byte
	last = []byte{}
	i0 = int64(0)
	i1 = int64(1)
	u255 = uint64(255)
	pos = i0
	for {
		done = (pos >= max_size)
		if done {
			break
		}
		last = append(last, byte(u255))
		pos = (pos + i1)
	}
	index = bytesToIndex(min_size, last)
	return index
}

func bytesFromIndex(min_size int64, max_size int64, index_raw []byte) ([]byte, bool) {
	var i0 int64
	var i1 int64
	var base []byte
	var remaining []byte
	var length int64
	var past_end bool
	var exponent uint64
	var bucket []byte
	var bucket_cmp int64
	var inside_bucket bool
	var out []byte
	var empty bool
	var some_empty optionalBytes
	var power int64
	var power_u64 uint64
	var divisor []byte
	var pos int64
	var done bool
	var digit []byte
	var byte_ uint64
	var some optionalBytes
	var none optionalBytes
	i0 = int64(0)
	i1 = int64(1)
	base = bigintBase256()
	remaining = signedAbsLe(index_raw)
	length = min_size
	for {
		past_end = (length > max_size)
		if past_end {
			break
		}
		exponent = uint64(length)
		bucket = positivePowLe(base, exponent)
		bucket_cmp = signedCompareLe(remaining, bucket)
		inside_bucket = (bucket_cmp < i0)
		if inside_bucket {
			out = []byte{}
			empty = (length == i0)
			if empty {
				some_empty = optionalBytes{value: out, ok: true}
				return some_empty.value, some_empty.ok
			}
			power = (length - i1)
			power_u64 = uint64(power)
			divisor = positivePowLe(base, power_u64)
			pos = i0
			for {
				done = (pos >= length)
				if done {
					break
				}
				digit = positiveDivLeNonzero(remaining, divisor)
				byte_ = uint64(digit[i0])
				out = append(out, byte(byte_))
				remaining = positiveRemLeNonzero(remaining, divisor)
				divisor = positiveDivLeNonzero(divisor, base)
				pos = (pos + i1)
			}
			some = optionalBytes{value: out, ok: true}
			return some.value, some.ok
		}
		remaining = signedSubLe(remaining, bucket)
		length = (length + i1)
	}
	none = optionalBytes{}
	return none.value, none.ok
}

func f64Abs(value float64) float64 {
	var mask uint64
	var bits uint64
	var cleared uint64
	var out float64
	mask = uint64(9223372036854775807)
	bits = math.Float64bits(value)
	cleared = (bits & mask)
	out = math.Float64frombits(cleared)
	return out
}

func f64Neg(value float64) float64 {
	var mask uint64
	var bits uint64
	var flipped uint64
	var out float64
	mask = uint64(9223372036854775808)
	bits = math.Float64bits(value)
	flipped = (bits ^ mask)
	out = math.Float64frombits(flipped)
	return out
}

func f64Infinity() float64 {
	var bits uint64
	var out float64
	bits = uint64(9218868437227405312)
	out = math.Float64frombits(bits)
	return out
}

func f64NegInfinity() float64 {
	var positive float64
	var out float64
	positive = f64Infinity()
	out = f64Neg(positive)
	return out
}

func f64Nan() float64 {
	var bits uint64
	var out float64
	bits = uint64(9221120237041090560)
	out = math.Float64frombits(bits)
	return out
}

func f64Max() float64 {
	var bits uint64
	var out float64
	bits = uint64(9218868437227405311)
	out = math.Float64frombits(bits)
	return out
}

func signAwareLte(left float64, right float64) bool {
	var zero float64
	var left_zero bool
	var right_zero bool
	var both_zero bool
	var left_negative bool
	var right_negative bool
	var left_positive bool
	var only_right_negative bool
	var ordered bool
	zero = float64(0)
	left_zero = (left == zero)
	right_zero = (right == zero)
	both_zero = (left_zero && right_zero)
	if both_zero {
		left_negative = math.Signbit(left)
		right_negative = math.Signbit(right)
		left_positive = (!left_negative)
		only_right_negative = (left_positive && right_negative)
		ordered = (!only_right_negative)
		return ordered
	}
	ordered = (left <= right)
	return ordered
}

func floatValidate(min_value float64, max_value float64, allow_nan bool, allow_infinity bool, smallest_nonzero float64, value float64) bool {
	var false_ bool
	var true_ bool
	var is_nan bool
	var is_infinite bool
	var no_infinity bool
	var negative_infinity float64
	var is_negative_infinity bool
	var min_bounded bool
	var reject_low bool
	var positive_infinity float64
	var is_positive_infinity bool
	var max_bounded bool
	var reject_high bool
	var zero float64
	var nonzero bool
	var magnitude float64
	var below_smallest bool
	var reject_tiny bool
	var above_min bool
	var below_max bool
	var valid bool
	false_ = false
	true_ = true
	is_nan = math.IsNaN(value)
	if is_nan {
		return allow_nan
	}
	is_infinite = math.IsInf(value, 0)
	if is_infinite {
		no_infinity = (!allow_infinity)
		if no_infinity {
			return false_
		}
		negative_infinity = f64NegInfinity()
		is_negative_infinity = (value == negative_infinity)
		min_bounded = (min_value > negative_infinity)
		reject_low = (is_negative_infinity && min_bounded)
		if reject_low {
			return false_
		}
		positive_infinity = f64Infinity()
		is_positive_infinity = (value == positive_infinity)
		max_bounded = (max_value < positive_infinity)
		reject_high = (is_positive_infinity && max_bounded)
		if reject_high {
			return false_
		}
		return true_
	}
	zero = float64(0)
	nonzero = (value != zero)
	magnitude = f64Abs(value)
	below_smallest = (magnitude < smallest_nonzero)
	reject_tiny = (nonzero && below_smallest)
	if reject_tiny {
		return false_
	}
	above_min = signAwareLte(min_value, value)
	below_max = signAwareLte(value, max_value)
	valid = (above_min && below_max)
	return valid
}

func floatSimplest(min_value float64, max_value float64, allow_nan bool, allow_infinity bool, smallest_nonzero float64) (float64, bool) {
	var zero float64
	var zero_ok bool
	var found optionalF64
	var negative_zero float64
	var negative_zero_ok bool
	var has_best bool
	var best_index uint64
	var best_value float64
	var largest float64
	var max_positive bool
	var low float64
	var min_below_smallest bool
	var high float64
	var max_above_largest bool
	var ordered bool
	var candidate float64
	var index uint64
	var min_negative bool
	var mirrored_low float64
	var mirror_below_smallest bool
	var mirrored_high float64
	var mirror_above_largest bool
	var no_best bool
	var strictly_lower bool
	var take bool
	var negated float64
	var positive_infinity float64
	var positive_ok bool
	var negative_infinity float64
	var negative_ok bool
	var not_a_number float64
	var missing optionalF64
	zero = float64(0)
	zero_ok = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, zero)
	if zero_ok {
		found = optionalF64{value: zero, ok: true}
		return found.value, found.ok
	}
	negative_zero = f64Neg(zero)
	negative_zero_ok = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, negative_zero)
	if negative_zero_ok {
		found = optionalF64{value: negative_zero, ok: true}
		return found.value, found.ok
	}
	has_best = false
	best_index = uint64(0)
	best_value = zero
	largest = f64Max()
	max_positive = (max_value > zero)
	if max_positive {
		low = min_value
		min_below_smallest = (min_value < smallest_nonzero)
		if min_below_smallest {
			low = smallest_nonzero
		}
		high = max_value
		max_above_largest = (max_value > largest)
		if max_above_largest {
			high = largest
		}
		ordered = (low <= high)
		if ordered {
			candidate = simplestInRange(low, high)
			index = floatToIndex(candidate)
			has_best = true
			best_index = index
			best_value = candidate
		}
	}
	min_negative = (min_value < zero)
	if min_negative {
		mirrored_low = f64Neg(max_value)
		low = mirrored_low
		mirror_below_smallest = (mirrored_low < smallest_nonzero)
		if mirror_below_smallest {
			low = smallest_nonzero
		}
		mirrored_high = f64Neg(min_value)
		high = mirrored_high
		mirror_above_largest = (mirrored_high > largest)
		if mirror_above_largest {
			high = largest
		}
		ordered = (low <= high)
		if ordered {
			candidate = simplestInRange(low, high)
			index = floatToIndex(candidate)
			no_best = (!has_best)
			strictly_lower = (index < best_index)
			take = (no_best || strictly_lower)
			if take {
				negated = f64Neg(candidate)
				has_best = true
				best_index = index
				best_value = negated
			}
		}
	}
	if has_best {
		found = optionalF64{value: best_value, ok: true}
		return found.value, found.ok
	}
	if allow_infinity {
		positive_infinity = f64Infinity()
		positive_ok = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, positive_infinity)
		if positive_ok {
			found = optionalF64{value: positive_infinity, ok: true}
			return found.value, found.ok
		}
		negative_infinity = f64NegInfinity()
		negative_ok = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, negative_infinity)
		if negative_ok {
			found = optionalF64{value: negative_infinity, ok: true}
			return found.value, found.ok
		}
	}
	if allow_nan {
		not_a_number = f64Nan()
		found = optionalF64{value: not_a_number, ok: true}
		return found.value, found.ok
	}
	missing = optionalF64{}
	return missing.value, missing.ok
}

func floatUnit(min_value float64, max_value float64, allow_nan bool, allow_infinity bool, smallest_nonzero float64) (float64, bool) {
	var simplest_opt optionalF64
	var present bool
	var absent bool
	var missing optionalF64
	var simplest float64
	var simplest_is_nan bool
	var found optionalF64
	var u1 uint64
	var u4 uint64
	var magnitude float64
	var base uint64
	var offset uint64
	var exhausted bool
	var index uint64
	var positive float64
	var positive_is_nan bool
	var positive_finite bool
	var positive_ok bool
	var take_positive bool
	var negative float64
	var negative_is_nan bool
	var negative_finite bool
	var negative_ok bool
	var take_negative bool
	var simplest_bits uint64
	var min_is_nan bool
	var min_finite bool
	var min_bits uint64
	var min_distinct bool
	var min_ok bool
	var min_usable bool
	var take_min bool
	var max_is_nan bool
	var max_finite bool
	var max_bits uint64
	var max_distinct bool
	var max_ok bool
	var max_usable bool
	var take_max bool
	var mirrored float64
	var mirrored_is_nan bool
	var mirrored_finite bool
	var mirrored_bits uint64
	var mirrored_distinct bool
	var mirrored_ok bool
	var mirrored_usable bool
	var take_mirrored bool
	simplest_opt.value, simplest_opt.ok = floatSimplest(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero)
	present = simplest_opt.ok
	absent = (!present)
	if absent {
		missing = optionalF64{}
		return missing.value, missing.ok
	}
	simplest = simplest_opt.value
	simplest_is_nan = math.IsNaN(simplest)
	if simplest_is_nan {
		found = optionalF64{value: simplest, ok: true}
		return found.value, found.ok
	}
	u1 = uint64(1)
	u4 = uint64(4)
	magnitude = f64Abs(simplest)
	base = floatToIndex(magnitude)
	offset = u1
	for {
		exhausted = (offset >= u4)
		if exhausted {
			break
		}
		index = (base + offset)
		positive = indexToFloat(index)
		positive_is_nan = math.IsNaN(positive)
		positive_finite = (!positive_is_nan)
		positive_ok = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, positive)
		take_positive = (positive_finite && positive_ok)
		if take_positive {
			found = optionalF64{value: positive, ok: true}
			return found.value, found.ok
		}
		negative = f64Neg(positive)
		negative_is_nan = math.IsNaN(negative)
		negative_finite = (!negative_is_nan)
		negative_ok = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, negative)
		take_negative = (negative_finite && negative_ok)
		if take_negative {
			found = optionalF64{value: negative, ok: true}
			return found.value, found.ok
		}
		offset = (offset + u1)
	}
	simplest_bits = math.Float64bits(simplest)
	min_is_nan = math.IsNaN(min_value)
	min_finite = (!min_is_nan)
	min_bits = math.Float64bits(min_value)
	min_distinct = (min_bits != simplest_bits)
	min_ok = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, min_value)
	min_usable = (min_finite && min_distinct)
	take_min = (min_usable && min_ok)
	if take_min {
		found = optionalF64{value: min_value, ok: true}
		return found.value, found.ok
	}
	max_is_nan = math.IsNaN(max_value)
	max_finite = (!max_is_nan)
	max_bits = math.Float64bits(max_value)
	max_distinct = (max_bits != simplest_bits)
	max_ok = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, max_value)
	max_usable = (max_finite && max_distinct)
	take_max = (max_usable && max_ok)
	if take_max {
		found = optionalF64{value: max_value, ok: true}
		return found.value, found.ok
	}
	mirrored = f64Neg(simplest)
	mirrored_is_nan = math.IsNaN(mirrored)
	mirrored_finite = (!mirrored_is_nan)
	mirrored_bits = math.Float64bits(mirrored)
	mirrored_distinct = (mirrored_bits != simplest_bits)
	mirrored_ok = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, mirrored)
	mirrored_usable = (mirrored_finite && mirrored_distinct)
	take_mirrored = (mirrored_usable && mirrored_ok)
	if take_mirrored {
		found = optionalF64{value: mirrored, ok: true}
		return found.value, found.ok
	}
	found = optionalF64{value: simplest, ok: true}
	return found.value, found.ok
}

func u64ToBigintLe(value uint64) []byte {
	var out []byte
	var i0 int64
	var i1 int64
	var i8 int64
	var u0 uint64
	var u8 uint64
	var u255 uint64
	var shift uint64
	var pos int64
	var done bool
	var shifted uint64
	var byte_ uint64
	var canonical []byte
	out = []byte{}
	i0 = int64(0)
	i1 = int64(1)
	i8 = int64(8)
	u0 = uint64(0)
	u8 = uint64(8)
	u255 = uint64(255)
	shift = u0
	pos = i0
	for {
		done = (pos >= i8)
		if done {
			break
		}
		shifted = (value >> shift)
		byte_ = (shifted & u255)
		out = append(out, byte(byte_))
		shift = (shift + u8)
		pos = (pos + i1)
	}
	out = append(out, byte(u0))
	canonical = canonicalizeSignedLe(out)
	return canonical
}

func bigintToU64Le(value []byte) (uint64, bool) {
	var canonical []byte
	var i0 int64
	var i1 int64
	var i8 int64
	var i9 int64
	var u0 uint64
	var u8 uint64
	var u128 uint64
	var length int64
	var last_pos int64
	var last uint64
	var sign_bit uint64
	var negative bool
	var none optionalU64
	var too_wide bool
	var at_limit bool
	var top uint64
	var magnitude_in_top bool
	var accumulator uint64
	var shift uint64
	var pos int64
	var past_width bool
	var past_end bool
	var byte_ uint64
	var shifted uint64
	var found optionalU64
	canonical = canonicalizeSignedLe(value)
	i0 = int64(0)
	i1 = int64(1)
	i8 = int64(8)
	i9 = int64(9)
	u0 = uint64(0)
	u8 = uint64(8)
	u128 = uint64(128)
	length = int64(len(canonical))
	last_pos = (length - i1)
	last = uint64(canonical[last_pos])
	sign_bit = (last & u128)
	negative = (sign_bit != u0)
	if negative {
		none = optionalU64{}
		return none.value, none.ok
	}
	too_wide = (length > i9)
	if too_wide {
		none = optionalU64{}
		return none.value, none.ok
	}
	at_limit = (length == i9)
	if at_limit {
		top = uint64(canonical[i8])
		magnitude_in_top = (top != u0)
		if magnitude_in_top {
			none = optionalU64{}
			return none.value, none.ok
		}
	}
	accumulator = u0
	shift = u0
	pos = i0
	for {
		past_width = (pos >= i8)
		if past_width {
			break
		}
		past_end = (pos >= length)
		if past_end {
			break
		}
		byte_ = uint64(canonical[pos])
		shifted = (byte_ << shift)
		accumulator = (accumulator | shifted)
		shift = (shift + u8)
		pos = (pos + i1)
	}
	found = optionalU64{value: accumulator, ok: true}
	return found.value, found.ok
}

func maxFiniteGlobalRank() []byte {
	var max_finite_lex uint64
	var lex []byte
	var two []byte
	var one []byte
	var doubled []byte
	var rank []byte
	max_finite_lex = uint64(18442240474082181119)
	lex = u64ToBigintLe(max_finite_lex)
	two = bigintTwo()
	one = bigintOne()
	doubled = signedMulLe(lex, two)
	rank = signedAddLe(doubled, one)
	return rank
}

func floatGlobalRank(value float64) []byte {
	var one []byte
	var two []byte
	var is_nan bool
	var mantissa_mask uint64
	var quiet_bit uint64
	var u63 uint64
	var bits uint64
	var mantissa uint64
	var nan_offset uint64
	var sign uint64
	var base []byte
	var three []byte
	var offset_big []byte
	var sign_big []byte
	var spread []byte
	var with_base []byte
	var with_offset []byte
	var rank []byte
	var is_infinite bool
	var zero float64
	var positive bool
	var negative bool
	var magnitude float64
	var lex uint64
	var lex_big []byte
	var doubled []byte
	one = bigintOne()
	two = bigintTwo()
	is_nan = math.IsNaN(value)
	if is_nan {
		mantissa_mask = uint64(4503599627370495)
		quiet_bit = uint64(2251799813685248)
		u63 = uint64(63)
		bits = math.Float64bits(value)
		mantissa = (bits & mantissa_mask)
		nan_offset = (mantissa ^ quiet_bit)
		sign = (bits >> u63)
		base = maxFiniteGlobalRank()
		three = signedAddLe(two, one)
		offset_big = u64ToBigintLe(nan_offset)
		sign_big = u64ToBigintLe(sign)
		spread = signedMulLe(offset_big, two)
		with_base = signedAddLe(base, three)
		with_offset = signedAddLe(with_base, spread)
		rank = signedAddLe(with_offset, sign_big)
		return rank
	}
	is_infinite = math.IsInf(value, 0)
	if is_infinite {
		zero = float64(0)
		base = maxFiniteGlobalRank()
		positive = (value > zero)
		if positive {
			rank = signedAddLe(base, one)
			return rank
		}
		rank = signedAddLe(base, two)
		return rank
	}
	negative = math.Signbit(value)
	magnitude = f64Abs(value)
	lex = floatToIndex(magnitude)
	lex_big = u64ToBigintLe(lex)
	doubled = signedMulLe(lex_big, two)
	sign_big = bigintZero()
	if negative {
		sign_big = bigintOne()
	}
	rank = signedAddLe(doubled, sign_big)
	return rank
}

func floatFromGlobalRank(rank []byte) (float64, bool) {
	var i0 int64
	var i1 int64
	var u0 uint64
	var u1 uint64
	var u52 uint64
	var u56 uint64
	var u63 uint64
	var one []byte
	var two []byte
	var max_finite []byte
	var ordering int64
	var beyond_finite bool
	var offset []byte
	var is_positive_infinity int64
	var positive_infinity_slot bool
	var infinity float64
	var found optionalF64
	var is_negative_infinity int64
	var negative_infinity_slot bool
	var negative_infinity float64
	var three []byte
	var nan_relative []byte
	var sign_big []byte
	var mantissa_big []byte
	var mantissa_opt optionalU64
	var mantissa_fits bool
	var mantissa_too_wide bool
	var none optionalF64
	var mantissa_base uint64
	var overflow uint64
	var overflows bool
	var quiet_bit uint64
	var exponent_bits uint64
	var mantissa uint64
	var sign_opt optionalU64
	var sign uint64
	var sign_shifted uint64
	var with_exponent uint64
	var bits uint64
	var candidate float64
	var candidate_is_nan bool
	var negative_big []byte
	var magnitude_big []byte
	var magnitude_opt optionalU64
	var magnitude_fits bool
	var magnitude_too_wide bool
	var lex uint64
	var simple_flag uint64
	var simple bool
	var high_bits uint64
	var too_large_for_simple bool
	var unreachable_ bool
	var magnitude float64
	var negative bool
	var mirrored float64
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u1 = uint64(1)
	u52 = uint64(52)
	u56 = uint64(56)
	u63 = uint64(63)
	one = bigintOne()
	two = bigintTwo()
	max_finite = maxFiniteGlobalRank()
	ordering = signedCompareLe(rank, max_finite)
	beyond_finite = (ordering == i1)
	if beyond_finite {
		offset = signedSubLe(rank, max_finite)
		is_positive_infinity = signedCompareLe(offset, one)
		positive_infinity_slot = (is_positive_infinity == i0)
		if positive_infinity_slot {
			infinity = f64Infinity()
			found = optionalF64{value: infinity, ok: true}
			return found.value, found.ok
		}
		is_negative_infinity = signedCompareLe(offset, two)
		negative_infinity_slot = (is_negative_infinity == i0)
		if negative_infinity_slot {
			negative_infinity = f64NegInfinity()
			found = optionalF64{value: negative_infinity, ok: true}
			return found.value, found.ok
		}
		three = signedAddLe(two, one)
		nan_relative = signedSubLe(offset, three)
		sign_big = positiveRemLeNonzero(nan_relative, two)
		mantissa_big = positiveDivLeNonzero(nan_relative, two)
		mantissa_opt.value, mantissa_opt.ok = bigintToU64Le(mantissa_big)
		mantissa_fits = mantissa_opt.ok
		mantissa_too_wide = (!mantissa_fits)
		if mantissa_too_wide {
			none = optionalF64{}
			return none.value, none.ok
		}
		mantissa_base = mantissa_opt.value
		overflow = (mantissa_base >> u52)
		overflows = (overflow != u0)
		if overflows {
			none = optionalF64{}
			return none.value, none.ok
		}
		quiet_bit = uint64(2251799813685248)
		exponent_bits = uint64(9218868437227405312)
		mantissa = (mantissa_base ^ quiet_bit)
		sign_opt.value, sign_opt.ok = bigintToU64Le(sign_big)
		sign = sign_opt.value
		sign_shifted = (sign << u63)
		with_exponent = (sign_shifted | exponent_bits)
		bits = (with_exponent | mantissa)
		candidate = math.Float64frombits(bits)
		candidate_is_nan = math.IsNaN(candidate)
		if candidate_is_nan {
			found = optionalF64{value: candidate, ok: true}
			return found.value, found.ok
		}
		none = optionalF64{}
		return none.value, none.ok
	}
	negative_big = positiveRemLeNonzero(rank, two)
	magnitude_big = positiveDivLeNonzero(rank, two)
	magnitude_opt.value, magnitude_opt.ok = bigintToU64Le(magnitude_big)
	magnitude_fits = magnitude_opt.ok
	magnitude_too_wide = (!magnitude_fits)
	if magnitude_too_wide {
		none = optionalF64{}
		return none.value, none.ok
	}
	lex = magnitude_opt.value
	simple_flag = (lex >> u63)
	simple = (simple_flag == u0)
	high_bits = (lex >> u56)
	too_large_for_simple = (high_bits != u0)
	unreachable_ = (simple && too_large_for_simple)
	if unreachable_ {
		none = optionalF64{}
		return none.value, none.ok
	}
	magnitude = indexToFloat(lex)
	sign_opt.value, sign_opt.ok = bigintToU64Le(negative_big)
	sign = sign_opt.value
	negative = (sign == u1)
	if negative {
		mirrored = f64Neg(magnitude)
		found = optionalF64{value: mirrored, ok: true}
		return found.value, found.ok
	}
	found = optionalF64{value: magnitude, ok: true}
	return found.value, found.ok
}

func floatChoiceMaxIndex() []byte {
	var nan_payloads uint64
	var base []byte
	var two []byte
	var payloads []byte
	var with_infinities []byte
	var max_index []byte
	nan_payloads = uint64(9007199254740992)
	base = maxFiniteGlobalRank()
	two = bigintTwo()
	payloads = u64ToBigintLe(nan_payloads)
	with_infinities = signedAddLe(base, two)
	max_index = signedAddLe(with_infinities, payloads)
	return max_index
}

func floatChoiceToIndex(min_value float64, max_value float64, allow_nan bool, allow_infinity bool, smallest_nonzero float64, value float64) ([]byte, bool) {
	var simplest_opt optionalF64
	var present bool
	var absent bool
	var none optionalBytes
	var simplest float64
	var value_rank []byte
	var simplest_rank []byte
	var index []byte
	var found optionalBytes
	simplest_opt.value, simplest_opt.ok = floatSimplest(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero)
	present = simplest_opt.ok
	absent = (!present)
	if absent {
		none = optionalBytes{}
		return none.value, none.ok
	}
	simplest = simplest_opt.value
	value_rank = floatGlobalRank(value)
	simplest_rank = floatGlobalRank(simplest)
	index = signedSubLe(value_rank, simplest_rank)
	found = optionalBytes{value: index, ok: true}
	return found.value, found.ok
}

func floatChoiceFromIndex(min_value float64, max_value float64, allow_nan bool, allow_infinity bool, smallest_nonzero float64, index []byte) (float64, bool) {
	var simplest_opt optionalF64
	var present bool
	var absent bool
	var none optionalF64
	var simplest float64
	var simplest_rank []byte
	var raw []byte
	var i0 int64
	var zero []byte
	var ordering int64
	var below_zero bool
	var candidate_opt optionalF64
	var decoded bool
	var undecodable bool
	var candidate float64
	var valid bool
	var found optionalF64
	simplest_opt.value, simplest_opt.ok = floatSimplest(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero)
	present = simplest_opt.ok
	absent = (!present)
	if absent {
		none = optionalF64{}
		return none.value, none.ok
	}
	simplest = simplest_opt.value
	simplest_rank = floatGlobalRank(simplest)
	raw = signedAddLe(simplest_rank, index)
	i0 = int64(0)
	zero = bigintZero()
	ordering = signedCompareLe(raw, zero)
	below_zero = (ordering < i0)
	if below_zero {
		none = optionalF64{}
		return none.value, none.ok
	}
	candidate_opt.value, candidate_opt.ok = floatFromGlobalRank(raw)
	decoded = candidate_opt.ok
	undecodable = (!decoded)
	if undecodable {
		none = optionalF64{}
		return none.value, none.ok
	}
	candidate = candidate_opt.value
	valid = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, candidate)
	if valid {
		found = optionalF64{value: candidate, ok: true}
		return found.value, found.ok
	}
	none = optionalF64{}
	return none.value, none.ok
}

func codeLengthCounts(lengths []byte, max_bits uint64) []byte {
	var i0 int64
	var i1 int64
	var i2 int64
	var u0 uint64
	var u1 uint64
	var u8 uint64
	var u255 uint64
	var counts []byte
	var max_bits_i int64
	var slots int64
	var width int64
	var fill int64
	var filled bool
	var n int64
	var pos int64
	var done bool
	var length uint64
	var unused bool
	var too_long bool
	var skip bool
	var countable bool
	var length_i int64
	var offset int64
	var high_offset int64
	var low uint64
	var high uint64
	var shifted uint64
	var current uint64
	var next_ uint64
	var next_low uint64
	var next_shifted uint64
	var next_high uint64
	i0 = int64(0)
	i1 = int64(1)
	i2 = int64(2)
	u0 = uint64(0)
	u1 = uint64(1)
	u8 = uint64(8)
	u255 = uint64(255)
	counts = []byte{}
	max_bits_i = int64(max_bits)
	slots = (max_bits_i + i1)
	width = (slots * i2)
	fill = i0
	for {
		filled = (fill >= width)
		if filled {
			break
		}
		counts = append(counts, byte(u0))
		fill = (fill + i1)
	}
	n = int64(len(lengths))
	pos = i0
	for {
		done = (pos >= n)
		if done {
			break
		}
		length = uint64(lengths[pos])
		unused = (length == u0)
		too_long = (length > max_bits)
		skip = (unused || too_long)
		countable = (!skip)
		if countable {
			length_i = int64(length)
			offset = (length_i * i2)
			high_offset = (offset + i1)
			low = uint64(counts[offset])
			high = uint64(counts[high_offset])
			shifted = (high << u8)
			current = (low | shifted)
			next_ = (current + u1)
			next_low = (next_ & u255)
			next_shifted = (next_ >> u8)
			next_high = (next_shifted & u255)
			counts[offset] = byte(next_low)
			counts[high_offset] = byte(next_high)
		}
		pos = (pos + i1)
	}
	return counts
}

func packedU16At(packed_ []byte, slot uint64) uint64 {
	var i1 int64
	var i2 int64
	var u0 uint64
	var u8 uint64
	var n int64
	var slot_i int64
	var offset int64
	var high_offset int64
	var past_end bool
	var low uint64
	var high uint64
	var shifted uint64
	var value uint64
	i1 = int64(1)
	i2 = int64(2)
	u0 = uint64(0)
	u8 = uint64(8)
	n = int64(len(packed_))
	slot_i = int64(slot)
	offset = (slot_i * i2)
	high_offset = (offset + i1)
	past_end = (high_offset >= n)
	if past_end {
		return u0
	}
	low = uint64(packed_[offset])
	high = uint64(packed_[high_offset])
	shifted = (high << u8)
	value = (low | shifted)
	return value
}

func codeLengthCountAt(counts []byte, length uint64) uint64 {
	var value uint64
	value = packedU16At(counts, length)
	return value
}

func codeLengthsValid(lengths []byte, max_bits uint64) bool {
	var u1 uint64
	var true_ bool
	var false_ bool
	var counts []byte
	var left uint64
	var length uint64
	var done bool
	var doubled uint64
	var count uint64
	var over bool
	u1 = uint64(1)
	true_ = true
	false_ = false
	counts = codeLengthCounts(lengths, max_bits)
	left = u1
	length = u1
	for {
		done = (length > max_bits)
		if done {
			break
		}
		doubled = (left + left)
		count = codeLengthCountAt(counts, length)
		over = (count > doubled)
		if over {
			return false_
		}
		left = (doubled - count)
		length = (length + u1)
	}
	return true_
}

func readBits(data []byte, bit_pos uint64, count uint64) (uint64, bool) {
	var u0 uint64
	var u1 uint64
	var u3 uint64
	var u7 uint64
	var u8 uint64
	var u64 uint64
	var too_wide bool
	var none optionalU64
	var n int64
	var n_u uint64
	var available uint64
	var end_ uint64
	var past_end bool
	var value uint64
	var taken uint64
	var done bool
	var at uint64
	var byte_index uint64
	var byte_offset int64
	var byte_ uint64
	var bit_index uint64
	var shifted uint64
	var bit uint64
	var placed uint64
	var some optionalU64
	u0 = uint64(0)
	u1 = uint64(1)
	u3 = uint64(3)
	u7 = uint64(7)
	u8 = uint64(8)
	u64 = uint64(64)
	too_wide = (count > u64)
	if too_wide {
		none = optionalU64{}
		return none.value, none.ok
	}
	n = int64(len(data))
	n_u = uint64(n)
	available = (n_u * u8)
	end_ = (bit_pos + count)
	past_end = (end_ > available)
	if past_end {
		none = optionalU64{}
		return none.value, none.ok
	}
	value = u0
	taken = u0
	for {
		done = (taken >= count)
		if done {
			break
		}
		at = (bit_pos + taken)
		byte_index = (at >> u3)
		byte_offset = int64(byte_index)
		byte_ = uint64(data[byte_offset])
		bit_index = (at & u7)
		shifted = (byte_ >> bit_index)
		bit = (shifted & u1)
		placed = (bit << taken)
		value = (value | placed)
		taken = (taken + u1)
	}
	some = optionalU64{value: value, ok: true}
	return some.value, some.ok
}

func symbolOffsets(counts []byte, max_bits uint64) []byte {
	var i0 int64
	var i1 int64
	var i2 int64
	var u0 uint64
	var u1 uint64
	var u8 uint64
	var u255 uint64
	var offsets []byte
	var max_bits_i int64
	var slots int64
	var width int64
	var fill int64
	var filled bool
	var running uint64
	var length uint64
	var done bool
	var length_i int64
	var offset int64
	var high_offset int64
	var low uint64
	var shifted uint64
	var high uint64
	var count uint64
	i0 = int64(0)
	i1 = int64(1)
	i2 = int64(2)
	u0 = uint64(0)
	u1 = uint64(1)
	u8 = uint64(8)
	u255 = uint64(255)
	offsets = []byte{}
	max_bits_i = int64(max_bits)
	slots = (max_bits_i + i2)
	width = (slots * i2)
	fill = i0
	for {
		filled = (fill >= width)
		if filled {
			break
		}
		offsets = append(offsets, byte(u0))
		fill = (fill + i1)
	}
	running = u0
	length = u1
	for {
		done = (length > max_bits)
		if done {
			break
		}
		length_i = int64(length)
		offset = (length_i * i2)
		high_offset = (offset + i1)
		low = (running & u255)
		shifted = (running >> u8)
		high = (shifted & u255)
		offsets[offset] = byte(low)
		offsets[high_offset] = byte(high)
		count = codeLengthCountAt(counts, length)
		running = (running + count)
		length = (length + u1)
	}
	return offsets
}

func sortedSymbols(lengths []byte, max_bits uint64) []byte {
	var i0 int64
	var i1 int64
	var i2 int64
	var u0 uint64
	var u1 uint64
	var u8 uint64
	var u255 uint64
	var counts []byte
	var offsets []byte
	var coded uint64
	var length uint64
	var done bool
	var count uint64
	var symbols []byte
	var coded_i int64
	var width int64
	var fill int64
	var filled bool
	var cursor []byte
	var n int64
	var symbol int64
	var unused bool
	var too_long bool
	var skip bool
	var placeable bool
	var slot uint64
	var length_i int64
	var cursor_offset int64
	var cursor_high int64
	var next_ uint64
	var next_low uint64
	var next_shifted uint64
	var next_high uint64
	var slot_i int64
	var slot_offset int64
	var slot_high int64
	var symbol_u uint64
	var sym_low uint64
	var sym_shifted uint64
	var sym_high uint64
	i0 = int64(0)
	i1 = int64(1)
	i2 = int64(2)
	u0 = uint64(0)
	u1 = uint64(1)
	u8 = uint64(8)
	u255 = uint64(255)
	counts = codeLengthCounts(lengths, max_bits)
	offsets = symbolOffsets(counts, max_bits)
	coded = u0
	length = u1
	for {
		done = (length > max_bits)
		if done {
			break
		}
		count = codeLengthCountAt(counts, length)
		coded = (coded + count)
		length = (length + u1)
	}
	symbols = []byte{}
	coded_i = int64(coded)
	width = (coded_i * i2)
	fill = i0
	for {
		filled = (fill >= width)
		if filled {
			break
		}
		symbols = append(symbols, byte(u0))
		fill = (fill + i1)
	}
	cursor = append([]byte(nil), offsets...)
	n = int64(len(lengths))
	symbol = i0
	for {
		done = (symbol >= n)
		if done {
			break
		}
		length = uint64(lengths[symbol])
		unused = (length == u0)
		too_long = (length > max_bits)
		skip = (unused || too_long)
		placeable = (!skip)
		if placeable {
			slot = codeLengthCountAt(cursor, length)
			length_i = int64(length)
			cursor_offset = (length_i * i2)
			cursor_high = (cursor_offset + i1)
			next_ = (slot + u1)
			next_low = (next_ & u255)
			next_shifted = (next_ >> u8)
			next_high = (next_shifted & u255)
			cursor[cursor_offset] = byte(next_low)
			cursor[cursor_high] = byte(next_high)
			slot_i = int64(slot)
			slot_offset = (slot_i * i2)
			slot_high = (slot_offset + i1)
			symbol_u = uint64(symbol)
			sym_low = (symbol_u & u255)
			sym_shifted = (symbol_u >> u8)
			sym_high = (sym_shifted & u255)
			symbols[slot_offset] = byte(sym_low)
			symbols[slot_high] = byte(sym_high)
		}
		symbol = (symbol + i1)
	}
	return symbols
}

func huffmanDecode(data []byte, bit_pos uint64, counts []byte, symbols []byte, max_bits uint64) (uint64, bool) {
	var u0 uint64
	var u1 uint64
	var u32 uint64
	var code uint64
	var first uint64
	var index uint64
	var length uint64
	var done bool
	var at uint64
	var bit_at uint64
	var bit optionalU64
	var have_bit bool
	var truncated bool
	var none optionalU64
	var bit_value uint64
	var count uint64
	var span uint64
	var within bool
	var rank uint64
	var slot uint64
	var symbol uint64
	var shifted_len uint64
	var packed_ uint64
	var some optionalU64
	u0 = uint64(0)
	u1 = uint64(1)
	u32 = uint64(32)
	code = u0
	first = u0
	index = u0
	length = u1
	for {
		done = (length > max_bits)
		if done {
			break
		}
		at = (length - u1)
		bit_at = (bit_pos + at)
		bit.value, bit.ok = readBits(data, bit_at, u1)
		have_bit = bit.ok
		truncated = (!have_bit)
		if truncated {
			none = optionalU64{}
			return none.value, none.ok
		}
		bit_value = bit.value
		code = (code | bit_value)
		count = codeLengthCountAt(counts, length)
		span = (first + count)
		within = (code < span)
		if within {
			rank = (code - first)
			slot = (index + rank)
			symbol = packedU16At(symbols, slot)
			shifted_len = (length << u32)
			packed_ = (shifted_len | symbol)
			some = optionalU64{value: packed_, ok: true}
			return some.value, some.ok
		}
		index = (index + count)
		first = (span + span)
		code = (code + code)
		length = (length + u1)
	}
	none = optionalU64{}
	return none.value, none.ok
}

func lengthBase(symbol uint64) uint64 {
	var u0 uint64
	var u2 uint64
	var u3 uint64
	var u4 uint64
	var u254 uint64
	var u257 uint64
	var u261 uint64
	var u264 uint64
	var u265 uint64
	var u285 uint64
	var u258 uint64
	var below bool
	var above bool
	var outside bool
	var is_max bool
	var flat bool
	var base uint64
	var extra_index uint64
	var extra uint64
	var step uint64
	var within uint64
	var mantissa uint64
	var scaled uint64
	u0 = uint64(0)
	u2 = uint64(2)
	u3 = uint64(3)
	u4 = uint64(4)
	u254 = uint64(254)
	u257 = uint64(257)
	u261 = uint64(261)
	u264 = uint64(264)
	u265 = uint64(265)
	u285 = uint64(285)
	u258 = uint64(258)
	below = (symbol < u257)
	above = (symbol > u285)
	outside = (below || above)
	if outside {
		return u0
	}
	is_max = (symbol == u285)
	if is_max {
		return u258
	}
	flat = (symbol <= u264)
	if flat {
		base = (symbol - u254)
		return base
	}
	extra_index = (symbol - u261)
	extra = (extra_index >> u2)
	step = (symbol - u265)
	within = (step & u3)
	mantissa = (within + u4)
	scaled = (mantissa << extra)
	base = (scaled + u3)
	return base
}

func lengthExtra(symbol uint64) uint64 {
	var u0 uint64
	var u2 uint64
	var u257 uint64
	var u261 uint64
	var u264 uint64
	var u285 uint64
	var below bool
	var above bool
	var outside bool
	var is_max bool
	var flat bool
	var extra_index uint64
	var extra uint64
	u0 = uint64(0)
	u2 = uint64(2)
	u257 = uint64(257)
	u261 = uint64(261)
	u264 = uint64(264)
	u285 = uint64(285)
	below = (symbol < u257)
	above = (symbol > u285)
	outside = (below || above)
	if outside {
		return u0
	}
	is_max = (symbol == u285)
	if is_max {
		return u0
	}
	flat = (symbol <= u264)
	if flat {
		return u0
	}
	extra_index = (symbol - u261)
	extra = (extra_index >> u2)
	return extra
}

func distanceBase(symbol uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var u29 uint64
	var above bool
	var flat bool
	var base uint64
	var extra_index uint64
	var extra uint64
	var parity uint64
	var mantissa uint64
	var scaled uint64
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	u29 = uint64(29)
	above = (symbol > u29)
	if above {
		return u0
	}
	flat = (symbol <= u3)
	if flat {
		base = (symbol + u1)
		return base
	}
	extra_index = (symbol - u2)
	extra = (extra_index >> u1)
	parity = (symbol & u1)
	mantissa = (parity + u2)
	scaled = (mantissa << extra)
	base = (scaled + u1)
	return base
}

func distanceExtra(symbol uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var u29 uint64
	var above bool
	var flat bool
	var extra_index uint64
	var extra uint64
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	u29 = uint64(29)
	above = (symbol > u29)
	if above {
		return u0
	}
	flat = (symbol <= u3)
	if flat {
		return u0
	}
	extra_index = (symbol - u2)
	extra = (extra_index >> u1)
	return extra
}

func fixedLiteralLengths() []byte {
	var i0 int64
	var i1 int64
	var i288 int64
	var i144 int64
	var i256 int64
	var i280 int64
	var u7 uint64
	var u8 uint64
	var u9 uint64
	var lengths []byte
	var symbol int64
	var done bool
	var length uint64
	var past_144 bool
	var below_256 bool
	var is_nine bool
	var past_256 bool
	var below_280 bool
	var is_seven bool
	i0 = int64(0)
	i1 = int64(1)
	i288 = int64(288)
	i144 = int64(144)
	i256 = int64(256)
	i280 = int64(280)
	u7 = uint64(7)
	u8 = uint64(8)
	u9 = uint64(9)
	lengths = []byte{}
	symbol = i0
	for {
		done = (symbol >= i288)
		if done {
			break
		}
		length = u8
		past_144 = (symbol >= i144)
		below_256 = (symbol < i256)
		is_nine = (past_144 && below_256)
		if is_nine {
			length = u9
		}
		past_256 = (symbol >= i256)
		below_280 = (symbol < i280)
		is_seven = (past_256 && below_280)
		if is_seven {
			length = u7
		}
		lengths = append(lengths, byte(length))
		symbol = (symbol + i1)
	}
	return lengths
}

func fixedDistanceLengths() []byte {
	var i0 int64
	var i1 int64
	var i30 int64
	var u5 uint64
	var lengths []byte
	var symbol int64
	var done bool
	i0 = int64(0)
	i1 = int64(1)
	i30 = int64(30)
	u5 = uint64(5)
	lengths = []byte{}
	symbol = i0
	for {
		done = (symbol >= i30)
		if done {
			break
		}
		lengths = append(lengths, byte(u5))
		symbol = (symbol + i1)
	}
	return lengths
}

func codeLengthOrder() []byte {
	var order []byte
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var u4 uint64
	var u5 uint64
	var u6 uint64
	var u7 uint64
	var u8 uint64
	var u9 uint64
	var u10 uint64
	var u11 uint64
	var u12 uint64
	var u13 uint64
	var u14 uint64
	var u15 uint64
	var u16 uint64
	var u17 uint64
	var u18 uint64
	order = []byte{}
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	u4 = uint64(4)
	u5 = uint64(5)
	u6 = uint64(6)
	u7 = uint64(7)
	u8 = uint64(8)
	u9 = uint64(9)
	u10 = uint64(10)
	u11 = uint64(11)
	u12 = uint64(12)
	u13 = uint64(13)
	u14 = uint64(14)
	u15 = uint64(15)
	u16 = uint64(16)
	u17 = uint64(17)
	u18 = uint64(18)
	order = append(order, byte(u16))
	order = append(order, byte(u17))
	order = append(order, byte(u18))
	order = append(order, byte(u0))
	order = append(order, byte(u8))
	order = append(order, byte(u7))
	order = append(order, byte(u9))
	order = append(order, byte(u6))
	order = append(order, byte(u10))
	order = append(order, byte(u5))
	order = append(order, byte(u11))
	order = append(order, byte(u4))
	order = append(order, byte(u12))
	order = append(order, byte(u3))
	order = append(order, byte(u13))
	order = append(order, byte(u2))
	order = append(order, byte(u14))
	order = append(order, byte(u1))
	order = append(order, byte(u15))
	return order
}

func maxDecompressedLen() uint64 {
	var max uint64
	max = uint64(16777216)
	return max
}

func inflate(data []byte) ([]byte, bool) {
	var limit uint64
	var out optionalBytes
	limit = maxDecompressedLen()
	out.value, out.ok = inflateBounded(data, limit)
	return out.value, out.ok
}

func inflateBounded(data []byte, limit uint64) ([]byte, bool) {
	var i0 int64
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var u4 uint64
	var u5 uint64
	var u7 uint64
	var u8 uint64
	var u15 uint64
	var u16 uint64
	var u17 uint64
	var u19 uint64
	var u32 uint64
	var u65535 uint64
	var u256 uint64
	var u257 uint64
	var u285 uint64
	var u286 uint64
	var u29 uint64
	var u30 uint64
	var u11 uint64
	var out []byte
	var bit_pos uint64
	var order []byte
	var final_opt optionalU64
	var have_final bool
	var no_final bool
	var none optionalBytes
	var final_ uint64
	var type_opt optionalU64
	var have_type bool
	var no_type bool
	var block_type uint64
	var reserved bool
	var stored bool
	var slack uint64
	var needs_pad bool
	var pad uint64
	var len_opt optionalU64
	var have_len bool
	var no_len bool
	var len_ uint64
	var nlen_opt optionalU64
	var have_nlen bool
	var no_nlen bool
	var nlen uint64
	var complement uint64
	var agrees bool
	var disagrees bool
	var stored_produced int64
	var stored_produced_u uint64
	var stored_total uint64
	var stored_over bool
	var copied uint64
	var copy_done bool
	var byte_opt optionalU64
	var have_byte bool
	var no_byte bool
	var byte_ uint64
	var compressed bool
	var lit_lengths []byte
	var dist_lengths []byte
	var dynamic bool
	var hlit_opt optionalU64
	var have_hlit bool
	var no_hlit bool
	var hlit_raw uint64
	var hlit uint64
	var hdist_opt optionalU64
	var have_hdist bool
	var no_hdist bool
	var hdist_raw uint64
	var hdist uint64
	var hclen_opt optionalU64
	var have_hclen bool
	var no_hclen bool
	var hclen_raw uint64
	var hclen uint64
	var hlit_bad bool
	var hdist_bad bool
	var header_bad bool
	var cl_lengths []byte
	var zero_fill uint64
	var zeroed bool
	var cl_read uint64
	var cl_done bool
	var cl_opt optionalU64
	var have_cl bool
	var no_cl bool
	var cl_value uint64
	var order_index int64
	var slot uint64
	var cl_ok bool
	var cl_bad bool
	var cl_counts []byte
	var cl_symbols []byte
	var wanted uint64
	var all_lengths []byte
	var have uint64
	var previous uint64
	var enough bool
	var decoded optionalU64
	var have_decoded bool
	var no_decoded bool
	var packed_ uint64
	var used uint64
	var symbol uint64
	var literal bool
	var repeats bool
	var repeat_previous bool
	var extra_width uint64
	var run_base uint64
	var value uint64
	var nothing_yet bool
	var short_zeroes bool
	var extra_opt optionalU64
	var have_extra bool
	var no_extra bool
	var extra uint64
	var run uint64
	var over uint64
	var overruns bool
	var written uint64
	var run_done bool
	var hlit_i int64
	var wanted_i int64
	var lit_ok bool
	var dist_ok bool
	var codes_ok bool
	var codes_bad bool
	var lit_counts []byte
	var lit_symbols []byte
	var dist_counts []byte
	var dist_symbols []byte
	var end_of_block bool
	var is_literal bool
	var literal_produced int64
	var literal_produced_u uint64
	var literal_full bool
	var is_match bool
	var unknown bool
	var base uint64
	var width uint64
	var len_extra_opt optionalU64
	var have_len_extra bool
	var no_len_extra bool
	var len_extra uint64
	var match_length uint64
	var dist_decoded optionalU64
	var have_dist bool
	var no_dist bool
	var dist_packed uint64
	var dist_used uint64
	var dist_symbol uint64
	var dist_unknown bool
	var dist_base uint64
	var dist_width uint64
	var dist_extra_opt optionalU64
	var have_dist_extra bool
	var no_dist_extra bool
	var dist_extra uint64
	var distance uint64
	var produced int64
	var produced_u uint64
	var too_far bool
	var match_total uint64
	var match_over bool
	var from_ uint64
	var taken uint64
	var match_done bool
	var at uint64
	var at_i int64
	var last bool
	var some optionalBytes
	i0 = int64(0)
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	u4 = uint64(4)
	u5 = uint64(5)
	u7 = uint64(7)
	u8 = uint64(8)
	u15 = uint64(15)
	u16 = uint64(16)
	u17 = uint64(17)
	u19 = uint64(19)
	u32 = uint64(32)
	u65535 = uint64(65535)
	u256 = uint64(256)
	u257 = uint64(257)
	u285 = uint64(285)
	u286 = uint64(286)
	u29 = uint64(29)
	u30 = uint64(30)
	u11 = uint64(11)
	out = []byte{}
	bit_pos = u0
	order = codeLengthOrder()
	for {
		final_opt.value, final_opt.ok = readBits(data, bit_pos, u1)
		have_final = final_opt.ok
		no_final = (!have_final)
		if no_final {
			none = optionalBytes{}
			return none.value, none.ok
		}
		final_ = final_opt.value
		bit_pos = (bit_pos + u1)
		type_opt.value, type_opt.ok = readBits(data, bit_pos, u2)
		have_type = type_opt.ok
		no_type = (!have_type)
		if no_type {
			none = optionalBytes{}
			return none.value, none.ok
		}
		block_type = type_opt.value
		bit_pos = (bit_pos + u2)
		reserved = (block_type == u3)
		if reserved {
			none = optionalBytes{}
			return none.value, none.ok
		}
		stored = (block_type == u0)
		if stored {
			slack = (bit_pos & u7)
			needs_pad = (slack > u0)
			if needs_pad {
				pad = (u8 - slack)
				bit_pos = (bit_pos + pad)
			}
			len_opt.value, len_opt.ok = readBits(data, bit_pos, u16)
			have_len = len_opt.ok
			no_len = (!have_len)
			if no_len {
				none = optionalBytes{}
				return none.value, none.ok
			}
			len_ = len_opt.value
			bit_pos = (bit_pos + u16)
			nlen_opt.value, nlen_opt.ok = readBits(data, bit_pos, u16)
			have_nlen = nlen_opt.ok
			no_nlen = (!have_nlen)
			if no_nlen {
				none = optionalBytes{}
				return none.value, none.ok
			}
			nlen = nlen_opt.value
			bit_pos = (bit_pos + u16)
			complement = (len_ ^ u65535)
			agrees = (nlen == complement)
			disagrees = (!agrees)
			if disagrees {
				none = optionalBytes{}
				return none.value, none.ok
			}
			stored_produced = int64(len(out))
			stored_produced_u = uint64(stored_produced)
			stored_total = (stored_produced_u + len_)
			stored_over = (stored_total > limit)
			if stored_over {
				none = optionalBytes{}
				return none.value, none.ok
			}
			copied = u0
			for {
				copy_done = (copied >= len_)
				if copy_done {
					break
				}
				byte_opt.value, byte_opt.ok = readBits(data, bit_pos, u8)
				have_byte = byte_opt.ok
				no_byte = (!have_byte)
				if no_byte {
					none = optionalBytes{}
					return none.value, none.ok
				}
				byte_ = byte_opt.value
				out = append(out, byte(byte_))
				bit_pos = (bit_pos + u8)
				copied = (copied + u1)
			}
		}
		compressed = (!stored)
		if compressed {
			lit_lengths = fixedLiteralLengths()
			dist_lengths = fixedDistanceLengths()
			dynamic = (block_type == u2)
			if dynamic {
				hlit_opt.value, hlit_opt.ok = readBits(data, bit_pos, u5)
				have_hlit = hlit_opt.ok
				no_hlit = (!have_hlit)
				if no_hlit {
					none = optionalBytes{}
					return none.value, none.ok
				}
				hlit_raw = hlit_opt.value
				hlit = (hlit_raw + u257)
				bit_pos = (bit_pos + u5)
				hdist_opt.value, hdist_opt.ok = readBits(data, bit_pos, u5)
				have_hdist = hdist_opt.ok
				no_hdist = (!have_hdist)
				if no_hdist {
					none = optionalBytes{}
					return none.value, none.ok
				}
				hdist_raw = hdist_opt.value
				hdist = (hdist_raw + u1)
				bit_pos = (bit_pos + u5)
				hclen_opt.value, hclen_opt.ok = readBits(data, bit_pos, u4)
				have_hclen = hclen_opt.ok
				no_hclen = (!have_hclen)
				if no_hclen {
					none = optionalBytes{}
					return none.value, none.ok
				}
				hclen_raw = hclen_opt.value
				hclen = (hclen_raw + u4)
				bit_pos = (bit_pos + u4)
				hlit_bad = (hlit > u286)
				hdist_bad = (hdist > u30)
				header_bad = (hlit_bad || hdist_bad)
				if header_bad {
					none = optionalBytes{}
					return none.value, none.ok
				}
				cl_lengths = []byte{}
				zero_fill = u0
				for {
					zeroed = (zero_fill >= u19)
					if zeroed {
						break
					}
					cl_lengths = append(cl_lengths, byte(u0))
					zero_fill = (zero_fill + u1)
				}
				cl_read = u0
				for {
					cl_done = (cl_read >= hclen)
					if cl_done {
						break
					}
					cl_opt.value, cl_opt.ok = readBits(data, bit_pos, u3)
					have_cl = cl_opt.ok
					no_cl = (!have_cl)
					if no_cl {
						none = optionalBytes{}
						return none.value, none.ok
					}
					cl_value = cl_opt.value
					bit_pos = (bit_pos + u3)
					order_index = int64(cl_read)
					slot = uint64(order[order_index])
					cl_lengths[slot] = byte(cl_value)
					cl_read = (cl_read + u1)
				}
				cl_ok = codeLengthsValid(cl_lengths, u7)
				cl_bad = (!cl_ok)
				if cl_bad {
					none = optionalBytes{}
					return none.value, none.ok
				}
				cl_counts = codeLengthCounts(cl_lengths, u7)
				cl_symbols = sortedSymbols(cl_lengths, u7)
				wanted = (hlit + hdist)
				all_lengths = []byte{}
				have = u0
				previous = u0
				for {
					enough = (have >= wanted)
					if enough {
						break
					}
					decoded.value, decoded.ok = huffmanDecode(data, bit_pos, cl_counts, cl_symbols, u7)
					have_decoded = decoded.ok
					no_decoded = (!have_decoded)
					if no_decoded {
						none = optionalBytes{}
						return none.value, none.ok
					}
					packed_ = decoded.value
					used = (packed_ >> u32)
					symbol = (packed_ & u65535)
					bit_pos = (bit_pos + used)
					literal = (symbol < u16)
					if literal {
						all_lengths = append(all_lengths, byte(symbol))
						previous = symbol
						have = (have + u1)
					}
					repeats = (!literal)
					if repeats {
						repeat_previous = (symbol == u16)
						extra_width = u7
						run_base = u11
						value = u0
						if repeat_previous {
							extra_width = u2
							run_base = u3
							value = previous
							nothing_yet = (have == u0)
							if nothing_yet {
								none = optionalBytes{}
								return none.value, none.ok
							}
						}
						short_zeroes = (symbol == u17)
						if short_zeroes {
							extra_width = u3
							run_base = u3
						}
						extra_opt.value, extra_opt.ok = readBits(data, bit_pos, extra_width)
						have_extra = extra_opt.ok
						no_extra = (!have_extra)
						if no_extra {
							none = optionalBytes{}
							return none.value, none.ok
						}
						extra = extra_opt.value
						bit_pos = (bit_pos + extra_width)
						run = (run_base + extra)
						over = (have + run)
						overruns = (over > wanted)
						if overruns {
							none = optionalBytes{}
							return none.value, none.ok
						}
						written = u0
						for {
							run_done = (written >= run)
							if run_done {
								break
							}
							all_lengths = append(all_lengths, byte(value))
							written = (written + u1)
						}
						have = (have + run)
					}
				}
				hlit_i = int64(hlit)
				wanted_i = int64(wanted)
				lit_lengths = append([]byte(nil), all_lengths[i0:hlit_i]...)
				dist_lengths = append([]byte(nil), all_lengths[hlit_i:wanted_i]...)
			}
			lit_ok = codeLengthsValid(lit_lengths, u15)
			dist_ok = codeLengthsValid(dist_lengths, u15)
			codes_ok = (lit_ok && dist_ok)
			codes_bad = (!codes_ok)
			if codes_bad {
				none = optionalBytes{}
				return none.value, none.ok
			}
			lit_counts = codeLengthCounts(lit_lengths, u15)
			lit_symbols = sortedSymbols(lit_lengths, u15)
			dist_counts = codeLengthCounts(dist_lengths, u15)
			dist_symbols = sortedSymbols(dist_lengths, u15)
			for {
				decoded.value, decoded.ok = huffmanDecode(data, bit_pos, lit_counts, lit_symbols, u15)
				have_decoded = decoded.ok
				no_decoded = (!have_decoded)
				if no_decoded {
					none = optionalBytes{}
					return none.value, none.ok
				}
				packed_ = decoded.value
				used = (packed_ >> u32)
				symbol = (packed_ & u65535)
				bit_pos = (bit_pos + used)
				end_of_block = (symbol == u256)
				if end_of_block {
					break
				}
				is_literal = (symbol < u256)
				if is_literal {
					literal_produced = int64(len(out))
					literal_produced_u = uint64(literal_produced)
					literal_full = (literal_produced_u >= limit)
					if literal_full {
						none = optionalBytes{}
						return none.value, none.ok
					}
					out = append(out, byte(symbol))
				}
				is_match = (symbol > u256)
				if is_match {
					unknown = (symbol > u285)
					if unknown {
						none = optionalBytes{}
						return none.value, none.ok
					}
					base = lengthBase(symbol)
					width = lengthExtra(symbol)
					len_extra_opt.value, len_extra_opt.ok = readBits(data, bit_pos, width)
					have_len_extra = len_extra_opt.ok
					no_len_extra = (!have_len_extra)
					if no_len_extra {
						none = optionalBytes{}
						return none.value, none.ok
					}
					len_extra = len_extra_opt.value
					bit_pos = (bit_pos + width)
					match_length = (base + len_extra)
					dist_decoded.value, dist_decoded.ok = huffmanDecode(data, bit_pos, dist_counts, dist_symbols, u15)
					have_dist = dist_decoded.ok
					no_dist = (!have_dist)
					if no_dist {
						none = optionalBytes{}
						return none.value, none.ok
					}
					dist_packed = dist_decoded.value
					dist_used = (dist_packed >> u32)
					dist_symbol = (dist_packed & u65535)
					bit_pos = (bit_pos + dist_used)
					dist_unknown = (dist_symbol > u29)
					if dist_unknown {
						none = optionalBytes{}
						return none.value, none.ok
					}
					dist_base = distanceBase(dist_symbol)
					dist_width = distanceExtra(dist_symbol)
					dist_extra_opt.value, dist_extra_opt.ok = readBits(data, bit_pos, dist_width)
					have_dist_extra = dist_extra_opt.ok
					no_dist_extra = (!have_dist_extra)
					if no_dist_extra {
						none = optionalBytes{}
						return none.value, none.ok
					}
					dist_extra = dist_extra_opt.value
					bit_pos = (bit_pos + dist_width)
					distance = (dist_base + dist_extra)
					produced = int64(len(out))
					produced_u = uint64(produced)
					too_far = (distance > produced_u)
					if too_far {
						none = optionalBytes{}
						return none.value, none.ok
					}
					match_total = (produced_u + match_length)
					match_over = (match_total > limit)
					if match_over {
						none = optionalBytes{}
						return none.value, none.ok
					}
					from_ = (produced_u - distance)
					taken = u0
					for {
						match_done = (taken >= match_length)
						if match_done {
							break
						}
						at = (from_ + taken)
						at_i = int64(at)
						byte_ = uint64(out[at_i])
						out = append(out, byte(byte_))
						taken = (taken + u1)
					}
				}
			}
		}
		last = (final_ > u0)
		if last {
			break
		}
	}
	some = optionalBytes{value: out, ok: true}
	return some.value, some.ok
}

func adler32(data []byte) uint64 {
	var i0 int64
	var i1 int64
	var u0 uint64
	var u1 uint64
	var u16 uint64
	var u65521 uint64
	var low uint64
	var high uint64
	var n int64
	var pos int64
	var done bool
	var byte_ uint64
	var low_sum uint64
	var low_over bool
	var high_sum uint64
	var high_over bool
	var shifted uint64
	var value uint64
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u1 = uint64(1)
	u16 = uint64(16)
	u65521 = uint64(65521)
	low = u1
	high = u0
	n = int64(len(data))
	pos = i0
	for {
		done = (pos >= n)
		if done {
			break
		}
		byte_ = uint64(data[pos])
		low_sum = (low + byte_)
		low = low_sum
		low_over = (low_sum >= u65521)
		if low_over {
			low = (low_sum - u65521)
		}
		high_sum = (high + low)
		high = high_sum
		high_over = (high_sum >= u65521)
		if high_over {
			high = (high_sum - u65521)
		}
		pos = (pos + i1)
	}
	shifted = (high << u16)
	value = (shifted | low)
	return value
}

func zlibDecompress(data []byte) ([]byte, bool) {
	var limit uint64
	var out optionalBytes
	limit = maxDecompressedLen()
	out.value, out.ok = zlibDecompressBounded(data, limit)
	return out.value, out.ok
}

func zlibDecompressBounded(data []byte, limit uint64) ([]byte, bool) {
	var i0 int64
	var i1 int64
	var i2 int64
	var i4 int64
	var i6 int64
	var u0 uint64
	var u8 uint64
	var u15 uint64
	var u31 uint64
	var u32 uint64
	var u256 uint64
	var n int64
	var too_short bool
	var none optionalBytes
	var cmf uint64
	var flg uint64
	var method uint64
	var not_deflate bool
	var scaled uint64
	var header uint64
	var check uint64
	var reduced bool
	var bad_check bool
	var u32_flag uint64
	var has_dict bool
	var body_end int64
	var body []byte
	var inflated optionalBytes
	var present bool
	var missing bool
	var out []byte
	var expected uint64
	var at int64
	var read_done bool
	var byte_ uint64
	var moved uint64
	var actual uint64
	var agrees bool
	var disagrees bool
	var some optionalBytes
	i0 = int64(0)
	i1 = int64(1)
	i2 = int64(2)
	i4 = int64(4)
	i6 = int64(6)
	u0 = uint64(0)
	u8 = uint64(8)
	u15 = uint64(15)
	u31 = uint64(31)
	u32 = uint64(32)
	u256 = uint64(256)
	n = int64(len(data))
	too_short = (n < i6)
	if too_short {
		none = optionalBytes{}
		return none.value, none.ok
	}
	cmf = uint64(data[i0])
	flg = uint64(data[i1])
	method = (cmf & u15)
	not_deflate = (method != u8)
	if not_deflate {
		none = optionalBytes{}
		return none.value, none.ok
	}
	scaled = (cmf * u256)
	header = (scaled + flg)
	check = header
	for {
		reduced = (check < u31)
		if reduced {
			break
		}
		check = (check - u31)
	}
	bad_check = (check != u0)
	if bad_check {
		none = optionalBytes{}
		return none.value, none.ok
	}
	u32_flag = (flg & u32)
	has_dict = (u32_flag > u0)
	if has_dict {
		none = optionalBytes{}
		return none.value, none.ok
	}
	body_end = (n - i4)
	body = append([]byte(nil), data[i2:body_end]...)
	inflated.value, inflated.ok = inflateBounded(body, limit)
	present = inflated.ok
	missing = (!present)
	if missing {
		none = optionalBytes{}
		return none.value, none.ok
	}
	out = inflated.value
	expected = u0
	at = body_end
	for {
		read_done = (at >= n)
		if read_done {
			break
		}
		byte_ = uint64(data[at])
		moved = (expected * u256)
		expected = (moved + byte_)
		at = (at + i1)
	}
	actual = adler32(out)
	agrees = (actual == expected)
	disagrees = (!agrees)
	if disagrees {
		none = optionalBytes{}
		return none.value, none.ok
	}
	some = optionalBytes{value: out, ok: true}
	return some.value, some.ok
}

func deflateWriteBits(state []byte, value uint64, count uint64) []byte {
	var i0 int64
	var i1 int64
	var u0 uint64
	var u1 uint64
	var u8 uint64
	var u255 uint64
	var out []byte
	var n int64
	var copy_pos int64
	var copy_done bool
	var copy_byte uint64
	var remaining uint64
	var written uint64
	var done bool
	var used uint64
	var at_boundary bool
	var out_n int64
	var last int64
	var current uint64
	var bit uint64
	var shifted uint64
	var next_ uint64
	var next_byte uint64
	var next_used_raw uint64
	var next_used uint64
	var full bool
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u1 = uint64(1)
	u8 = uint64(8)
	u255 = uint64(255)
	out = []byte{}
	n = int64(len(state))
	copy_pos = i0
	for {
		copy_done = (copy_pos >= n)
		if copy_done {
			break
		}
		copy_byte = uint64(state[copy_pos])
		out = append(out, byte(copy_byte))
		copy_pos = (copy_pos + i1)
	}
	remaining = value
	written = u0
	for {
		done = (written >= count)
		if done {
			break
		}
		used = uint64(out[i0])
		at_boundary = (used == u0)
		if at_boundary {
			out = append(out, byte(u0))
		}
		out_n = int64(len(out))
		last = (out_n - i1)
		current = uint64(out[last])
		bit = (remaining & u1)
		shifted = (bit << used)
		next_ = (current | shifted)
		next_byte = (next_ & u255)
		out[last] = byte(next_byte)
		next_used_raw = (used + u1)
		next_used = next_used_raw
		full = (next_used_raw == u8)
		if full {
			next_used = u0
		}
		out[i0] = byte(next_used)
		remaining = (remaining >> u1)
		written = (written + u1)
	}
	return out
}

func deflateDrainComplete(state []byte) []byte {
	var i0 int64
	var i1 int64
	var u0 uint64
	var used uint64
	var n int64
	var end_ int64
	var partial bool
	var out []byte
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	used = uint64(state[i0])
	n = int64(len(state))
	end_ = n
	partial = (used != u0)
	if partial {
		end_ = (n - i1)
	}
	out = append([]byte(nil), state[i1:end_]...)
	return out
}

func deflateDrainCarry(state []byte) []byte {
	var i0 int64
	var i1 int64
	var u0 uint64
	var out []byte
	var used uint64
	var partial bool
	var n int64
	var last int64
	var last_byte uint64
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	out = []byte{}
	used = uint64(state[i0])
	out = append(out, byte(used))
	partial = (used != u0)
	if partial {
		n = int64(len(state))
		last = (n - i1)
		last_byte = uint64(state[last])
		out = append(out, byte(last_byte))
	}
	return out
}

func deflateReverseCode(code uint64, width uint64) uint64 {
	var u64 uint64
	var reversed uint64
	var shift uint64
	var out uint64
	u64 = uint64(64)
	reversed = bits.Reverse64(code)
	shift = (u64 - width)
	out = (reversed >> shift)
	return out
}

func deflateFixedSymbol(state []byte, symbol uint64) []byte {
	var u0 uint64
	var u7 uint64
	var u8 uint64
	var u9 uint64
	var u48 uint64
	var u143 uint64
	var u144 uint64
	var u255 uint64
	var u256 uint64
	var u279 uint64
	var u280 uint64
	var u400 uint64
	var u192 uint64
	var code uint64
	var code_width uint64
	var first bool
	var second_low bool
	var second_high bool
	var second bool
	var third_low bool
	var third_high bool
	var third bool
	var fourth bool
	var offset uint64
	var wire_code uint64
	var out []byte
	u0 = uint64(0)
	u7 = uint64(7)
	u8 = uint64(8)
	u9 = uint64(9)
	u48 = uint64(48)
	u143 = uint64(143)
	u144 = uint64(144)
	u255 = uint64(255)
	u256 = uint64(256)
	u279 = uint64(279)
	u280 = uint64(280)
	u400 = uint64(400)
	u192 = uint64(192)
	code = u0
	code_width = u8
	first = (symbol <= u143)
	if first {
		code = (u48 + symbol)
	}
	second_low = (symbol >= u144)
	second_high = (symbol <= u255)
	second = (second_low && second_high)
	if second {
		code = (u400 + symbol)
		code = (code - u144)
		code_width = u9
	}
	third_low = (symbol >= u256)
	third_high = (symbol <= u279)
	third = (third_low && third_high)
	if third {
		code = (symbol - u256)
		code_width = u7
	}
	fourth = (symbol >= u280)
	if fourth {
		offset = (symbol - u280)
		code = (u192 + offset)
		code_width = u8
	}
	wire_code = deflateReverseCode(code, code_width)
	out = deflateWriteBits(state, wire_code, code_width)
	return out
}

func deflateHash(b0 uint64, b1 uint64, b2 uint64) uint64 {
	var u8 uint64
	var u16 uint64
	var u17 uint64
	var u2654435761 uint64
	var u4294967295 uint64
	var b0s uint64
	var b1s uint64
	var partial uint64
	var key uint64
	var mixed uint64
	var mixed32 uint64
	var bucket uint64
	u8 = uint64(8)
	u16 = uint64(16)
	u17 = uint64(17)
	u2654435761 = uint64(2654435761)
	u4294967295 = uint64(4294967295)
	b0s = (b0 << u16)
	b1s = (b1 << u8)
	partial = (b0s | b1s)
	key = (partial | b2)
	mixed = (key * u2654435761)
	mixed32 = (mixed & u4294967295)
	bucket = (mixed32 >> u17)
	return bucket
}

func deflateTokenEntry(tag uint64, value int64, distance int64) []byte {
	var u8 uint64
	var u255 uint64
	var value_u uint64
	var distance_u uint64
	var value_low uint64
	var value_shifted uint64
	var value_high uint64
	var distance_low uint64
	var distance_shifted uint64
	var distance_high uint64
	var entry []byte
	u8 = uint64(8)
	u255 = uint64(255)
	value_u = uint64(value)
	distance_u = uint64(distance)
	value_low = (value_u & u255)
	value_shifted = (value_u >> u8)
	value_high = (value_shifted & u255)
	distance_low = (distance_u & u255)
	distance_shifted = (distance_u >> u8)
	distance_high = (distance_shifted & u255)
	entry = []byte{}
	entry = append(entry, byte(tag))
	entry = append(entry, byte(value_low))
	entry = append(entry, byte(value_high))
	entry = append(entry, byte(distance_low))
	entry = append(entry, byte(distance_high))
	return entry
}

func deflateTokens(data []byte) []byte {
	var i0 int64
	var i1 int64
	var i2 int64
	var i3 int64
	var i5 int64
	var i128 int64
	var i258 int64
	var i32768 int64
	var u0 uint64
	var u1 uint64
	var u4 uint64
	var u8 uint64
	var u16 uint64
	var u24 uint64
	var u255 uint64
	var u32767 uint64
	var u131072 uint64
	var tokens []byte
	var n int64
	var head []byte
	var prev []byte
	var hashed int64
	var pos int64
	var saved_length int64
	var saved_distance int64
	var done bool
	var caught_up bool
	var hash_end int64
	var hashable bool
	var hb0 uint64
	var hb1_at int64
	var hb1 uint64
	var hb2 uint64
	var bucket uint64
	var head_off_u uint64
	var head_off int64
	var head_off1 int64
	var head_off2 int64
	var head_off3 int64
	var old0 uint64
	var old1 uint64
	var old2 uint64
	var old3 uint64
	var p_u uint64
	var p_slot uint64
	var prev_off_u uint64
	var prev_off int64
	var prev_off1 int64
	var prev_off2 int64
	var prev_off3 int64
	var p_b0 uint64
	var p_s1 uint64
	var p_b1 uint64
	var p_s2 uint64
	var p_b2 uint64
	var p_s3 uint64
	var p_b3 uint64
	var has_saved bool
	var threshold int64
	var best_length int64
	var best_distance int64
	var probes_left int64
	var tail2 int64
	var searchable bool
	var window_start int64
	var before_start bool
	var sb0 uint64
	var sb1_at int64
	var sb1 uint64
	var sb2 uint64
	var search_bucket uint64
	var sh_off_u uint64
	var sh_off int64
	var sh_off1 int64
	var sh_off2 int64
	var sh_off3 int64
	var c0 uint64
	var c1 uint64
	var c2 uint64
	var c3 uint64
	var c1s uint64
	var c2s uint64
	var c3s uint64
	var c01 uint64
	var c012 uint64
	var candidate_u uint64
	var candidate int64
	var in_window bool
	var out_of_window bool
	var distance int64
	var matched int64
	var at_limit bool
	var target_at int64
	var past_input bool
	var source_offset int64
	var offset_small bool
	var source_at int64
	var source_byte uint64
	var target_byte uint64
	var different bool
	var better bool
	var maximal bool
	var exhausted bool
	var c_u uint64
	var c_slot uint64
	var pv_off_u uint64
	var pv_off int64
	var pv_off1 int64
	var pv_off2 int64
	var pv_off3 int64
	var pv0 uint64
	var pv1 uint64
	var pv2 uint64
	var pv3 uint64
	var pv1s uint64
	var pv2s uint64
	var pv3s uint64
	var pv01 uint64
	var pv012 uint64
	var next_u uint64
	var found bool
	var current_length int64
	var current_distance int64
	var later_better bool
	var literal_at int64
	var literal_u uint64
	var literal int64
	var entry []byte
	var entry_at int64
	var entry_done bool
	var entry_byte uint64
	var long_ bool
	var short_ bool
	var keep_saved bool
	var remaining int64
	var no_saved bool
	var has_match bool
	var no_match bool
	var pending bool
	i0 = int64(0)
	i1 = int64(1)
	i2 = int64(2)
	i3 = int64(3)
	i5 = int64(5)
	i128 = int64(128)
	i258 = int64(258)
	i32768 = int64(32768)
	u0 = uint64(0)
	u1 = uint64(1)
	u4 = uint64(4)
	u8 = uint64(8)
	u16 = uint64(16)
	u24 = uint64(24)
	u255 = uint64(255)
	u32767 = uint64(32767)
	u131072 = uint64(131072)
	tokens = []byte{}
	n = int64(len(data))
	head = make([]byte, int(u131072))
	prev = make([]byte, int(u131072))
	hashed = i1
	pos = i0
	saved_length = i0
	saved_distance = i0
	for {
		done = (pos >= n)
		if done {
			break
		}
		for {
			caught_up = (hashed >= pos)
			if caught_up {
				break
			}
			hash_end = (hashed + i2)
			hashable = (hash_end < n)
			if hashable {
				hb0 = uint64(data[hashed])
				hb1_at = (hashed + i1)
				hb1 = uint64(data[hb1_at])
				hb2 = uint64(data[hash_end])
				bucket = deflateHash(hb0, hb1, hb2)
				head_off_u = (bucket * u4)
				head_off = int64(head_off_u)
				head_off1 = (head_off + i1)
				head_off2 = (head_off + i2)
				head_off3 = (head_off + i3)
				old0 = uint64(head[head_off])
				old1 = uint64(head[head_off1])
				old2 = uint64(head[head_off2])
				old3 = uint64(head[head_off3])
				p_u = uint64(hashed)
				p_slot = (p_u & u32767)
				prev_off_u = (p_slot * u4)
				prev_off = int64(prev_off_u)
				prev_off1 = (prev_off + i1)
				prev_off2 = (prev_off + i2)
				prev_off3 = (prev_off + i3)
				prev[prev_off] = byte(old0)
				prev[prev_off1] = byte(old1)
				prev[prev_off2] = byte(old2)
				prev[prev_off3] = byte(old3)
				p_b0 = (p_u & u255)
				p_s1 = (p_u >> u8)
				p_b1 = (p_s1 & u255)
				p_s2 = (p_u >> u16)
				p_b2 = (p_s2 & u255)
				p_s3 = (p_u >> u24)
				p_b3 = (p_s3 & u255)
				head[head_off] = byte(p_b0)
				head[head_off1] = byte(p_b1)
				head[head_off2] = byte(p_b2)
				head[head_off3] = byte(p_b3)
			}
			hashed = (hashed + i1)
		}
		has_saved = (saved_length > i0)
		threshold = i2
		if has_saved {
			threshold = saved_length
		}
		best_length = threshold
		best_distance = i0
		probes_left = i128
		tail2 = (pos + i2)
		searchable = (tail2 < n)
		if searchable {
			window_start = (pos - i32768)
			before_start = (window_start < i1)
			if before_start {
				window_start = i1
			}
			sb0 = uint64(data[pos])
			sb1_at = (pos + i1)
			sb1 = uint64(data[sb1_at])
			sb2 = uint64(data[tail2])
			search_bucket = deflateHash(sb0, sb1, sb2)
			sh_off_u = (search_bucket * u4)
			sh_off = int64(sh_off_u)
			sh_off1 = (sh_off + i1)
			sh_off2 = (sh_off + i2)
			sh_off3 = (sh_off + i3)
			c0 = uint64(head[sh_off])
			c1 = uint64(head[sh_off1])
			c2 = uint64(head[sh_off2])
			c3 = uint64(head[sh_off3])
			c1s = (c1 << u8)
			c2s = (c2 << u16)
			c3s = (c3 << u24)
			c01 = (c0 | c1s)
			c012 = (c01 | c2s)
			candidate_u = (c012 | c3s)
			candidate = int64(candidate_u)
			for {
				in_window = (candidate >= window_start)
				out_of_window = (!in_window)
				if out_of_window {
					break
				}
				distance = (pos - candidate)
				matched = i0
				for {
					at_limit = (matched >= i258)
					if at_limit {
						break
					}
					target_at = (pos + matched)
					past_input = (target_at >= n)
					if past_input {
						break
					}
					source_offset = matched
					for {
						offset_small = (source_offset < distance)
						if offset_small {
							break
						}
						source_offset = (source_offset - distance)
					}
					source_at = (candidate + source_offset)
					source_byte = uint64(data[source_at])
					target_byte = uint64(data[target_at])
					different = (source_byte != target_byte)
					if different {
						break
					}
					matched = (matched + i1)
				}
				better = (matched > best_length)
				if better {
					best_length = matched
					best_distance = distance
				}
				maximal = (best_length == i258)
				if maximal {
					break
				}
				probes_left = (probes_left - i1)
				exhausted = (probes_left <= i0)
				if exhausted {
					break
				}
				c_u = uint64(candidate)
				c_slot = (c_u & u32767)
				pv_off_u = (c_slot * u4)
				pv_off = int64(pv_off_u)
				pv_off1 = (pv_off + i1)
				pv_off2 = (pv_off + i2)
				pv_off3 = (pv_off + i3)
				pv0 = uint64(prev[pv_off])
				pv1 = uint64(prev[pv_off1])
				pv2 = uint64(prev[pv_off2])
				pv3 = uint64(prev[pv_off3])
				pv1s = (pv1 << u8)
				pv2s = (pv2 << u16)
				pv3s = (pv3 << u24)
				pv01 = (pv0 | pv1s)
				pv012 = (pv01 | pv2s)
				next_u = (pv012 | pv3s)
				candidate = int64(next_u)
			}
		}
		found = (best_distance > i0)
		current_length = i0
		current_distance = i0
		if found {
			current_length = best_length
			current_distance = best_distance
		}
		if has_saved {
			later_better = (current_length > saved_length)
			if later_better {
				literal_at = (pos - i1)
				literal_u = uint64(data[literal_at])
				literal = int64(literal_u)
				entry = deflateTokenEntry(u0, literal, i0)
				entry_at = i0
				for {
					entry_done = (entry_at >= i5)
					if entry_done {
						break
					}
					entry_byte = uint64(entry[entry_at])
					tokens = append(tokens, byte(entry_byte))
					entry_at = (entry_at + i1)
				}
				long_ = (current_length >= i128)
				if long_ {
					entry = deflateTokenEntry(u1, current_length, current_distance)
					entry_at = i0
					for {
						entry_done = (entry_at >= i5)
						if entry_done {
							break
						}
						entry_byte = uint64(entry[entry_at])
						tokens = append(tokens, byte(entry_byte))
						entry_at = (entry_at + i1)
					}
					pos = (pos + current_length)
					saved_length = i0
					saved_distance = i0
				}
				short_ = (!long_)
				if short_ {
					saved_length = current_length
					saved_distance = current_distance
					pos = (pos + i1)
				}
			}
			keep_saved = (!later_better)
			if keep_saved {
				entry = deflateTokenEntry(u1, saved_length, saved_distance)
				entry_at = i0
				for {
					entry_done = (entry_at >= i5)
					if entry_done {
						break
					}
					entry_byte = uint64(entry[entry_at])
					tokens = append(tokens, byte(entry_byte))
					entry_at = (entry_at + i1)
				}
				remaining = (saved_length - i1)
				pos = (pos + remaining)
				saved_length = i0
				saved_distance = i0
			}
		}
		no_saved = (!has_saved)
		if no_saved {
			has_match = (current_length >= i3)
			if has_match {
				long_ = (current_length >= i128)
				if long_ {
					entry = deflateTokenEntry(u1, current_length, current_distance)
					entry_at = i0
					for {
						entry_done = (entry_at >= i5)
						if entry_done {
							break
						}
						entry_byte = uint64(entry[entry_at])
						tokens = append(tokens, byte(entry_byte))
						entry_at = (entry_at + i1)
					}
					pos = (pos + current_length)
				}
				short_ = (!long_)
				if short_ {
					saved_length = current_length
					saved_distance = current_distance
					pos = (pos + i1)
				}
			}
			no_match = (!has_match)
			if no_match {
				literal_u = uint64(data[pos])
				literal = int64(literal_u)
				entry = deflateTokenEntry(u0, literal, i0)
				entry_at = i0
				for {
					entry_done = (entry_at >= i5)
					if entry_done {
						break
					}
					entry_byte = uint64(entry[entry_at])
					tokens = append(tokens, byte(entry_byte))
					entry_at = (entry_at + i1)
				}
				pos = (pos + i1)
			}
		}
	}
	pending = (saved_length > i0)
	if pending {
		entry = deflateTokenEntry(u1, saved_length, saved_distance)
		entry_at = i0
		for {
			entry_done = (entry_at >= i5)
			if entry_done {
				break
			}
			entry_byte = uint64(entry[entry_at])
			tokens = append(tokens, byte(entry_byte))
			entry_at = (entry_at + i1)
		}
	}
	return tokens
}

func deflateFixed(data []byte) []byte {
	var i0 int64
	var i1 int64
	var i2 int64
	var i3 int64
	var i4 int64
	var i5 int64
	var u0 uint64
	var u1 uint64
	var u3 uint64
	var u5 uint64
	var u8 uint64
	var u256 uint64
	var u257 uint64
	var u258 uint64
	var u285 uint64
	var u30 uint64
	var out []byte
	var state []byte
	var tokens []byte
	var n int64
	var pos int64
	var done bool
	var tag uint64
	var is_match bool
	var value_low_at int64
	var value_high_at int64
	var distance_low_at int64
	var distance_high_at int64
	var value_low uint64
	var value_high uint64
	var value_high_shifted uint64
	var length_u uint64
	var distance_low uint64
	var distance_high uint64
	var distance_high_shifted uint64
	var distance_u uint64
	var length_symbol uint64
	var last_length bool
	var base uint64
	var extra_count uint64
	var span uint64
	var upper uint64
	var fits bool
	var maximum_length bool
	var length_base_value uint64
	var length_extra_count uint64
	var length_extra_value uint64
	var distance_symbol uint64
	var last_distance bool
	var distance_base_value uint64
	var distance_extra_count uint64
	var distance_span uint64
	var distance_upper uint64
	var distance_fits bool
	var distance_wire uint64
	var distance_base_final uint64
	var distance_extra_final uint64
	var distance_value uint64
	var is_literal bool
	var literal_at int64
	var literal uint64
	var complete []byte
	var complete_n int64
	var drain_at int64
	var drain_done bool
	var drain_byte uint64
	var state_n int64
	var tail_at int64
	var tail_done bool
	var tail_byte uint64
	i0 = int64(0)
	i1 = int64(1)
	i2 = int64(2)
	i3 = int64(3)
	i4 = int64(4)
	i5 = int64(5)
	u0 = uint64(0)
	u1 = uint64(1)
	u3 = uint64(3)
	u5 = uint64(5)
	u8 = uint64(8)
	u256 = uint64(256)
	u257 = uint64(257)
	u258 = uint64(258)
	u285 = uint64(285)
	u30 = uint64(30)
	out = []byte{}
	state = []byte{}
	state = append(state, byte(u0))
	state = deflateWriteBits(state, u3, u3)
	tokens = deflateTokens(data)
	n = int64(len(tokens))
	pos = i0
	for {
		done = (pos >= n)
		if done {
			break
		}
		tag = uint64(tokens[pos])
		is_match = (tag == u1)
		if is_match {
			value_low_at = (pos + i1)
			value_high_at = (pos + i2)
			distance_low_at = (pos + i3)
			distance_high_at = (pos + i4)
			value_low = uint64(tokens[value_low_at])
			value_high = uint64(tokens[value_high_at])
			value_high_shifted = (value_high << u8)
			length_u = (value_low | value_high_shifted)
			distance_low = uint64(tokens[distance_low_at])
			distance_high = uint64(tokens[distance_high_at])
			distance_high_shifted = (distance_high << u8)
			distance_u = (distance_low | distance_high_shifted)
			length_symbol = u257
			for {
				last_length = (length_symbol >= u285)
				if last_length {
					break
				}
				base = lengthBase(length_symbol)
				extra_count = lengthExtra(length_symbol)
				span = (u1 << extra_count)
				upper = (base + span)
				fits = (length_u < upper)
				if fits {
					break
				}
				length_symbol = (length_symbol + u1)
			}
			maximum_length = (length_u == u258)
			if maximum_length {
				length_symbol = u285
			}
			state = deflateFixedSymbol(state, length_symbol)
			length_base_value = lengthBase(length_symbol)
			length_extra_count = lengthExtra(length_symbol)
			length_extra_value = (length_u - length_base_value)
			state = deflateWriteBits(state, length_extra_value, length_extra_count)
			distance_symbol = u0
			for {
				last_distance = (distance_symbol >= u30)
				if last_distance {
					break
				}
				distance_base_value = distanceBase(distance_symbol)
				distance_extra_count = distanceExtra(distance_symbol)
				distance_span = (u1 << distance_extra_count)
				distance_upper = (distance_base_value + distance_span)
				distance_fits = (distance_u < distance_upper)
				if distance_fits {
					break
				}
				distance_symbol = (distance_symbol + u1)
			}
			distance_wire = deflateReverseCode(distance_symbol, u5)
			state = deflateWriteBits(state, distance_wire, u5)
			distance_base_final = distanceBase(distance_symbol)
			distance_extra_final = distanceExtra(distance_symbol)
			distance_value = (distance_u - distance_base_final)
			state = deflateWriteBits(state, distance_value, distance_extra_final)
		}
		is_literal = (!is_match)
		if is_literal {
			literal_at = (pos + i1)
			literal = uint64(tokens[literal_at])
			state = deflateFixedSymbol(state, literal)
		}
		complete = deflateDrainComplete(state)
		complete_n = int64(len(complete))
		drain_at = i0
		for {
			drain_done = (drain_at >= complete_n)
			if drain_done {
				break
			}
			drain_byte = uint64(complete[drain_at])
			out = append(out, byte(drain_byte))
			drain_at = (drain_at + i1)
		}
		state = deflateDrainCarry(state)
		pos = (pos + i5)
	}
	state = deflateFixedSymbol(state, u256)
	state_n = int64(len(state))
	tail_at = i1
	for {
		tail_done = (tail_at >= state_n)
		if tail_done {
			break
		}
		tail_byte = uint64(state[tail_at])
		out = append(out, byte(tail_byte))
		tail_at = (tail_at + i1)
	}
	return out
}

func deflateU16Zeros(count uint64) []byte {
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var out []byte
	var width uint64
	var at uint64
	var done bool
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	out = []byte{}
	width = (count * u2)
	at = u0
	for {
		done = (at >= width)
		if done {
			break
		}
		out = append(out, byte(u0))
		at = (at + u1)
	}
	return out
}

func deflateU16Get(data []byte, index uint64) uint64 {
	var i1 int64
	var u8 uint64
	var u2 uint64
	var offset_u uint64
	var offset int64
	var high_at int64
	var low uint64
	var high uint64
	var shifted uint64
	var value uint64
	i1 = int64(1)
	u8 = uint64(8)
	u2 = uint64(2)
	offset_u = (index * u2)
	offset = int64(offset_u)
	high_at = (offset + i1)
	low = uint64(data[offset])
	high = uint64(data[high_at])
	shifted = (high << u8)
	value = (low | shifted)
	return value
}

func deflateU16Set(data []byte, index uint64, value uint64) []byte {
	var i1 int64
	var u2 uint64
	var u8 uint64
	var u255 uint64
	var i0 int64
	var out []byte
	var n int64
	var copy_at int64
	var copy_done bool
	var copy_byte uint64
	var offset_u uint64
	var offset int64
	var high_at int64
	var low uint64
	var shifted uint64
	var high uint64
	i1 = int64(1)
	u2 = uint64(2)
	u8 = uint64(8)
	u255 = uint64(255)
	i0 = int64(0)
	out = []byte{}
	n = int64(len(data))
	copy_at = i0
	for {
		copy_done = (copy_at >= n)
		if copy_done {
			break
		}
		copy_byte = uint64(data[copy_at])
		out = append(out, byte(copy_byte))
		copy_at = (copy_at + i1)
	}
	offset_u = (index * u2)
	offset = int64(offset_u)
	high_at = (offset + i1)
	low = (value & u255)
	shifted = (value >> u8)
	high = (shifted & u255)
	out[offset] = byte(low)
	out[high_at] = byte(high)
	return out
}

func deflateU16Increment(data []byte, index uint64, amount uint64) []byte {
	var u65535 uint64
	var old uint64
	var sum uint64
	var wrapped uint64
	var out []byte
	u65535 = uint64(65535)
	old = deflateU16Get(data, index)
	sum = (old + amount)
	wrapped = (sum & u65535)
	out = deflateU16Set(data, index, wrapped)
	return out
}

func deflateRecordKeyGet(records []byte, record uint64) uint64 {
	var u2 uint64
	var word uint64
	var key uint64
	u2 = uint64(2)
	word = (record * u2)
	key = deflateU16Get(records, word)
	return key
}

func deflateRecordKeySet(records []byte, record uint64, key uint64) []byte {
	var u2 uint64
	var word uint64
	var out []byte
	u2 = uint64(2)
	word = (record * u2)
	out = deflateU16Set(records, word, key)
	return out
}

func deflateOptimizedLengths(counts []byte, table_len uint64, limit uint64) []byte {
	var i0 int64
	var i1 int64
	var i2 int64
	var i3 int64
	var i4 int64
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u4 uint64
	var u8 uint64
	var u33 uint64
	var u255 uint64
	var u65535 uint64
	var used []byte
	var used_count uint64
	var symbol uint64
	var done bool
	var frequency uint64
	var present bool
	var frequency_low uint64
	var frequency_shifted uint64
	var frequency_high uint64
	var symbol_low uint64
	var symbol_shifted uint64
	var symbol_high uint64
	var used_count_i int64
	var chosen []byte
	var chosen_at uint64
	var chosen_done bool
	var sorted []byte
	var slot uint64
	var sorted_done bool
	var scan uint64
	var best_index uint64
	var best_frequency uint64
	var have_best uint64
	var scan_done bool
	var scan_i int64
	var already uint64
	var unchosen bool
	var record_u uint64
	var scan_record int64
	var record_high int64
	var low uint64
	var high uint64
	var high_shifted uint64
	var first_candidate bool
	var lower bool
	var take bool
	var best_i int64
	var best_record_u uint64
	var best_record int64
	var copy_ int64
	var copy_done bool
	var source int64
	var byte_ uint64
	var one bool
	var many bool
	var f0 uint64
	var f1 uint64
	var initial_sum uint64
	var initial_wrapped uint64
	var root uint64
	var leaf uint64
	var next_ uint64
	var last_internal uint64
	var merge_done bool
	var leaf_end bool
	var take_root bool
	var leaf_available bool
	var root_key uint64
	var leaf_key uint64
	var root_lower bool
	var take_leaf bool
	var leaf_end2 bool
	var take_root2 bool
	var leaf_available2 bool
	var root_before_next bool
	var root_key2 uint64
	var leaf_key2 uint64
	var root_lower2 bool
	var next_key uint64
	var combined uint64
	var combined_wrapped uint64
	var take_leaf2 bool
	var n_minus2 uint64
	var back uint64
	var back_done bool
	var parent uint64
	var parent_depth uint64
	var depth uint64
	var slots_available uint64
	var used_here uint64
	var root_i int64
	var next_i int64
	var assign_done bool
	var root_negative bool
	var root_u uint64
	var root_depth uint64
	var different_depth bool
	var enough bool
	var next_u uint64
	var num_codes []byte
	var count_record uint64
	var count_record_done bool
	var count_depth uint64
	var over uint64
	var over_sum uint64
	var over_done bool
	var amount uint64
	var limit_count uint64
	var limit_total uint64
	var total uint64
	var width uint64
	var width_done bool
	var count uint64
	var shift uint64
	var weighted uint64
	var capacity uint64
	var balanced bool
	var limit_less uint64
	var seek uint64
	var seek_count uint64
	var found bool
	var seek_less uint64
	var deeper uint64
	var deeper_count uint64
	var deeper_more uint64
	var lengths []byte
	var fill uint64
	var fill_done bool
	var last uint64
	var length_first uint64
	var length_record uint64
	var length_record_done bool
	var record_base_u uint64
	var record_base int64
	var symbol_low_at int64
	var symbol_high_at int64
	var symbol_high_shifted uint64
	var symbol_u uint64
	var symbol_i int64
	i0 = int64(0)
	i1 = int64(1)
	i2 = int64(2)
	i3 = int64(3)
	i4 = int64(4)
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u4 = uint64(4)
	u8 = uint64(8)
	u33 = uint64(33)
	u255 = uint64(255)
	u65535 = uint64(65535)
	used = []byte{}
	used_count = u0
	symbol = u0
	for {
		done = (symbol >= table_len)
		if done {
			break
		}
		frequency = deflateU16Get(counts, symbol)
		present = (frequency != u0)
		if present {
			frequency_low = (frequency & u255)
			frequency_shifted = (frequency >> u8)
			frequency_high = (frequency_shifted & u255)
			symbol_low = (symbol & u255)
			symbol_shifted = (symbol >> u8)
			symbol_high = (symbol_shifted & u255)
			used = append(used, byte(frequency_low))
			used = append(used, byte(frequency_high))
			used = append(used, byte(symbol_low))
			used = append(used, byte(symbol_high))
			used_count = (used_count + u1)
		}
		symbol = (symbol + u1)
	}
	used_count_i = int64(used_count)
	chosen = []byte{}
	chosen_at = u0
	for {
		chosen_done = (chosen_at >= used_count)
		if chosen_done {
			break
		}
		chosen = append(chosen, byte(u0))
		chosen_at = (chosen_at + u1)
	}
	sorted = []byte{}
	slot = u0
	for {
		sorted_done = (slot >= used_count)
		if sorted_done {
			break
		}
		scan = u0
		best_index = u0
		best_frequency = u65535
		have_best = u0
		for {
			scan_done = (scan >= used_count)
			if scan_done {
				break
			}
			scan_i = int64(scan)
			already = uint64(chosen[scan_i])
			unchosen = (already == u0)
			if unchosen {
				record_u = (scan * u4)
				scan_record = int64(record_u)
				record_high = (scan_record + i1)
				low = uint64(used[scan_record])
				high = uint64(used[record_high])
				high_shifted = (high << u8)
				frequency = (low | high_shifted)
				first_candidate = (have_best == u0)
				lower = (frequency < best_frequency)
				take = (first_candidate || lower)
				if take {
					best_index = scan
					best_frequency = frequency
					have_best = u1
				}
			}
			scan = (scan + u1)
		}
		best_i = int64(best_index)
		chosen[best_i] = byte(u1)
		best_record_u = (best_index * u4)
		best_record = int64(best_record_u)
		copy_ = i0
		for {
			copy_done = (copy_ >= i4)
			if copy_done {
				break
			}
			source = (best_record + copy_)
			byte_ = uint64(used[source])
			sorted = append(sorted, byte(byte_))
			copy_ = (copy_ + i1)
		}
		slot = (slot + u1)
	}
	one = (used_count == u1)
	if one {
		sorted[i0] = byte(u1)
		sorted[i1] = byte(u0)
	}
	many = (used_count > u1)
	if many {
		f0 = deflateRecordKeyGet(sorted, u0)
		f1 = deflateRecordKeyGet(sorted, u1)
		initial_sum = (f0 + f1)
		initial_wrapped = (initial_sum & u65535)
		sorted = deflateRecordKeySet(sorted, u0, initial_wrapped)
		root = u0
		leaf = u2
		next_ = u1
		last_internal = (used_count - u1)
		for {
			merge_done = (next_ >= last_internal)
			if merge_done {
				break
			}
			leaf_end = (leaf >= used_count)
			take_root = leaf_end
			leaf_available = (!leaf_end)
			if leaf_available {
				root_key = deflateRecordKeyGet(sorted, root)
				leaf_key = deflateRecordKeyGet(sorted, leaf)
				root_lower = (root_key < leaf_key)
				take_root = root_lower
			}
			if take_root {
				root_key = deflateRecordKeyGet(sorted, root)
				sorted = deflateRecordKeySet(sorted, next_, root_key)
				sorted = deflateRecordKeySet(sorted, root, next_)
				root = (root + u1)
			}
			take_leaf = (!take_root)
			if take_leaf {
				leaf_key = deflateRecordKeyGet(sorted, leaf)
				sorted = deflateRecordKeySet(sorted, next_, leaf_key)
				leaf = (leaf + u1)
			}
			leaf_end2 = (leaf >= used_count)
			take_root2 = leaf_end2
			leaf_available2 = (!leaf_end2)
			if leaf_available2 {
				root_before_next = (root < next_)
				if root_before_next {
					root_key2 = deflateRecordKeyGet(sorted, root)
					leaf_key2 = deflateRecordKeyGet(sorted, leaf)
					root_lower2 = (root_key2 < leaf_key2)
					take_root2 = root_lower2
				}
			}
			next_key = deflateRecordKeyGet(sorted, next_)
			if take_root2 {
				root_key2 = deflateRecordKeyGet(sorted, root)
				combined = (next_key + root_key2)
				combined_wrapped = (combined & u65535)
				sorted = deflateRecordKeySet(sorted, next_, combined_wrapped)
				sorted = deflateRecordKeySet(sorted, root, next_)
				root = (root + u1)
			}
			take_leaf2 = (!take_root2)
			if take_leaf2 {
				leaf_key2 = deflateRecordKeyGet(sorted, leaf)
				combined = (next_key + leaf_key2)
				combined_wrapped = (combined & u65535)
				sorted = deflateRecordKeySet(sorted, next_, combined_wrapped)
				leaf = (leaf + u1)
			}
			next_ = (next_ + u1)
		}
		n_minus2 = (used_count - u2)
		sorted = deflateRecordKeySet(sorted, n_minus2, u0)
		back = n_minus2
		for {
			back_done = (back == u0)
			if back_done {
				break
			}
			back = (back - u1)
			parent = deflateRecordKeyGet(sorted, back)
			parent_depth = deflateRecordKeyGet(sorted, parent)
			depth = (parent_depth + u1)
			sorted = deflateRecordKeySet(sorted, back, depth)
		}
		slots_available = u1
		used_here = u0
		depth = u0
		root_i = int64(n_minus2)
		next_i = (used_count_i - i1)
		for {
			assign_done = (slots_available == u0)
			if assign_done {
				break
			}
			for {
				root_negative = (root_i < i0)
				if root_negative {
					break
				}
				root_u = uint64(root_i)
				root_depth = deflateRecordKeyGet(sorted, root_u)
				different_depth = (root_depth != depth)
				if different_depth {
					break
				}
				used_here = (used_here + u1)
				root_i = (root_i - i1)
			}
			for {
				enough = (slots_available <= used_here)
				if enough {
					break
				}
				next_u = uint64(next_i)
				sorted = deflateRecordKeySet(sorted, next_u, depth)
				next_i = (next_i - i1)
				slots_available = (slots_available - u1)
			}
			slots_available = (used_here * u2)
			depth = (depth + u1)
			used_here = u0
		}
	}
	num_codes = deflateU16Zeros(u33)
	count_record = u0
	for {
		count_record_done = (count_record >= used_count)
		if count_record_done {
			break
		}
		count_depth = deflateRecordKeyGet(sorted, count_record)
		num_codes = deflateU16Increment(num_codes, count_depth, u1)
		count_record = (count_record + u1)
	}
	over = (limit + u1)
	over_sum = u0
	for {
		over_done = (over >= u33)
		if over_done {
			break
		}
		amount = deflateU16Get(num_codes, over)
		over_sum = (over_sum + amount)
		over = (over + u1)
	}
	limit_count = deflateU16Get(num_codes, limit)
	limit_total = (limit_count + over_sum)
	num_codes = deflateU16Set(num_codes, limit, limit_total)
	total = u0
	width = u1
	for {
		width_done = (width > limit)
		if width_done {
			break
		}
		count = deflateU16Get(num_codes, width)
		shift = (limit - width)
		weighted = (count << shift)
		total = (total + weighted)
		width = (width + u1)
	}
	capacity = (u1 << limit)
	for {
		balanced = (total <= capacity)
		if balanced {
			break
		}
		limit_count = deflateU16Get(num_codes, limit)
		limit_less = (limit_count - u1)
		num_codes = deflateU16Set(num_codes, limit, limit_less)
		seek = (limit - u1)
		for {
			seek_count = deflateU16Get(num_codes, seek)
			found = (seek_count != u0)
			if found {
				seek_less = (seek_count - u1)
				num_codes = deflateU16Set(num_codes, seek, seek_less)
				deeper = (seek + u1)
				deeper_count = deflateU16Get(num_codes, deeper)
				deeper_more = (deeper_count + u2)
				num_codes = deflateU16Set(num_codes, deeper, deeper_more)
				break
			}
			seek = (seek - u1)
		}
		total = (total - u1)
	}
	lengths = []byte{}
	fill = u0
	for {
		fill_done = (fill >= table_len)
		if fill_done {
			break
		}
		lengths = append(lengths, byte(u0))
		fill = (fill + u1)
	}
	last = used_count
	width = u1
	for {
		width_done = (width > limit)
		if width_done {
			break
		}
		amount = deflateU16Get(num_codes, width)
		length_first = (last - amount)
		length_record = length_first
		for {
			length_record_done = (length_record >= last)
			if length_record_done {
				break
			}
			record_base_u = (length_record * u4)
			record_base = int64(record_base_u)
			symbol_low_at = (record_base + i2)
			symbol_high_at = (record_base + i3)
			symbol_low = uint64(sorted[symbol_low_at])
			symbol_high = uint64(sorted[symbol_high_at])
			symbol_high_shifted = (symbol_high << u8)
			symbol_u = (symbol_low | symbol_high_shifted)
			symbol_i = int64(symbol_u)
			lengths[symbol_i] = byte(width)
			length_record = (length_record + u1)
		}
		last = length_first
		width = (width + u1)
	}
	return lengths
}

func deflateCanonicalCodes(lengths []byte, max_bits uint64) []byte {
	var i0 int64
	var i1 int64
	var u0 uint64
	var u1 uint64
	var counts []byte
	var counts_head []byte
	var n int64
	var at int64
	var count_done bool
	var width uint64
	var coded bool
	var next_count uint64
	var next_ []byte
	var code uint64
	var next_done bool
	var previous uint64
	var previous_count uint64
	var sum uint64
	var n_u uint64
	var codes []byte
	var symbol int64
	var codes_done bool
	var canonical uint64
	var wire uint64
	var symbol_u uint64
	var canonical_next uint64
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u1 = uint64(1)
	counts = deflateU16Zeros(max_bits)
	counts_head = []byte{}
	counts_head = append(counts_head, byte(u0))
	counts_head = append(counts_head, byte(u0))
	counts = appendBytes(counts_head, counts)
	n = int64(len(lengths))
	at = i0
	for {
		count_done = (at >= n)
		if count_done {
			break
		}
		width = uint64(lengths[at])
		coded = (width != u0)
		if coded {
			counts = deflateU16Increment(counts, width, u1)
		}
		at = (at + i1)
	}
	next_count = (max_bits + u1)
	next_ = deflateU16Zeros(next_count)
	code = u0
	width = uint64(2)
	for {
		next_done = (width > max_bits)
		if next_done {
			break
		}
		previous = (width - u1)
		previous_count = deflateU16Get(counts, previous)
		sum = (code + previous_count)
		code = (sum << u1)
		next_ = deflateU16Set(next_, width, code)
		width = (width + u1)
	}
	n_u = uint64(n)
	codes = deflateU16Zeros(n_u)
	symbol = i0
	for {
		codes_done = (symbol >= n)
		if codes_done {
			break
		}
		width = uint64(lengths[symbol])
		coded = (width != u0)
		if coded {
			canonical = deflateU16Get(next_, width)
			wire = deflateReverseCode(canonical, width)
			symbol_u = uint64(symbol)
			codes = deflateU16Set(codes, symbol_u, wire)
			canonical_next = (canonical + u1)
			next_ = deflateU16Set(next_, width, canonical_next)
		}
		symbol = (symbol + i1)
	}
	return codes
}

func deflatePackCodeLengths(lengths []byte) []byte {
	var i0 int64
	var i1 int64
	var u0 uint64
	var u1 uint64
	var u3 uint64
	var u6 uint64
	var u10 uint64
	var u11 uint64
	var u16 uint64
	var u17 uint64
	var u18 uint64
	var u19 uint64
	var u138 uint64
	var counts []byte
	var packed_ []byte
	var n int64
	var at int64
	var done bool
	var width uint64
	var zero bool
	var run uint64
	var scan int64
	var scan_end bool
	var run_full bool
	var stop bool
	var next_width uint64
	var different bool
	var short_ bool
	var left uint64
	var emitted bool
	var not_short bool
	var medium bool
	var extra uint64
	var long_ bool
	var run_i int64
	var nonzero bool
	var repeat uint64
	var repeat_done bool
	var chunk_full bool
	var repeat_short bool
	var repeat_long bool
	var out []byte
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u1 = uint64(1)
	u3 = uint64(3)
	u6 = uint64(6)
	u10 = uint64(10)
	u11 = uint64(11)
	u16 = uint64(16)
	u17 = uint64(17)
	u18 = uint64(18)
	u19 = uint64(19)
	u138 = uint64(138)
	counts = deflateU16Zeros(u19)
	packed_ = []byte{}
	n = int64(len(lengths))
	at = i0
	for {
		done = (at >= n)
		if done {
			break
		}
		width = uint64(lengths[at])
		zero = (width == u0)
		if zero {
			run = u1
			scan = (at + i1)
			for {
				scan_end = (scan >= n)
				run_full = (run >= u138)
				stop = (scan_end || run_full)
				if stop {
					break
				}
				next_width = uint64(lengths[scan])
				different = (next_width != u0)
				if different {
					break
				}
				run = (run + u1)
				scan = (scan + i1)
			}
			short_ = (run < u3)
			if short_ {
				left = run
				for {
					emitted = (left == u0)
					if emitted {
						break
					}
					packed_ = append(packed_, byte(u0))
					counts = deflateU16Increment(counts, u0, u1)
					left = (left - u1)
				}
			}
			not_short = (!short_)
			if not_short {
				medium = (run <= u10)
				if medium {
					packed_ = append(packed_, byte(u17))
					extra = (run - u3)
					packed_ = append(packed_, byte(extra))
					counts = deflateU16Increment(counts, u17, u1)
				}
				long_ = (!medium)
				if long_ {
					packed_ = append(packed_, byte(u18))
					extra = (run - u11)
					packed_ = append(packed_, byte(extra))
					counts = deflateU16Increment(counts, u18, u1)
				}
			}
			run_i = int64(run)
			at = (at + run_i)
		}
		nonzero = (!zero)
		if nonzero {
			packed_ = append(packed_, byte(width))
			counts = deflateU16Increment(counts, width, u1)
			at = (at + i1)
			repeat = u0
			for {
				repeat_done = (at >= n)
				if repeat_done {
					break
				}
				next_width = uint64(lengths[at])
				different = (next_width != width)
				if different {
					break
				}
				repeat = (repeat + u1)
				at = (at + i1)
				chunk_full = (repeat == u6)
				if chunk_full {
					packed_ = append(packed_, byte(u16))
					extra = (repeat - u3)
					packed_ = append(packed_, byte(extra))
					counts = deflateU16Increment(counts, u16, u1)
					repeat = u0
				}
			}
			repeat_short = (repeat < u3)
			if repeat_short {
				left = repeat
				for {
					emitted = (left == u0)
					if emitted {
						break
					}
					packed_ = append(packed_, byte(width))
					counts = deflateU16Increment(counts, width, u1)
					left = (left - u1)
				}
			}
			repeat_long = (repeat >= u3)
			if repeat_long {
				packed_ = append(packed_, byte(u16))
				extra = (repeat - u3)
				packed_ = append(packed_, byte(extra))
				counts = deflateU16Increment(counts, u16, u1)
			}
		}
	}
	out = appendBytes(counts, packed_)
	return out
}

func deflateHuffmanSymbol(state []byte, codes []byte, lengths []byte, symbol uint64) []byte {
	var symbol_i int64
	var width uint64
	var code uint64
	var out []byte
	symbol_i = int64(symbol)
	width = uint64(lengths[symbol_i])
	code = deflateU16Get(codes, symbol)
	out = deflateWriteBits(state, code, width)
	return out
}

func deflateMatchSymbols(length uint64, distance uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var u30 uint64
	var u32 uint64
	var u257 uint64
	var u258 uint64
	var u285 uint64
	var length_symbol uint64
	var last_length bool
	var base uint64
	var extra_count uint64
	var span uint64
	var upper uint64
	var fits bool
	var maximum_length bool
	var distance_symbol uint64
	var last_distance bool
	var length_high uint64
	var packed_ uint64
	u0 = uint64(0)
	u1 = uint64(1)
	u30 = uint64(30)
	u32 = uint64(32)
	u257 = uint64(257)
	u258 = uint64(258)
	u285 = uint64(285)
	length_symbol = u257
	for {
		last_length = (length_symbol >= u285)
		if last_length {
			break
		}
		base = lengthBase(length_symbol)
		extra_count = lengthExtra(length_symbol)
		span = (u1 << extra_count)
		upper = (base + span)
		fits = (length < upper)
		if fits {
			break
		}
		length_symbol = (length_symbol + u1)
	}
	maximum_length = (length == u258)
	if maximum_length {
		length_symbol = u285
	}
	distance_symbol = u0
	for {
		last_distance = (distance_symbol >= u30)
		if last_distance {
			break
		}
		base = distanceBase(distance_symbol)
		extra_count = distanceExtra(distance_symbol)
		span = (u1 << extra_count)
		upper = (base + span)
		fits = (distance < upper)
		if fits {
			break
		}
		distance_symbol = (distance_symbol + u1)
	}
	length_high = (length_symbol << u32)
	packed_ = (length_high | distance_symbol)
	return packed_
}

func deflateDynamic(data []byte) []byte {
	var i0 int64
	var i1 int64
	var i2 int64
	var i3 int64
	var i4 int64
	var i5 int64
	var i38 int64
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var u4 uint64
	var u5 uint64
	var u7 uint64
	var u8 uint64
	var u15 uint64
	var u16 uint64
	var u19 uint64
	var u30 uint64
	var u32 uint64
	var u256 uint64
	var u257 uint64
	var u286 uint64
	var u288 uint64
	var u4294967295 uint64
	var tokens []byte
	var lit_counts []byte
	var dist_counts []byte
	var tokens_n int64
	var token_at int64
	var count_done bool
	var tag uint64
	var match_ bool
	var value_low_at int64
	var value_high_at int64
	var distance_low_at int64
	var distance_high_at int64
	var value_low uint64
	var value_high uint64
	var value_high_shifted uint64
	var length uint64
	var distance_low uint64
	var distance_high uint64
	var distance_high_shifted uint64
	var distance uint64
	var symbols uint64
	var length_symbol uint64
	var distance_symbol uint64
	var literal bool
	var literal_at int64
	var literal_symbol uint64
	var lit_lengths []byte
	var dist_lengths []byte
	var lit_codes []byte
	var dist_codes []byte
	var num_lit uint64
	var minimum_lit bool
	var last_lit uint64
	var last_lit_i int64
	var last_width uint64
	var used_lit bool
	var num_dist uint64
	var minimum_dist bool
	var last_dist uint64
	var last_dist_i int64
	var used_dist bool
	var num_lit_i int64
	var num_dist_i int64
	var packed_lit_lengths []byte
	var packed_dist_lengths []byte
	var sizes []byte
	var packed_bundle []byte
	var bundle_n int64
	var cl_counts []byte
	var packed_ []byte
	var cl_lengths []byte
	var cl_codes []byte
	var order []byte
	var num_bit_lengths uint64
	var minimum_cl bool
	var order_at uint64
	var order_at_i int64
	var cl_symbol uint64
	var cl_symbol_i int64
	var cl_width uint64
	var used_cl bool
	var out []byte
	var state []byte
	var hlit uint64
	var hdist uint64
	var hclen uint64
	var order_done bool
	var packed_n int64
	var packed_at int64
	var packed_done bool
	var has_extra bool
	var extra uint64
	var extra_index uint64
	var extra_width uint64
	var is17 bool
	var is18 bool
	var emit_done bool
	var length_base_value uint64
	var length_extra_width uint64
	var length_extra_value uint64
	var distance_base_value uint64
	var distance_extra_width uint64
	var distance_extra_value uint64
	var complete []byte
	var complete_n int64
	var drain_at int64
	var drain_done bool
	var drain_byte uint64
	var state_n int64
	var tail_at int64
	var tail_done bool
	var tail_byte uint64
	i0 = int64(0)
	i1 = int64(1)
	i2 = int64(2)
	i3 = int64(3)
	i4 = int64(4)
	i5 = int64(5)
	i38 = int64(38)
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	u4 = uint64(4)
	u5 = uint64(5)
	u7 = uint64(7)
	u8 = uint64(8)
	u15 = uint64(15)
	u16 = uint64(16)
	u19 = uint64(19)
	u30 = uint64(30)
	u32 = uint64(32)
	u256 = uint64(256)
	u257 = uint64(257)
	u286 = uint64(286)
	u288 = uint64(288)
	u4294967295 = uint64(4294967295)
	tokens = deflateTokens(data)
	lit_counts = deflateU16Zeros(u288)
	dist_counts = deflateU16Zeros(u32)
	tokens_n = int64(len(tokens))
	token_at = i0
	for {
		count_done = (token_at >= tokens_n)
		if count_done {
			break
		}
		tag = uint64(tokens[token_at])
		match_ = (tag == u1)
		if match_ {
			value_low_at = (token_at + i1)
			value_high_at = (token_at + i2)
			distance_low_at = (token_at + i3)
			distance_high_at = (token_at + i4)
			value_low = uint64(tokens[value_low_at])
			value_high = uint64(tokens[value_high_at])
			value_high_shifted = (value_high << u8)
			length = (value_low | value_high_shifted)
			distance_low = uint64(tokens[distance_low_at])
			distance_high = uint64(tokens[distance_high_at])
			distance_high_shifted = (distance_high << u8)
			distance = (distance_low | distance_high_shifted)
			symbols = deflateMatchSymbols(length, distance)
			length_symbol = (symbols >> u32)
			distance_symbol = (symbols & u4294967295)
			lit_counts = deflateU16Increment(lit_counts, length_symbol, u1)
			dist_counts = deflateU16Increment(dist_counts, distance_symbol, u1)
		}
		literal = (!match_)
		if literal {
			literal_at = (token_at + i1)
			literal_symbol = uint64(tokens[literal_at])
			lit_counts = deflateU16Increment(lit_counts, literal_symbol, u1)
		}
		token_at = (token_at + i5)
	}
	lit_counts = deflateU16Set(lit_counts, u256, u1)
	lit_lengths = deflateOptimizedLengths(lit_counts, u288, u15)
	dist_lengths = deflateOptimizedLengths(dist_counts, u32, u15)
	lit_codes = deflateCanonicalCodes(lit_lengths, u15)
	dist_codes = deflateCanonicalCodes(dist_lengths, u15)
	num_lit = u286
	for {
		minimum_lit = (num_lit <= u257)
		if minimum_lit {
			break
		}
		last_lit = (num_lit - u1)
		last_lit_i = int64(last_lit)
		last_width = uint64(lit_lengths[last_lit_i])
		used_lit = (last_width != u0)
		if used_lit {
			break
		}
		num_lit = last_lit
	}
	num_dist = u30
	for {
		minimum_dist = (num_dist <= u1)
		if minimum_dist {
			break
		}
		last_dist = (num_dist - u1)
		last_dist_i = int64(last_dist)
		last_width = uint64(dist_lengths[last_dist_i])
		used_dist = (last_width != u0)
		if used_dist {
			break
		}
		num_dist = last_dist
	}
	num_lit_i = int64(num_lit)
	num_dist_i = int64(num_dist)
	packed_lit_lengths = append([]byte(nil), lit_lengths[i0:num_lit_i]...)
	packed_dist_lengths = append([]byte(nil), dist_lengths[i0:num_dist_i]...)
	sizes = appendBytes(packed_lit_lengths, packed_dist_lengths)
	packed_bundle = deflatePackCodeLengths(sizes)
	bundle_n = int64(len(packed_bundle))
	cl_counts = append([]byte(nil), packed_bundle[i0:i38]...)
	packed_ = append([]byte(nil), packed_bundle[i38:bundle_n]...)
	cl_lengths = deflateOptimizedLengths(cl_counts, u19, u7)
	cl_codes = deflateCanonicalCodes(cl_lengths, u7)
	order = codeLengthOrder()
	num_bit_lengths = u19
	for {
		minimum_cl = (num_bit_lengths <= u4)
		if minimum_cl {
			break
		}
		order_at = (num_bit_lengths - u1)
		order_at_i = int64(order_at)
		cl_symbol = uint64(order[order_at_i])
		cl_symbol_i = int64(cl_symbol)
		cl_width = uint64(cl_lengths[cl_symbol_i])
		used_cl = (cl_width != u0)
		if used_cl {
			break
		}
		num_bit_lengths = order_at
	}
	out = []byte{}
	state = []byte{}
	state = append(state, byte(u0))
	state = deflateWriteBits(state, u5, u3)
	hlit = (num_lit - u257)
	hdist = (num_dist - u1)
	hclen = (num_bit_lengths - u4)
	state = deflateWriteBits(state, hlit, u5)
	state = deflateWriteBits(state, hdist, u5)
	state = deflateWriteBits(state, hclen, u4)
	order_at = u0
	for {
		order_done = (order_at >= num_bit_lengths)
		if order_done {
			break
		}
		order_at_i = int64(order_at)
		cl_symbol = uint64(order[order_at_i])
		cl_symbol_i = int64(cl_symbol)
		cl_width = uint64(cl_lengths[cl_symbol_i])
		state = deflateWriteBits(state, cl_width, u3)
		order_at = (order_at + u1)
	}
	packed_n = int64(len(packed_))
	packed_at = i0
	for {
		packed_done = (packed_at >= packed_n)
		if packed_done {
			break
		}
		cl_symbol = uint64(packed_[packed_at])
		state = deflateHuffmanSymbol(state, cl_codes, cl_lengths, cl_symbol)
		packed_at = (packed_at + i1)
		has_extra = (cl_symbol >= u16)
		if has_extra {
			extra = uint64(packed_[packed_at])
			extra_index = (cl_symbol - u16)
			extra_width = u2
			is17 = (extra_index == u1)
			if is17 {
				extra_width = u3
			}
			is18 = (extra_index == u2)
			if is18 {
				extra_width = u7
			}
			state = deflateWriteBits(state, extra, extra_width)
			packed_at = (packed_at + i1)
		}
	}
	token_at = i0
	for {
		emit_done = (token_at >= tokens_n)
		if emit_done {
			break
		}
		tag = uint64(tokens[token_at])
		match_ = (tag == u1)
		if match_ {
			value_low_at = (token_at + i1)
			value_high_at = (token_at + i2)
			distance_low_at = (token_at + i3)
			distance_high_at = (token_at + i4)
			value_low = uint64(tokens[value_low_at])
			value_high = uint64(tokens[value_high_at])
			value_high_shifted = (value_high << u8)
			length = (value_low | value_high_shifted)
			distance_low = uint64(tokens[distance_low_at])
			distance_high = uint64(tokens[distance_high_at])
			distance_high_shifted = (distance_high << u8)
			distance = (distance_low | distance_high_shifted)
			symbols = deflateMatchSymbols(length, distance)
			length_symbol = (symbols >> u32)
			distance_symbol = (symbols & u4294967295)
			state = deflateHuffmanSymbol(state, lit_codes, lit_lengths, length_symbol)
			length_base_value = lengthBase(length_symbol)
			length_extra_width = lengthExtra(length_symbol)
			length_extra_value = (length - length_base_value)
			state = deflateWriteBits(state, length_extra_value, length_extra_width)
			state = deflateHuffmanSymbol(state, dist_codes, dist_lengths, distance_symbol)
			distance_base_value = distanceBase(distance_symbol)
			distance_extra_width = distanceExtra(distance_symbol)
			distance_extra_value = (distance - distance_base_value)
			state = deflateWriteBits(state, distance_extra_value, distance_extra_width)
		}
		literal = (!match_)
		if literal {
			literal_at = (token_at + i1)
			literal_symbol = uint64(tokens[literal_at])
			state = deflateHuffmanSymbol(state, lit_codes, lit_lengths, literal_symbol)
		}
		complete = deflateDrainComplete(state)
		complete_n = int64(len(complete))
		drain_at = i0
		for {
			drain_done = (drain_at >= complete_n)
			if drain_done {
				break
			}
			drain_byte = uint64(complete[drain_at])
			out = append(out, byte(drain_byte))
			drain_at = (drain_at + i1)
		}
		state = deflateDrainCarry(state)
		token_at = (token_at + i5)
	}
	state = deflateHuffmanSymbol(state, lit_codes, lit_lengths, u256)
	state_n = int64(len(state))
	tail_at = i1
	for {
		tail_done = (tail_at >= state_n)
		if tail_done {
			break
		}
		tail_byte = uint64(state[tail_at])
		out = append(out, byte(tail_byte))
		tail_at = (tail_at + i1)
	}
	return out
}

func deflateStored(data []byte) []byte {
	var u1 uint64
	var u8 uint64
	var u255 uint64
	var u65535 uint64
	var head []byte
	var n_i int64
	var n uint64
	var n_low uint64
	var n_shifted uint64
	var n_high uint64
	var complement uint64
	var complement_low uint64
	var complement_shifted uint64
	var complement_high uint64
	var out []byte
	u1 = uint64(1)
	u8 = uint64(8)
	u255 = uint64(255)
	u65535 = uint64(65535)
	head = []byte{}
	head = append(head, byte(u1))
	n_i = int64(len(data))
	n = uint64(n_i)
	n_low = (n & u255)
	n_shifted = (n >> u8)
	n_high = (n_shifted & u255)
	complement = (u65535 - n)
	complement_low = (complement & u255)
	complement_shifted = (complement >> u8)
	complement_high = (complement_shifted & u255)
	head = append(head, byte(n_low))
	head = append(head, byte(n_high))
	head = append(head, byte(complement_low))
	head = append(head, byte(complement_high))
	out = appendBytes(head, data)
	return out
}

func deflateMinizLevel6(data []byte) []byte {
	var i0 int64
	var i32 int64
	var i48 int64
	var i65535 int64
	var n int64
	var body []byte
	var fixed bool
	var dynamic bool
	var body_n int64
	var large bool
	var expanded bool
	var fits_stored bool
	var fallback_candidate bool
	var fallback bool
	i0 = int64(0)
	i32 = int64(32)
	i48 = int64(48)
	i65535 = int64(65535)
	n = int64(len(data))
	body = append([]byte(nil), data[i0:i0]...)
	fixed = (n < i48)
	if fixed {
		body = deflateFixed(data)
	}
	dynamic = (n >= i48)
	if dynamic {
		body = deflateDynamic(data)
	}
	body_n = int64(len(body))
	large = (n > i32)
	expanded = (body_n >= n)
	fits_stored = (n <= i65535)
	fallback_candidate = (large && expanded)
	fallback = (fallback_candidate && fits_stored)
	if fallback {
		body = deflateStored(data)
	}
	return body
}

func zlibCompress(data []byte) []byte {
	var u8 uint64
	var u16 uint64
	var u24 uint64
	var u120 uint64
	var u156 uint64
	var u255 uint64
	var out []byte
	var body []byte
	var prefixed []byte
	var checksum uint64
	var b3_shifted uint64
	var b3 uint64
	var b2_shifted uint64
	var b2 uint64
	var b1_shifted uint64
	var b1 uint64
	var b0 uint64
	var trailer []byte
	var complete []byte
	u8 = uint64(8)
	u16 = uint64(16)
	u24 = uint64(24)
	u120 = uint64(120)
	u156 = uint64(156)
	u255 = uint64(255)
	out = []byte{}
	out = append(out, byte(u120))
	out = append(out, byte(u156))
	body = deflateMinizLevel6(data)
	prefixed = appendBytes(out, body)
	checksum = adler32(data)
	b3_shifted = (checksum >> u24)
	b3 = (b3_shifted & u255)
	b2_shifted = (checksum >> u16)
	b2 = (b2_shifted & u255)
	b1_shifted = (checksum >> u8)
	b1 = (b1_shifted & u255)
	b0 = (checksum & u255)
	trailer = []byte{}
	trailer = append(trailer, byte(b3))
	trailer = append(trailer, byte(b2))
	trailer = append(trailer, byte(b1))
	trailer = append(trailer, byte(b0))
	complete = appendBytes(prefixed, trailer)
	return complete
}

func appendBytes(left []byte, right []byte) []byte {
	var out []byte
	var i0 int64
	var i1 int64
	var left_n int64
	var right_n int64
	var pos int64
	var done bool
	var byte_ uint64
	out = []byte{}
	i0 = int64(0)
	i1 = int64(1)
	left_n = int64(len(left))
	right_n = int64(len(right))
	pos = i0
	for {
		done = (pos >= left_n)
		if done {
			break
		}
		byte_ = uint64(left[pos])
		out = append(out, byte(byte_))
		pos = (pos + i1)
	}
	pos = i0
	for {
		done = (pos >= right_n)
		if done {
			break
		}
		byte_ = uint64(right[pos])
		out = append(out, byte(byte_))
		pos = (pos + i1)
	}
	return out
}

func u32ToLe(value uint64) []byte {
	var out []byte
	var u8 uint64
	var u255 uint64
	var u4 uint64
	var pos uint64
	var remaining uint64
	var done bool
	var byte_ uint64
	var one uint64
	out = []byte{}
	u8 = uint64(8)
	u255 = uint64(255)
	u4 = uint64(4)
	pos = uint64(0)
	remaining = value
	for {
		done = (pos >= u4)
		if done {
			break
		}
		byte_ = (remaining & u255)
		out = append(out, byte(byte_))
		remaining = (remaining >> u8)
		one = uint64(1)
		pos = (pos + one)
	}
	return out
}

func u64ToLe(value uint64) []byte {
	var out []byte
	var u8 uint64
	var u255 uint64
	var count uint64
	var pos uint64
	var remaining uint64
	var done bool
	var byte_ uint64
	var one uint64
	out = []byte{}
	u8 = uint64(8)
	u255 = uint64(255)
	count = uint64(8)
	pos = uint64(0)
	remaining = value
	for {
		done = (pos >= count)
		if done {
			break
		}
		byte_ = (remaining & u255)
		out = append(out, byte(byte_))
		remaining = (remaining >> u8)
		one = uint64(1)
		pos = (pos + one)
	}
	return out
}

func readU32Le(data []byte, start int64) uint64 {
	var u0 uint64
	var u8 uint64
	var i0 int64
	var i1 int64
	var i4 int64
	var result uint64
	var shift uint64
	var pos int64
	var read int64
	var done bool
	var byte_ uint64
	var part uint64
	u0 = uint64(0)
	u8 = uint64(8)
	i0 = int64(0)
	i1 = int64(1)
	i4 = int64(4)
	result = u0
	shift = u0
	pos = start
	read = i0
	for {
		done = (read >= i4)
		if done {
			break
		}
		byte_ = uint64(data[pos])
		part = (byte_ << shift)
		result = (result | part)
		shift = (shift + u8)
		pos = (pos + i1)
		read = (read + i1)
	}
	return result
}

func readU64Le(data []byte, start int64) uint64 {
	var u0 uint64
	var u8 uint64
	var i0 int64
	var i1 int64
	var i8 int64
	var result uint64
	var shift uint64
	var pos int64
	var read int64
	var done bool
	var byte_ uint64
	var part uint64
	u0 = uint64(0)
	u8 = uint64(8)
	i0 = int64(0)
	i1 = int64(1)
	i8 = int64(8)
	result = u0
	shift = u0
	pos = start
	read = i0
	for {
		done = (read >= i8)
		if done {
			break
		}
		byte_ = uint64(data[pos])
		part = (byte_ << shift)
		result = (result | part)
		shift = (shift + u8)
		pos = (pos + i1)
		read = (read + i1)
	}
	return result
}

func boundedAdvance(data []byte, start int64, amount uint64) int64 {
	var i0 int64
	var negative bool
	var amount_i64 int64
	var end_ int64
	var n int64
	var inside bool
	i0 = int64(0)
	negative = (start < i0)
	if negative {
		return i0
	}
	amount_i64 = int64(amount)
	end_ = (start + amount_i64)
	n = int64(len(data))
	inside = (end_ <= n)
	if inside {
		return end_
	}
	return i0
}

func choiceListHeader(count uint64) []byte {
	var header []byte
	header = u32ToLe(count)
	return header
}

func serializeIntegerChoice(value_raw []byte) []byte {
	var out []byte
	var tag uint64
	var subtag uint64
	var value []byte
	var length_i64 int64
	var length uint64
	var length_bytes []byte
	var with_length []byte
	var encoded []byte
	out = []byte{}
	tag = uint64(0)
	subtag = uint64(10)
	out = append(out, byte(tag))
	out = append(out, byte(subtag))
	value = canonicalizeSignedLe(value_raw)
	length_i64 = int64(len(value))
	length = uint64(length_i64)
	length_bytes = u32ToLe(length)
	with_length = appendBytes(out, length_bytes)
	encoded = appendBytes(with_length, value)
	return encoded
}

func serializeBooleanChoice(value bool) []byte {
	var out []byte
	var tag uint64
	var byte_ uint64
	out = []byte{}
	tag = uint64(1)
	byte_ = uint64(0)
	if value {
		byte_ = uint64(1)
	}
	out = append(out, byte(tag))
	out = append(out, byte(byte_))
	return out
}

func serializeFloatChoice(value float64) []byte {
	var out []byte
	var tag uint64
	var bits uint64
	var raw []byte
	var encoded []byte
	out = []byte{}
	tag = uint64(2)
	out = append(out, byte(tag))
	bits = math.Float64bits(value)
	raw = u64ToLe(bits)
	encoded = appendBytes(out, raw)
	return encoded
}

func serializeBytesChoice(value []byte) []byte {
	var out []byte
	var tag uint64
	var length_i64 int64
	var length uint64
	var length_bytes []byte
	var with_length []byte
	var encoded []byte
	out = []byte{}
	tag = uint64(3)
	out = append(out, byte(tag))
	length_i64 = int64(len(value))
	length = uint64(length_i64)
	length_bytes = u32ToLe(length)
	with_length = appendBytes(out, length_bytes)
	encoded = appendBytes(with_length, value)
	return encoded
}

func serializeStringChoicePacked(count uint64, codepoints_le []byte) []byte {
	var out []byte
	var tag uint64
	var length_bytes []byte
	var with_length []byte
	var encoded []byte
	out = []byte{}
	tag = uint64(4)
	out = append(out, byte(tag))
	length_bytes = u32ToLe(count)
	with_length = appendBytes(out, length_bytes)
	encoded = appendBytes(with_length, codepoints_le)
	return encoded
}

func serializeCloneHeader(child_count uint64) []byte {
	var out []byte
	var tag uint64
	var header []byte
	var encoded []byte
	out = []byte{}
	tag = uint64(5)
	out = append(out, byte(tag))
	header = choiceListHeader(child_count)
	encoded = appendBytes(out, header)
	return encoded
}

func serializeKeyedChoice(key uint64, inner []byte) []byte {
	var out []byte
	var tag uint64
	var key_bytes []byte
	var with_key []byte
	var encoded []byte
	out = []byte{}
	tag = uint64(6)
	out = append(out, byte(tag))
	key_bytes = u64ToLe(key)
	with_key = appendBytes(out, key_bytes)
	encoded = appendBytes(with_key, inner)
	return encoded
}

func keyedEntryKey(data []byte, start int64) (uint64, bool) {
	var i0 int64
	var i1 int64
	var u6 uint64
	var u8 uint64
	var none optionalU64
	var n int64
	var start_negative bool
	var missing_tag bool
	var tag uint64
	var keyed bool
	var unkeyed bool
	var key_start int64
	var key_end int64
	var truncated bool
	var key uint64
	var found optionalU64
	i0 = int64(0)
	i1 = int64(1)
	u6 = uint64(6)
	u8 = uint64(8)
	none = optionalU64{}
	n = int64(len(data))
	start_negative = (start < i0)
	if start_negative {
		return none.value, none.ok
	}
	missing_tag = (start >= n)
	if missing_tag {
		return none.value, none.ok
	}
	tag = uint64(data[start])
	keyed = (tag == u6)
	unkeyed = (!keyed)
	if unkeyed {
		return none.value, none.ok
	}
	key_start = (start + i1)
	key_end = boundedAdvance(data, key_start, u8)
	truncated = (key_end == i0)
	if truncated {
		return none.value, none.ok
	}
	key = readU64Le(data, key_start)
	found = optionalU64{value: key, ok: true}
	return found.value, found.ok
}

func keyedEntryInner(data []byte, start int64) int64 {
	var i0 int64
	var i1 int64
	var u8 uint64
	var key optionalU64
	var present bool
	var absent bool
	var key_start int64
	var inner int64
	i0 = int64(0)
	i1 = int64(1)
	u8 = uint64(8)
	key.value, key.ok = keyedEntryKey(data, start)
	present = key.ok
	absent = (!present)
	if absent {
		return i0
	}
	key_start = (start + i1)
	inner = boundedAdvance(data, key_start, u8)
	return inner
}

func choiceEntryEnd(data []byte, start int64, depth uint64) int64 {
	var i0 int64
	var i1 int64
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u4 uint64
	var u8 uint64
	var u16 uint64
	var u100 uint64
	var too_deep bool
	var n int64
	var start_negative bool
	var missing_tag bool
	var tag uint64
	var value_start int64
	var integer bool
	var sub_end int64
	var sub_missing bool
	var sub uint64
	var sub0 bool
	var end_ int64
	var sub1 bool
	var sub2 bool
	var u3 uint64
	var sub3 bool
	var sub4 bool
	var u5 uint64
	var sub5 bool
	var u6 uint64
	var sub6 bool
	var u7 uint64
	var sub7 bool
	var sub8 bool
	var u9 uint64
	var sub9 bool
	var u10 uint64
	var sub10 bool
	var header_end int64
	var header_missing bool
	var length uint64
	var boolean_ bool
	var float_ bool
	var bytes bool
	var string_ bool
	var byte_length uint64
	var keyed bool
	var inner int64
	var truncated bool
	var nested_key optionalU64
	var double_keyed bool
	var clone bool
	var count uint64
	var child_pos int64
	var child_index uint64
	var child_depth uint64
	var done bool
	var child_end int64
	var child_invalid bool
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u4 = uint64(4)
	u8 = uint64(8)
	u16 = uint64(16)
	u100 = uint64(100)
	too_deep = (depth > u100)
	if too_deep {
		return i0
	}
	n = int64(len(data))
	start_negative = (start < i0)
	if start_negative {
		return i0
	}
	missing_tag = (start >= n)
	if missing_tag {
		return i0
	}
	tag = uint64(data[start])
	value_start = (start + i1)
	integer = (tag == u0)
	if integer {
		sub_end = boundedAdvance(data, value_start, u1)
		sub_missing = (sub_end == i0)
		if sub_missing {
			return i0
		}
		sub = uint64(data[value_start])
		sub0 = (sub == u0)
		if sub0 {
			end_ = boundedAdvance(data, sub_end, u1)
			return end_
		}
		sub1 = (sub == u1)
		if sub1 {
			end_ = boundedAdvance(data, sub_end, u2)
			return end_
		}
		sub2 = (sub == u2)
		if sub2 {
			end_ = boundedAdvance(data, sub_end, u4)
			return end_
		}
		u3 = uint64(3)
		sub3 = (sub == u3)
		if sub3 {
			end_ = boundedAdvance(data, sub_end, u8)
			return end_
		}
		sub4 = (sub == u4)
		if sub4 {
			end_ = boundedAdvance(data, sub_end, u16)
			return end_
		}
		u5 = uint64(5)
		sub5 = (sub == u5)
		if sub5 {
			end_ = boundedAdvance(data, sub_end, u1)
			return end_
		}
		u6 = uint64(6)
		sub6 = (sub == u6)
		if sub6 {
			end_ = boundedAdvance(data, sub_end, u2)
			return end_
		}
		u7 = uint64(7)
		sub7 = (sub == u7)
		if sub7 {
			end_ = boundedAdvance(data, sub_end, u4)
			return end_
		}
		sub8 = (sub == u8)
		if sub8 {
			end_ = boundedAdvance(data, sub_end, u8)
			return end_
		}
		u9 = uint64(9)
		sub9 = (sub == u9)
		if sub9 {
			end_ = boundedAdvance(data, sub_end, u16)
			return end_
		}
		u10 = uint64(10)
		sub10 = (sub == u10)
		if sub10 {
			header_end = boundedAdvance(data, sub_end, u4)
			header_missing = (header_end == i0)
			if header_missing {
				return i0
			}
			length = readU32Le(data, sub_end)
			end_ = boundedAdvance(data, header_end, length)
			return end_
		}
		return i0
	}
	boolean_ = (tag == u1)
	if boolean_ {
		end_ = boundedAdvance(data, value_start, u1)
		return end_
	}
	float_ = (tag == u2)
	if float_ {
		end_ = boundedAdvance(data, value_start, u8)
		return end_
	}
	u3 = uint64(3)
	bytes = (tag == u3)
	if bytes {
		header_end = boundedAdvance(data, value_start, u4)
		header_missing = (header_end == i0)
		if header_missing {
			return i0
		}
		length = readU32Le(data, value_start)
		end_ = boundedAdvance(data, header_end, length)
		return end_
	}
	string_ = (tag == u4)
	if string_ {
		header_end = boundedAdvance(data, value_start, u4)
		header_missing = (header_end == i0)
		if header_missing {
			return i0
		}
		length = readU32Le(data, value_start)
		byte_length = (length * u4)
		end_ = boundedAdvance(data, header_end, byte_length)
		return end_
	}
	u6 = uint64(6)
	keyed = (tag == u6)
	if keyed {
		inner = boundedAdvance(data, value_start, u8)
		truncated = (inner == i0)
		if truncated {
			return i0
		}
		nested_key.value, nested_key.ok = keyedEntryKey(data, inner)
		double_keyed = nested_key.ok
		if double_keyed {
			return i0
		}
		end_ = choiceEntryEnd(data, inner, depth)
		return end_
	}
	u5 = uint64(5)
	clone = (tag == u5)
	if clone {
		header_end = boundedAdvance(data, value_start, u4)
		header_missing = (header_end == i0)
		if header_missing {
			return i0
		}
		count = readU32Le(data, value_start)
		child_pos = header_end
		child_index = u0
		child_depth = (depth + u1)
		for {
			done = (child_index >= count)
			if done {
				break
			}
			child_end = choiceEntryEnd(data, child_pos, child_depth)
			child_invalid = (child_end == i0)
			if child_invalid {
				return i0
			}
			child_pos = child_end
			child_index = (child_index + u1)
		}
		return child_pos
	}
	return i0
}

func choiceStreamValidate(data []byte) bool {
	var i0 int64
	var i4 int64
	var u0 uint64
	var u1 uint64
	var u4 uint64
	var header_end int64
	var header_missing bool
	var invalid bool
	var count uint64
	var pos int64
	var index uint64
	var done bool
	var end_ int64
	var invalid_entry bool
	var valid bool
	i0 = int64(0)
	i4 = int64(4)
	u0 = uint64(0)
	u1 = uint64(1)
	u4 = uint64(4)
	header_end = boundedAdvance(data, i0, u4)
	header_missing = (header_end == i0)
	if header_missing {
		invalid = false
		return invalid
	}
	count = readU32Le(data, i0)
	pos = i4
	index = u0
	for {
		done = (index >= count)
		if done {
			break
		}
		end_ = choiceEntryEnd(data, pos, u0)
		invalid_entry = (end_ == i0)
		if invalid_entry {
			invalid = false
			return invalid
		}
		pos = end_
		index = (index + u1)
	}
	valid = true
	return valid
}

func choiceStreamCount(data []byte) uint64 {
	var i0 int64
	var count uint64
	i0 = int64(0)
	count = readU32Le(data, i0)
	return count
}

func deserializeIntegerChoice(data []byte, start int64) ([]byte, bool) {
	var i0 int64
	var i1 int64
	var i6 int64
	var u0 uint64
	var u10 uint64
	var end_ int64
	var invalid bool
	var none optionalBytes
	var tag uint64
	var wrong_tag bool
	var sub_pos int64
	var sub uint64
	var wrong_sub bool
	var value_start int64
	var value []byte
	var canonical []byte
	var some optionalBytes
	i0 = int64(0)
	i1 = int64(1)
	i6 = int64(6)
	u0 = uint64(0)
	u10 = uint64(10)
	end_ = choiceEntryEnd(data, start, u0)
	invalid = (end_ == i0)
	if invalid {
		none = optionalBytes{}
		return none.value, none.ok
	}
	tag = uint64(data[start])
	wrong_tag = (tag != u0)
	if wrong_tag {
		none = optionalBytes{}
		return none.value, none.ok
	}
	sub_pos = (start + i1)
	sub = uint64(data[sub_pos])
	wrong_sub = (sub != u10)
	if wrong_sub {
		none = optionalBytes{}
		return none.value, none.ok
	}
	value_start = (start + i6)
	value = append([]byte(nil), data[value_start:end_]...)
	canonical = canonicalizeSignedLe(value)
	some = optionalBytes{value: canonical, ok: true}
	return some.value, some.ok
}

func deserializeBooleanChoice(data []byte, start int64) (bool, bool) {
	var i0 int64
	var i1 int64
	var u0 uint64
	var u1 uint64
	var end_ int64
	var invalid bool
	var none optionalBool
	var tag uint64
	var wrong_tag bool
	var value_pos int64
	var byte_ uint64
	var value bool
	var some optionalBool
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u1 = uint64(1)
	end_ = choiceEntryEnd(data, start, u0)
	invalid = (end_ == i0)
	if invalid {
		none = optionalBool{}
		return none.value, none.ok
	}
	tag = uint64(data[start])
	wrong_tag = (tag != u1)
	if wrong_tag {
		none = optionalBool{}
		return none.value, none.ok
	}
	value_pos = (start + i1)
	byte_ = uint64(data[value_pos])
	value = (byte_ != u0)
	some = optionalBool{value: value, ok: true}
	return some.value, some.ok
}

func deserializeFloatChoice(data []byte, start int64) (float64, bool) {
	var i0 int64
	var i1 int64
	var u0 uint64
	var u2 uint64
	var end_ int64
	var invalid bool
	var none optionalF64
	var tag uint64
	var wrong_tag bool
	var value_pos int64
	var bits uint64
	var value float64
	var some optionalF64
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u2 = uint64(2)
	end_ = choiceEntryEnd(data, start, u0)
	invalid = (end_ == i0)
	if invalid {
		none = optionalF64{}
		return none.value, none.ok
	}
	tag = uint64(data[start])
	wrong_tag = (tag != u2)
	if wrong_tag {
		none = optionalF64{}
		return none.value, none.ok
	}
	value_pos = (start + i1)
	bits = readU64Le(data, value_pos)
	value = math.Float64frombits(bits)
	some = optionalF64{value: value, ok: true}
	return some.value, some.ok
}

func deserializeBytesChoice(data []byte, start int64) ([]byte, bool) {
	var i0 int64
	var i5 int64
	var u0 uint64
	var u3 uint64
	var end_ int64
	var invalid bool
	var none optionalBytes
	var tag uint64
	var wrong_tag bool
	var value_start int64
	var value []byte
	var some optionalBytes
	i0 = int64(0)
	i5 = int64(5)
	u0 = uint64(0)
	u3 = uint64(3)
	end_ = choiceEntryEnd(data, start, u0)
	invalid = (end_ == i0)
	if invalid {
		none = optionalBytes{}
		return none.value, none.ok
	}
	tag = uint64(data[start])
	wrong_tag = (tag != u3)
	if wrong_tag {
		none = optionalBytes{}
		return none.value, none.ok
	}
	value_start = (start + i5)
	value = append([]byte(nil), data[value_start:end_]...)
	some = optionalBytes{value: value, ok: true}
	return some.value, some.ok
}

func deserializeStringChoice(data []byte, start int64) ([]byte, bool) {
	var i0 int64
	var i5 int64
	var u0 uint64
	var u4 uint64
	var end_ int64
	var invalid bool
	var none optionalBytes
	var tag uint64
	var wrong_tag bool
	var value_start int64
	var value []byte
	var some optionalBytes
	i0 = int64(0)
	i5 = int64(5)
	u0 = uint64(0)
	u4 = uint64(4)
	end_ = choiceEntryEnd(data, start, u0)
	invalid = (end_ == i0)
	if invalid {
		none = optionalBytes{}
		return none.value, none.ok
	}
	tag = uint64(data[start])
	wrong_tag = (tag != u4)
	if wrong_tag {
		none = optionalBytes{}
		return none.value, none.ok
	}
	value_start = (start + i5)
	value = append([]byte(nil), data[value_start:end_]...)
	some = optionalBytes{value: value, ok: true}
	return some.value, some.ok
}

func failureUseCompressed(compressed_n uint64, raw_n uint64) bool {
	var max uint64
	var shorter bool
	var fits bool
	var use_ bool
	max = maxDecompressedLen()
	shorter = (compressed_n < raw_n)
	fits = (raw_n <= max)
	use_ = (shorter && fits)
	return use_
}

func rawFailureEncode(serialized_choices []byte) string {
	var raw_payload []byte
	var prefix_raw uint64
	var prefix_zlib uint64
	var raw_prefixed []byte
	var compressed []byte
	var compressed_len int64
	var compressed_n uint64
	var raw_len int64
	var raw_n uint64
	var use_compressed bool
	var chosen []byte
	var zlib_payload []byte
	var blob []byte
	raw_payload = []byte{}
	prefix_raw = uint64(0)
	prefix_zlib = uint64(1)
	raw_payload = append(raw_payload, byte(prefix_raw))
	raw_prefixed = appendBytes(raw_payload, serialized_choices)
	compressed = zlibCompress(serialized_choices)
	compressed_len = int64(len(compressed))
	compressed_n = uint64(compressed_len)
	raw_len = int64(len(serialized_choices))
	raw_n = uint64(raw_len)
	use_compressed = failureUseCompressed(compressed_n, raw_n)
	chosen = append([]byte(nil), raw_prefixed...)
	if use_compressed {
		zlib_payload = []byte{}
		zlib_payload = append(zlib_payload, byte(prefix_zlib))
		chosen = appendBytes(zlib_payload, compressed)
	}
	blob = []byte(base64Encode(chosen))
	return string(blob)
}

func rawFailureDecode(blob []byte) ([]byte, bool) {
	var decoded optionalBytes
	var present bool
	var missing bool
	var none optionalBytes
	var payload []byte
	var i0 int64
	var i1 int64
	var u0 uint64
	var n int64
	var empty bool
	var prefix uint64
	var u1_prefix uint64
	var raw bool
	var compressed bool
	var known bool
	var unknown bool
	var serialized []byte
	var inflated optionalBytes
	var inflated_ok bool
	var inflate_failed bool
	var valid bool
	var invalid bool
	var some optionalBytes
	decoded.value, decoded.ok = base64Decode(blob)
	present = decoded.ok
	missing = (!present)
	if missing {
		none = optionalBytes{}
		return none.value, none.ok
	}
	payload = decoded.value
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	n = int64(len(payload))
	empty = (n == i0)
	if empty {
		none = optionalBytes{}
		return none.value, none.ok
	}
	prefix = uint64(payload[i0])
	u1_prefix = uint64(1)
	raw = (prefix == u0)
	compressed = (prefix == u1_prefix)
	known = (raw || compressed)
	unknown = (!known)
	if unknown {
		none = optionalBytes{}
		return none.value, none.ok
	}
	serialized = append([]byte(nil), payload[i1:n]...)
	if compressed {
		inflated.value, inflated.ok = zlibDecompress(serialized)
		inflated_ok = inflated.ok
		inflate_failed = (!inflated_ok)
		if inflate_failed {
			none = optionalBytes{}
			return none.value, none.ok
		}
		serialized = inflated.value
	}
	valid = choiceStreamValidate(serialized)
	invalid = (!valid)
	if invalid {
		none = optionalBytes{}
		return none.value, none.ok
	}
	some = optionalBytes{value: serialized, ok: true}
	return some.value, some.ok
}

func alphabetCodepointAllowed(codepoint uint64) bool {
	var false_ bool
	var true_ bool
	var max uint64
	var surrogate_low uint64
	var surrogate_high uint64
	var too_high bool
	var at_or_above_low bool
	var at_or_below_high bool
	var surrogate bool
	false_ = false
	true_ = true
	max = uint64(1114111)
	surrogate_low = uint64(55296)
	surrogate_high = uint64(57343)
	too_high = (codepoint > max)
	if too_high {
		return false_
	}
	at_or_above_low = (codepoint >= surrogate_low)
	at_or_below_high = (codepoint <= surrogate_high)
	surrogate = (at_or_above_low && at_or_below_high)
	if surrogate {
		return false_
	}
	return true_
}

func alphabetIntervalCount(alphabet []byte) uint64 {
	var u3 uint64
	var length int64
	var length_u64 uint64
	var count uint64
	u3 = uint64(3)
	length = int64(len(alphabet))
	length_u64 = uint64(length)
	count = (length_u64 >> u3)
	return count
}

func alphabetIntervalStart(alphabet []byte, interval uint64) uint64 {
	var i8 int64
	var interval_i64 int64
	var offset int64
	var start uint64
	i8 = int64(8)
	interval_i64 = int64(interval)
	offset = (interval_i64 * i8)
	start = readU32Le(alphabet, offset)
	return start
}

func alphabetIntervalEnd(alphabet []byte, interval uint64) uint64 {
	var i4 int64
	var i8 int64
	var interval_i64 int64
	var base int64
	var offset int64
	var end_ uint64
	i4 = int64(4)
	i8 = int64(8)
	interval_i64 = int64(interval)
	base = (interval_i64 * i8)
	offset = (base + i4)
	end_ = readU32Le(alphabet, offset)
	return end_
}

func alphabetValid(alphabet []byte) bool {
	var false_ bool
	var true_ bool
	var i0 int64
	var u0 uint64
	var u1 uint64
	var u3 uint64
	var length int64
	var empty bool
	var length_u64 uint64
	var count_u64 uint64
	var exact uint64
	var ragged bool
	var previous_end uint64
	var interval uint64
	var done bool
	var start uint64
	var end_ uint64
	var descending bool
	var start_ok bool
	var start_bad bool
	var end_ok bool
	var end_bad bool
	var surrogate_low uint64
	var surrogate_high uint64
	var spans_below bool
	var spans_above bool
	var straddles bool
	var later bool
	var needed uint64
	var touching bool
	false_ = false
	true_ = true
	i0 = int64(0)
	u0 = uint64(0)
	u1 = uint64(1)
	u3 = uint64(3)
	length = int64(len(alphabet))
	empty = (length <= i0)
	if empty {
		return false_
	}
	length_u64 = uint64(length)
	count_u64 = (length_u64 >> u3)
	exact = (count_u64 << u3)
	ragged = (exact != length_u64)
	if ragged {
		return false_
	}
	previous_end = u0
	interval = u0
	for {
		done = (interval >= count_u64)
		if done {
			break
		}
		start = alphabetIntervalStart(alphabet, interval)
		end_ = alphabetIntervalEnd(alphabet, interval)
		descending = (start > end_)
		if descending {
			return false_
		}
		start_ok = alphabetCodepointAllowed(start)
		start_bad = (!start_ok)
		if start_bad {
			return false_
		}
		end_ok = alphabetCodepointAllowed(end_)
		end_bad = (!end_ok)
		if end_bad {
			return false_
		}
		surrogate_low = uint64(55296)
		surrogate_high = uint64(57343)
		spans_below = (start <= surrogate_high)
		spans_above = (end_ >= surrogate_low)
		straddles = (spans_below && spans_above)
		if straddles {
			return false_
		}
		later = (interval > u0)
		if later {
			needed = (previous_end + u1)
			touching = (start <= needed)
			if touching {
				return false_
			}
		}
		previous_end = end_
		interval = (interval + u1)
	}
	return true_
}

func alphabetSize(alphabet []byte) uint64 {
	var u0 uint64
	var u1 uint64
	var count uint64
	var size uint64
	var interval uint64
	var done bool
	var start uint64
	var end_ uint64
	var span uint64
	var members uint64
	u0 = uint64(0)
	u1 = uint64(1)
	count = alphabetIntervalCount(alphabet)
	size = u0
	interval = u0
	for {
		done = (interval >= count)
		if done {
			break
		}
		start = alphabetIntervalStart(alphabet, interval)
		end_ = alphabetIntervalEnd(alphabet, interval)
		span = (end_ - start)
		members = (span + u1)
		size = (size + members)
		interval = (interval + u1)
	}
	return size
}

func alphabetIndexOf(alphabet []byte, codepoint uint64) (uint64, bool) {
	var u0 uint64
	var u1 uint64
	var count uint64
	var offset uint64
	var interval uint64
	var done bool
	var start uint64
	var end_ uint64
	var at_or_above bool
	var at_or_below bool
	var inside bool
	var within uint64
	var index uint64
	var found optionalU64
	var span uint64
	var members uint64
	var missing optionalU64
	u0 = uint64(0)
	u1 = uint64(1)
	count = alphabetIntervalCount(alphabet)
	offset = u0
	interval = u0
	for {
		done = (interval >= count)
		if done {
			break
		}
		start = alphabetIntervalStart(alphabet, interval)
		end_ = alphabetIntervalEnd(alphabet, interval)
		at_or_above = (codepoint >= start)
		at_or_below = (codepoint <= end_)
		inside = (at_or_above && at_or_below)
		if inside {
			within = (codepoint - start)
			index = (offset + within)
			found = optionalU64{value: index, ok: true}
			return found.value, found.ok
		}
		span = (end_ - start)
		members = (span + u1)
		offset = (offset + members)
		interval = (interval + u1)
	}
	missing = optionalU64{}
	return missing.value, missing.ok
}

func alphabetCodepointAt(alphabet []byte, index uint64) (uint64, bool) {
	var u0 uint64
	var u1 uint64
	var count uint64
	var remaining uint64
	var interval uint64
	var done bool
	var start uint64
	var end_ uint64
	var span uint64
	var members uint64
	var inside bool
	var codepoint uint64
	var found optionalU64
	var missing optionalU64
	u0 = uint64(0)
	u1 = uint64(1)
	count = alphabetIntervalCount(alphabet)
	remaining = index
	interval = u0
	for {
		done = (interval >= count)
		if done {
			break
		}
		start = alphabetIntervalStart(alphabet, interval)
		end_ = alphabetIntervalEnd(alphabet, interval)
		span = (end_ - start)
		members = (span + u1)
		inside = (remaining < members)
		if inside {
			codepoint = (start + remaining)
			found = optionalU64{value: codepoint, ok: true}
			return found.value, found.ok
		}
		remaining = (remaining - members)
		interval = (interval + u1)
	}
	missing = optionalU64{}
	return missing.value, missing.ok
}

func alphabetContains(alphabet []byte, codepoint uint64) bool {
	var index optionalU64
	var present bool
	index.value, index.ok = alphabetIndexOf(alphabet, codepoint)
	present = index.ok
	return present
}

func alphabetIndexAbove(alphabet []byte, codepoint uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var count uint64
	var below uint64
	var interval uint64
	var done bool
	var start uint64
	var end_ uint64
	var before_interval bool
	var inside bool
	var within uint64
	var index uint64
	var span uint64
	var members uint64
	u0 = uint64(0)
	u1 = uint64(1)
	count = alphabetIntervalCount(alphabet)
	below = u0
	interval = u0
	for {
		done = (interval >= count)
		if done {
			break
		}
		start = alphabetIntervalStart(alphabet, interval)
		end_ = alphabetIntervalEnd(alphabet, interval)
		before_interval = (codepoint <= start)
		if before_interval {
			return below
		}
		inside = (codepoint <= end_)
		if inside {
			within = (codepoint - start)
			index = (below + within)
			return index
		}
		span = (end_ - start)
		members = (span + u1)
		below = (below + members)
		interval = (interval + u1)
	}
	return below
}

func alphabetShrinkPivot(alphabet []byte) uint64 {
	var zero uint64
	var pivot uint64
	zero = uint64(48)
	pivot = alphabetIndexAbove(alphabet, zero)
	return pivot
}

func alphabetShrinkLast(alphabet []byte) uint64 {
	var u0 uint64
	var u1 uint64
	var upper_z uint64
	var found optionalU64
	var present bool
	var index uint64
	var pivot uint64
	var at_front bool
	var last uint64
	u0 = uint64(0)
	u1 = uint64(1)
	upper_z = uint64(90)
	found.value, found.ok = alphabetIndexOf(alphabet, upper_z)
	present = found.ok
	if present {
		index = found.value
		return index
	}
	pivot = alphabetShrinkPivot(alphabet)
	at_front = (pivot == u0)
	if at_front {
		return u0
	}
	last = (pivot - u1)
	return last
}

func alphabetShrinkPromoted(alphabet []byte) uint64 {
	var u0 uint64
	var u1 uint64
	var pivot uint64
	var last uint64
	var empty bool
	var span uint64
	var promoted uint64
	u0 = uint64(0)
	u1 = uint64(1)
	pivot = alphabetShrinkPivot(alphabet)
	last = alphabetShrinkLast(alphabet)
	empty = (last < pivot)
	if empty {
		return u0
	}
	span = (last - pivot)
	promoted = (span + u1)
	return promoted
}

func alphabetShrinkIndex(alphabet []byte, rank uint64) uint64 {
	var size uint64
	var past_end bool
	var pivot uint64
	var last uint64
	var promoted uint64
	var in_promoted bool
	var index uint64
	var displaced bool
	size = alphabetSize(alphabet)
	past_end = (rank >= size)
	if past_end {
		return rank
	}
	pivot = alphabetShrinkPivot(alphabet)
	last = alphabetShrinkLast(alphabet)
	promoted = alphabetShrinkPromoted(alphabet)
	in_promoted = (rank < promoted)
	if in_promoted {
		index = (pivot + rank)
		return index
	}
	displaced = (rank <= last)
	if displaced {
		index = (rank - promoted)
		return index
	}
	return rank
}

func alphabetShrinkRank(alphabet []byte, index uint64) uint64 {
	var size uint64
	var past_end bool
	var pivot uint64
	var last uint64
	var promoted uint64
	var at_or_above bool
	var at_or_below bool
	var in_promoted bool
	var rank uint64
	var below bool
	size = alphabetSize(alphabet)
	past_end = (index >= size)
	if past_end {
		return index
	}
	pivot = alphabetShrinkPivot(alphabet)
	last = alphabetShrinkLast(alphabet)
	promoted = alphabetShrinkPromoted(alphabet)
	at_or_above = (index >= pivot)
	at_or_below = (index <= last)
	in_promoted = (at_or_above && at_or_below)
	if in_promoted {
		rank = (index - pivot)
		return rank
	}
	below = (index < pivot)
	if below {
		rank = (index + promoted)
		return rank
	}
	return index
}

func alphabetShrinkCodepoint(alphabet []byte, rank uint64) (uint64, bool) {
	var index uint64
	var codepoint optionalU64
	index = alphabetShrinkIndex(alphabet, rank)
	codepoint.value, codepoint.ok = alphabetCodepointAt(alphabet, index)
	return codepoint.value, codepoint.ok
}

func alphabetRange(min_codepoint uint64, max_codepoint uint64) []byte {
	var out []byte
	var max uint64
	var surrogate_below uint64
	var surrogate_above uint64
	var low uint64
	var high uint64
	var over bool
	var empty bool
	var starts_below bool
	var first_end uint64
	var clip bool
	var start_bytes []byte
	var end_bytes []byte
	var ends_above bool
	var second_start uint64
	var lift bool
	out = []byte{}
	max = uint64(1114111)
	surrogate_below = uint64(55295)
	surrogate_above = uint64(57344)
	low = min_codepoint
	high = max_codepoint
	over = (high > max)
	if over {
		high = max
	}
	empty = (low > high)
	if empty {
		return out
	}
	starts_below = (low <= surrogate_below)
	if starts_below {
		first_end = high
		clip = (first_end > surrogate_below)
		if clip {
			first_end = surrogate_below
		}
		start_bytes = u32ToLe(low)
		end_bytes = u32ToLe(first_end)
		out = appendBytes(out, start_bytes)
		out = appendBytes(out, end_bytes)
	}
	ends_above = (high >= surrogate_above)
	if ends_above {
		second_start = low
		lift = (second_start < surrogate_above)
		if lift {
			second_start = surrogate_above
		}
		start_bytes = u32ToLe(second_start)
		end_bytes = u32ToLe(high)
		out = appendBytes(out, start_bytes)
		out = appendBytes(out, end_bytes)
	}
	return out
}

func alphabetInsert(alphabet []byte, codepoint uint64) []byte {
	var u0 uint64
	var u1 uint64
	var count uint64
	var out []byte
	var allowed bool
	var rejected bool
	var unchanged []byte
	var pending_start uint64
	var pending_end uint64
	var placed bool
	var interval uint64
	var done bool
	var start uint64
	var end_ uint64
	var gap_start uint64
	var strictly_below bool
	var start_bytes []byte
	var end_bytes []byte
	var touches_below bool
	var gap_end uint64
	var strictly_above bool
	var not_yet bool
	var pending_start_bytes []byte
	var pending_end_bytes []byte
	var overlaps bool
	var lower bool
	var higher bool
	var unplaced bool
	u0 = uint64(0)
	u1 = uint64(1)
	count = alphabetIntervalCount(alphabet)
	out = []byte{}
	allowed = alphabetCodepointAllowed(codepoint)
	rejected = (!allowed)
	if rejected {
		unchanged = appendBytes(out, alphabet)
		return unchanged
	}
	pending_start = codepoint
	pending_end = codepoint
	placed = false
	interval = u0
	for {
		done = (interval >= count)
		if done {
			break
		}
		start = alphabetIntervalStart(alphabet, interval)
		end_ = alphabetIntervalEnd(alphabet, interval)
		gap_start = (end_ + u1)
		strictly_below = (gap_start < pending_start)
		if strictly_below {
			start_bytes = u32ToLe(start)
			end_bytes = u32ToLe(end_)
			out = appendBytes(out, start_bytes)
			out = appendBytes(out, end_bytes)
		}
		touches_below = (!strictly_below)
		if touches_below {
			gap_end = (pending_end + u1)
			strictly_above = (start > gap_end)
			if strictly_above {
				not_yet = (!placed)
				if not_yet {
					pending_start_bytes = u32ToLe(pending_start)
					pending_end_bytes = u32ToLe(pending_end)
					out = appendBytes(out, pending_start_bytes)
					out = appendBytes(out, pending_end_bytes)
					placed = true
				}
				start_bytes = u32ToLe(start)
				end_bytes = u32ToLe(end_)
				out = appendBytes(out, start_bytes)
				out = appendBytes(out, end_bytes)
			}
			overlaps = (!strictly_above)
			if overlaps {
				lower = (start < pending_start)
				if lower {
					pending_start = start
				}
				higher = (end_ > pending_end)
				if higher {
					pending_end = end_
				}
			}
		}
		interval = (interval + u1)
	}
	unplaced = (!placed)
	if unplaced {
		pending_start_bytes = u32ToLe(pending_start)
		pending_end_bytes = u32ToLe(pending_end)
		out = appendBytes(out, pending_start_bytes)
		out = appendBytes(out, pending_end_bytes)
	}
	return out
}

func stringLength(value []byte) int64 {
	var u2 uint64
	var length int64
	var length_u64 uint64
	var count_u64 uint64
	var count int64
	u2 = uint64(2)
	length = int64(len(value))
	length_u64 = uint64(length)
	count_u64 = (length_u64 >> u2)
	count = int64(count_u64)
	return count
}

func stringCodepointAt(value []byte, position int64) uint64 {
	var i4 int64
	var offset int64
	var codepoint uint64
	i4 = int64(4)
	offset = (position * i4)
	codepoint = readU32Le(value, offset)
	return codepoint
}

func stringValidate(alphabet []byte, min_size int64, max_size int64, value []byte) bool {
	var false_ bool
	var true_ bool
	var i0 int64
	var i1 int64
	var i4 int64
	var length int64
	var count int64
	var exact int64
	var ragged bool
	var too_short bool
	var too_long bool
	var position int64
	var done bool
	var codepoint uint64
	var member bool
	var outside bool
	false_ = false
	true_ = true
	i0 = int64(0)
	i1 = int64(1)
	i4 = int64(4)
	length = int64(len(value))
	count = stringLength(value)
	exact = (count * i4)
	ragged = (exact != length)
	if ragged {
		return false_
	}
	too_short = (count < min_size)
	if too_short {
		return false_
	}
	too_long = (count > max_size)
	if too_long {
		return false_
	}
	position = i0
	for {
		done = (position >= count)
		if done {
			break
		}
		codepoint = stringCodepointAt(value, position)
		member = alphabetContains(alphabet, codepoint)
		outside = (!member)
		if outside {
			return false_
		}
		position = (position + i1)
	}
	return true_
}

func stringSimplest(alphabet []byte, min_size int64) []byte {
	var out []byte
	var i0 int64
	var i1 int64
	var u0 uint64
	var first optionalU64
	var present bool
	var empty_alphabet bool
	var codepoint uint64
	var packed_ []byte
	var position int64
	var done bool
	out = []byte{}
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	first.value, first.ok = alphabetShrinkCodepoint(alphabet, u0)
	present = first.ok
	empty_alphabet = (!present)
	if empty_alphabet {
		return out
	}
	codepoint = first.value
	packed_ = u32ToLe(codepoint)
	position = i0
	for {
		done = (position >= min_size)
		if done {
			break
		}
		out = appendBytes(out, packed_)
		position = (position + i1)
	}
	return out
}

func stringUnit(alphabet []byte, min_size int64, max_size int64) []byte {
	var out []byte
	var i0 int64
	var i1 int64
	var u0 uint64
	var u1 uint64
	var first optionalU64
	var present bool
	var empty_alphabet bool
	var lowest uint64
	var next_ optionalU64
	var has_next bool
	var stepped uint64
	var second uint64
	var lowest_packed []byte
	var stepped_packed []byte
	var has_minimum bool
	var last int64
	var position int64
	var done bool
	var is_last bool
	var is_earlier bool
	var has_room bool
	out = []byte{}
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u1 = uint64(1)
	first.value, first.ok = alphabetShrinkCodepoint(alphabet, u0)
	present = first.ok
	empty_alphabet = (!present)
	if empty_alphabet {
		return out
	}
	lowest = first.value
	next_.value, next_.ok = alphabetShrinkCodepoint(alphabet, u1)
	has_next = next_.ok
	stepped = lowest
	if has_next {
		second = next_.value
		stepped = second
	}
	lowest_packed = u32ToLe(lowest)
	stepped_packed = u32ToLe(stepped)
	has_minimum = (min_size > i0)
	if has_minimum {
		last = (min_size - i1)
		position = i0
		for {
			done = (position >= min_size)
			if done {
				break
			}
			is_last = (position == last)
			if is_last {
				out = appendBytes(out, stepped_packed)
			}
			is_earlier = (!is_last)
			if is_earlier {
				out = appendBytes(out, lowest_packed)
			}
			position = (position + i1)
		}
		return out
	}
	has_room = (max_size > i0)
	if has_room {
		out = appendBytes(out, stepped_packed)
	}
	return out
}

func stringDuplicateRank(alphabet []byte, value []byte) (uint64, bool) {
	var i0 int64
	var i1 int64
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var length int64
	var best uint64
	var found bool
	var at int64
	var done bool
	var codepoint uint64
	var index optionalU64
	var present bool
	var index_value uint64
	var rank uint64
	var lowerable bool
	var improves bool
	var not_found bool
	var better bool
	var worth_counting bool
	var seen uint64
	var other int64
	var counted bool
	var other_codepoint uint64
	var other_index optionalU64
	var other_present bool
	var other_index_value uint64
	var other_rank uint64
	var same bool
	var repeated bool
	var missing bool
	var none optionalU64
	var some optionalU64
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	length = stringLength(value)
	best = u0
	found = false
	at = i0
	for {
		done = (at >= length)
		if done {
			break
		}
		codepoint = stringCodepointAt(value, at)
		index.value, index.ok = alphabetIndexOf(alphabet, codepoint)
		present = index.ok
		if present {
			index_value = index.value
			rank = alphabetShrinkRank(alphabet, index_value)
			lowerable = (rank != u0)
			improves = (rank > best)
			not_found = (!found)
			better = (improves || not_found)
			worth_counting = (lowerable && better)
			if worth_counting {
				seen = u0
				other = i0
				for {
					counted = (other >= length)
					if counted {
						break
					}
					other_codepoint = stringCodepointAt(value, other)
					other_index.value, other_index.ok = alphabetIndexOf(alphabet, other_codepoint)
					other_present = other_index.ok
					if other_present {
						other_index_value = other_index.value
						other_rank = alphabetShrinkRank(alphabet, other_index_value)
						same = (other_rank == rank)
						if same {
							seen = (seen + u1)
						}
					}
					other = (other + i1)
				}
				repeated = (seen >= u2)
				if repeated {
					best = rank
					found = true
				}
			}
		}
		at = (at + i1)
	}
	missing = (!found)
	if missing {
		none = optionalU64{}
		return none.value, none.ok
	}
	some = optionalU64{value: best, ok: true}
	return some.value, some.ok
}

func stringReplaceRank(alphabet []byte, value []byte, from_rank uint64, to_rank uint64) ([]byte, bool) {
	var i0 int64
	var i1 int64
	var unchanged bool
	var none_same optionalBytes
	var target optionalU64
	var target_present bool
	var target_missing bool
	var none_target optionalBytes
	var target_codepoint uint64
	var length int64
	var out []byte
	var at int64
	var done bool
	var codepoint uint64
	var emitted uint64
	var index optionalU64
	var present bool
	var index_value uint64
	var rank uint64
	var matches bool
	var packed_ []byte
	var some optionalBytes
	i0 = int64(0)
	i1 = int64(1)
	unchanged = (from_rank == to_rank)
	if unchanged {
		none_same = optionalBytes{}
		return none_same.value, none_same.ok
	}
	target.value, target.ok = alphabetShrinkCodepoint(alphabet, to_rank)
	target_present = target.ok
	target_missing = (!target_present)
	if target_missing {
		none_target = optionalBytes{}
		return none_target.value, none_target.ok
	}
	target_codepoint = target.value
	length = stringLength(value)
	out = []byte{}
	at = i0
	for {
		done = (at >= length)
		if done {
			break
		}
		codepoint = stringCodepointAt(value, at)
		emitted = codepoint
		index.value, index.ok = alphabetIndexOf(alphabet, codepoint)
		present = index.ok
		if present {
			index_value = index.value
			rank = alphabetShrinkRank(alphabet, index_value)
			matches = (rank == from_rank)
			if matches {
				emitted = target_codepoint
			}
		}
		packed_ = u32ToLe(emitted)
		out = appendBytes(out, packed_)
		at = (at + i1)
	}
	some = optionalBytes{value: out, ok: true}
	return some.value, some.ok
}

func stringDropCharacter(value []byte, min_size int64, position int64) ([]byte, bool) {
	var i0 int64
	var i1 int64
	var length int64
	var at_minimum bool
	var none_minimum optionalBytes
	var before_start bool
	var none_before optionalBytes
	var past_end bool
	var none_past optionalBytes
	var out []byte
	var at int64
	var done bool
	var dropped bool
	var kept bool
	var codepoint uint64
	var packed_ []byte
	var some optionalBytes
	i0 = int64(0)
	i1 = int64(1)
	length = stringLength(value)
	at_minimum = (length <= min_size)
	if at_minimum {
		none_minimum = optionalBytes{}
		return none_minimum.value, none_minimum.ok
	}
	before_start = (position < i0)
	if before_start {
		none_before = optionalBytes{}
		return none_before.value, none_before.ok
	}
	past_end = (position >= length)
	if past_end {
		none_past = optionalBytes{}
		return none_past.value, none_past.ok
	}
	out = []byte{}
	at = i0
	for {
		done = (at >= length)
		if done {
			break
		}
		dropped = (at == position)
		kept = (!dropped)
		if kept {
			codepoint = stringCodepointAt(value, at)
			packed_ = u32ToLe(codepoint)
			out = appendBytes(out, packed_)
		}
		at = (at + i1)
	}
	some = optionalBytes{value: out, ok: true}
	return some.value, some.ok
}

func stringToIndex(alphabet []byte, min_size int64, value []byte) []byte {
	var i0 int64
	var i1 int64
	var u0 uint64
	var size uint64
	var base []byte
	var count int64
	var offset []byte
	var length int64
	var done bool
	var exponent uint64
	var bucket []byte
	var position []byte
	var pos int64
	var codepoint uint64
	var found optionalU64
	var present bool
	var rank uint64
	var alphabet_index uint64
	var digit []byte
	var shifted []byte
	var index []byte
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	size = alphabetSize(alphabet)
	base = u64ToBigintLe(size)
	count = stringLength(value)
	offset = bigintZero()
	length = min_size
	for {
		done = (length >= count)
		if done {
			break
		}
		exponent = uint64(length)
		bucket = positivePowLe(base, exponent)
		offset = signedAddLe(offset, bucket)
		length = (length + i1)
	}
	position = bigintZero()
	pos = i0
	for {
		done = (pos >= count)
		if done {
			break
		}
		codepoint = stringCodepointAt(value, pos)
		found.value, found.ok = alphabetIndexOf(alphabet, codepoint)
		present = found.ok
		rank = u0
		if present {
			alphabet_index = found.value
			rank = alphabetShrinkRank(alphabet, alphabet_index)
		}
		digit = u64ToBigintLe(rank)
		shifted = signedMulLe(position, base)
		position = signedAddLe(shifted, digit)
		pos = (pos + i1)
	}
	index = signedAddLe(offset, position)
	return index
}

func stringMaxIndex(alphabet []byte, min_size int64, max_size int64) []byte {
	var i0 int64
	var i1 int64
	var u0 uint64
	var u1 uint64
	var size uint64
	var no_members bool
	var zero []byte
	var top_rank uint64
	var top optionalU64
	var present bool
	var missing bool
	var codepoint uint64
	var packed_ []byte
	var last []byte
	var position int64
	var done bool
	var index []byte
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u1 = uint64(1)
	size = alphabetSize(alphabet)
	no_members = (size == u0)
	if no_members {
		zero = bigintZero()
		return zero
	}
	top_rank = (size - u1)
	top.value, top.ok = alphabetShrinkCodepoint(alphabet, top_rank)
	present = top.ok
	missing = (!present)
	if missing {
		zero = bigintZero()
		return zero
	}
	codepoint = top.value
	packed_ = u32ToLe(codepoint)
	last = []byte{}
	position = i0
	for {
		done = (position >= max_size)
		if done {
			break
		}
		last = appendBytes(last, packed_)
		position = (position + i1)
	}
	index = stringToIndex(alphabet, min_size, last)
	return index
}

func stringFromIndex(alphabet []byte, min_size int64, max_size int64, index_raw []byte) ([]byte, bool) {
	var i0 int64
	var i1 int64
	var u0 uint64
	var size uint64
	var no_members bool
	var none optionalBytes
	var base []byte
	var remaining []byte
	var length int64
	var past_end bool
	var exponent uint64
	var bucket []byte
	var bucket_cmp int64
	var inside_bucket bool
	var out []byte
	var empty bool
	var some_empty optionalBytes
	var power int64
	var power_u64 uint64
	var divisor []byte
	var pos int64
	var done bool
	var digit []byte
	var digit_u64 optionalU64
	var digit_fits bool
	var rank uint64
	var found optionalU64
	var present bool
	var codepoint uint64
	var packed_ []byte
	var some optionalBytes
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	size = alphabetSize(alphabet)
	no_members = (size == u0)
	if no_members {
		none = optionalBytes{}
		return none.value, none.ok
	}
	base = u64ToBigintLe(size)
	remaining = signedAbsLe(index_raw)
	length = min_size
	for {
		past_end = (length > max_size)
		if past_end {
			break
		}
		exponent = uint64(length)
		bucket = positivePowLe(base, exponent)
		bucket_cmp = signedCompareLe(remaining, bucket)
		inside_bucket = (bucket_cmp < i0)
		if inside_bucket {
			out = []byte{}
			empty = (length == i0)
			if empty {
				some_empty = optionalBytes{value: out, ok: true}
				return some_empty.value, some_empty.ok
			}
			power = (length - i1)
			power_u64 = uint64(power)
			divisor = positivePowLe(base, power_u64)
			pos = i0
			for {
				done = (pos >= length)
				if done {
					break
				}
				digit = positiveDivLeNonzero(remaining, divisor)
				digit_u64.value, digit_u64.ok = bigintToU64Le(digit)
				digit_fits = digit_u64.ok
				rank = u0
				if digit_fits {
					rank = digit_u64.value
				}
				found.value, found.ok = alphabetShrinkCodepoint(alphabet, rank)
				present = found.ok
				codepoint = u0
				if present {
					codepoint = found.value
				}
				packed_ = u32ToLe(codepoint)
				out = appendBytes(out, packed_)
				remaining = positiveRemLeNonzero(remaining, divisor)
				divisor = positiveDivLeNonzero(divisor, base)
				pos = (pos + i1)
			}
			some = optionalBytes{value: out, ok: true}
			return some.value, some.ok
		}
		remaining = signedSubLe(remaining, bucket)
		length = (length + i1)
	}
	none = optionalBytes{}
	return none.value, none.ok
}

func choiceDeleteRange(data []byte, start_index uint64, end_index uint64) ([]byte, bool) {
	var i4 int64
	var u0 uint64
	var u1 uint64
	var stream_valid bool
	var stream_invalid bool
	var none optionalBytes
	var count uint64
	var ordered bool
	var end_valid bool
	var bounds_valid bool
	var bounds_invalid bool
	var position int64
	var index uint64
	var start_offset int64
	var done bool
	var at_start bool
	var next_ int64
	var empty_range bool
	var end_offset int64
	var data_end int64
	var before []byte
	var after []byte
	var removed uint64
	var new_count uint64
	var header []byte
	var with_before []byte
	var out []byte
	var some optionalBytes
	i4 = int64(4)
	u0 = uint64(0)
	u1 = uint64(1)
	stream_valid = choiceStreamValidate(data)
	stream_invalid = (!stream_valid)
	if stream_invalid {
		none = optionalBytes{}
		return none.value, none.ok
	}
	count = choiceStreamCount(data)
	ordered = (start_index <= end_index)
	end_valid = (end_index <= count)
	bounds_valid = (ordered && end_valid)
	bounds_invalid = (!bounds_valid)
	if bounds_invalid {
		none = optionalBytes{}
		return none.value, none.ok
	}
	position = i4
	index = u0
	start_offset = i4
	for {
		done = (index >= end_index)
		if done {
			break
		}
		at_start = (index == start_index)
		if at_start {
			start_offset = position
		}
		next_ = choiceEntryEnd(data, position, u0)
		position = next_
		index = (index + u1)
	}
	empty_range = (start_index == end_index)
	if empty_range {
		start_offset = position
	}
	end_offset = position
	data_end = int64(len(data))
	before = append([]byte(nil), data[i4:start_offset]...)
	after = append([]byte(nil), data[end_offset:data_end]...)
	removed = (end_index - start_index)
	new_count = (count - removed)
	header = choiceListHeader(new_count)
	with_before = appendBytes(header, before)
	out = appendBytes(with_before, after)
	some = optionalBytes{value: out, ok: true}
	return some.value, some.ok
}

func choiceSwapRanges(data []byte, a_start uint64, a_end uint64, b_start uint64, b_end uint64) ([]byte, bool) {
	var i4 int64
	var u0 uint64
	var u1 uint64
	var stream_valid bool
	var stream_invalid bool
	var none_stream optionalBytes
	var count uint64
	var a_ordered bool
	var disjoint bool
	var b_ordered bool
	var in_range bool
	var front_ok bool
	var back_ok bool
	var bounds_valid bool
	var bounds_invalid bool
	var none_bounds optionalBytes
	var position int64
	var index uint64
	var a_start_offset int64
	var a_end_offset int64
	var b_start_offset int64
	var done bool
	var at_a_start bool
	var at_a_end bool
	var at_b_start bool
	var next_ int64
	var a_start_at_end bool
	var a_end_at_end bool
	var b_start_at_end bool
	var b_end_offset int64
	var data_end int64
	var before []byte
	var first []byte
	var middle []byte
	var second []byte
	var after []byte
	var header []byte
	var with_before []byte
	var with_second []byte
	var with_middle []byte
	var with_first []byte
	var out []byte
	var some optionalBytes
	i4 = int64(4)
	u0 = uint64(0)
	u1 = uint64(1)
	stream_valid = choiceStreamValidate(data)
	stream_invalid = (!stream_valid)
	if stream_invalid {
		none_stream = optionalBytes{}
		return none_stream.value, none_stream.ok
	}
	count = choiceStreamCount(data)
	a_ordered = (a_start <= a_end)
	disjoint = (a_end <= b_start)
	b_ordered = (b_start <= b_end)
	in_range = (b_end <= count)
	front_ok = (a_ordered && disjoint)
	back_ok = (b_ordered && in_range)
	bounds_valid = (front_ok && back_ok)
	bounds_invalid = (!bounds_valid)
	if bounds_invalid {
		none_bounds = optionalBytes{}
		return none_bounds.value, none_bounds.ok
	}
	position = i4
	index = u0
	a_start_offset = i4
	a_end_offset = i4
	b_start_offset = i4
	for {
		done = (index >= b_end)
		if done {
			break
		}
		at_a_start = (index == a_start)
		if at_a_start {
			a_start_offset = position
		}
		at_a_end = (index == a_end)
		if at_a_end {
			a_end_offset = position
		}
		at_b_start = (index == b_start)
		if at_b_start {
			b_start_offset = position
		}
		next_ = choiceEntryEnd(data, position, u0)
		position = next_
		index = (index + u1)
	}
	a_start_at_end = (a_start == b_end)
	if a_start_at_end {
		a_start_offset = position
	}
	a_end_at_end = (a_end == b_end)
	if a_end_at_end {
		a_end_offset = position
	}
	b_start_at_end = (b_start == b_end)
	if b_start_at_end {
		b_start_offset = position
	}
	b_end_offset = position
	data_end = int64(len(data))
	before = append([]byte(nil), data[i4:a_start_offset]...)
	first = append([]byte(nil), data[a_start_offset:a_end_offset]...)
	middle = append([]byte(nil), data[a_end_offset:b_start_offset]...)
	second = append([]byte(nil), data[b_start_offset:b_end_offset]...)
	after = append([]byte(nil), data[b_end_offset:data_end]...)
	header = choiceListHeader(count)
	with_before = appendBytes(header, before)
	with_second = appendBytes(with_before, second)
	with_middle = appendBytes(with_second, middle)
	with_first = appendBytes(with_middle, first)
	out = appendBytes(with_first, after)
	some = optionalBytes{value: out, ok: true}
	return some.value, some.ok
}

func choiceDuplicateRanges(data []byte, a_start uint64, a_end uint64, b_start uint64, b_end uint64, use_a bool) ([]byte, bool) {
	var i4 int64
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var stream_valid bool
	var stream_invalid bool
	var none_stream optionalBytes
	var count uint64
	var a_ordered bool
	var disjoint bool
	var b_ordered bool
	var in_range bool
	var front_ok bool
	var back_ok bool
	var bounds_valid bool
	var bounds_invalid bool
	var none_bounds optionalBytes
	var position int64
	var index uint64
	var a_start_offset int64
	var a_end_offset int64
	var b_start_offset int64
	var done bool
	var at_a_start bool
	var at_a_end bool
	var at_b_start bool
	var next_ int64
	var b_end_offset int64
	var data_end int64
	var before []byte
	var first []byte
	var middle []byte
	var second []byte
	var after []byte
	var donor []byte
	var donor_count uint64
	var a_count uint64
	var b_count uint64
	var removed uint64
	var base_count uint64
	var twice_donor uint64
	var new_count uint64
	var header []byte
	var with_before []byte
	var with_first []byte
	var with_middle []byte
	var with_second []byte
	var out []byte
	var some optionalBytes
	i4 = int64(4)
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	stream_valid = choiceStreamValidate(data)
	stream_invalid = (!stream_valid)
	if stream_invalid {
		none_stream = optionalBytes{}
		return none_stream.value, none_stream.ok
	}
	count = choiceStreamCount(data)
	a_ordered = (a_start < a_end)
	disjoint = (a_end <= b_start)
	b_ordered = (b_start < b_end)
	in_range = (b_end <= count)
	front_ok = (a_ordered && disjoint)
	back_ok = (b_ordered && in_range)
	bounds_valid = (front_ok && back_ok)
	bounds_invalid = (!bounds_valid)
	if bounds_invalid {
		none_bounds = optionalBytes{}
		return none_bounds.value, none_bounds.ok
	}
	position = i4
	index = u0
	a_start_offset = i4
	a_end_offset = i4
	b_start_offset = i4
	for {
		done = (index >= b_end)
		if done {
			break
		}
		at_a_start = (index == a_start)
		if at_a_start {
			a_start_offset = position
		}
		at_a_end = (index == a_end)
		if at_a_end {
			a_end_offset = position
		}
		at_b_start = (index == b_start)
		if at_b_start {
			b_start_offset = position
		}
		next_ = choiceEntryEnd(data, position, u0)
		position = next_
		index = (index + u1)
	}
	b_end_offset = position
	data_end = int64(len(data))
	before = append([]byte(nil), data[i4:a_start_offset]...)
	first = append([]byte(nil), data[a_start_offset:a_end_offset]...)
	middle = append([]byte(nil), data[a_end_offset:b_start_offset]...)
	second = append([]byte(nil), data[b_start_offset:b_end_offset]...)
	after = append([]byte(nil), data[b_end_offset:data_end]...)
	donor = append([]byte(nil), second...)
	donor_count = (b_end - b_start)
	if use_a {
		donor = append([]byte(nil), first...)
		donor_count = (a_end - a_start)
	}
	a_count = (a_end - a_start)
	b_count = (b_end - b_start)
	removed = (a_count + b_count)
	base_count = (count - removed)
	twice_donor = (donor_count * u2)
	new_count = (base_count + twice_donor)
	header = choiceListHeader(new_count)
	with_before = appendBytes(header, before)
	with_first = appendBytes(with_before, donor)
	with_middle = appendBytes(with_first, middle)
	with_second = appendBytes(with_middle, donor)
	out = appendBytes(with_second, after)
	some = optionalBytes{value: out, ok: true}
	return some.value, some.ok
}

func choiceDuplicateNestedPrefix(data []byte, outer_start uint64, outer_end uint64, inner_start uint64, inner_end uint64) ([]byte, bool) {
	var i4 int64
	var u0 uint64
	var u1 uint64
	var stream_valid bool
	var stream_invalid bool
	var none_stream optionalBytes
	var count uint64
	var outer_nonempty bool
	var inner_nonempty bool
	var starts_inside bool
	var ends_inside bool
	var in_range bool
	var nonempty bool
	var contained bool
	var shape_valid bool
	var bounds_valid bool
	var bounds_invalid bool
	var none_bounds optionalBytes
	var position int64
	var index uint64
	var outer_offset int64
	var done bool
	var at_outer bool
	var next_ int64
	var same_start bool
	var inner_offset int64
	var data_end int64
	var before_inner []byte
	var from_outer []byte
	var prefix_count uint64
	var new_count uint64
	var header []byte
	var with_before []byte
	var out []byte
	var some optionalBytes
	i4 = int64(4)
	u0 = uint64(0)
	u1 = uint64(1)
	stream_valid = choiceStreamValidate(data)
	stream_invalid = (!stream_valid)
	if stream_invalid {
		none_stream = optionalBytes{}
		return none_stream.value, none_stream.ok
	}
	count = choiceStreamCount(data)
	outer_nonempty = (outer_start < outer_end)
	inner_nonempty = (inner_start < inner_end)
	starts_inside = (outer_start <= inner_start)
	ends_inside = (inner_end <= outer_end)
	in_range = (outer_end <= count)
	nonempty = (outer_nonempty && inner_nonempty)
	contained = (starts_inside && ends_inside)
	shape_valid = (nonempty && contained)
	bounds_valid = (shape_valid && in_range)
	bounds_invalid = (!bounds_valid)
	if bounds_invalid {
		none_bounds = optionalBytes{}
		return none_bounds.value, none_bounds.ok
	}
	position = i4
	index = u0
	outer_offset = i4
	for {
		done = (index >= inner_start)
		if done {
			break
		}
		at_outer = (index == outer_start)
		if at_outer {
			outer_offset = position
		}
		next_ = choiceEntryEnd(data, position, u0)
		position = next_
		index = (index + u1)
	}
	same_start = (outer_start == inner_start)
	if same_start {
		outer_offset = position
	}
	inner_offset = position
	data_end = int64(len(data))
	before_inner = append([]byte(nil), data[i4:inner_offset]...)
	from_outer = append([]byte(nil), data[outer_offset:data_end]...)
	prefix_count = (inner_start - outer_start)
	new_count = (count + prefix_count)
	header = choiceListHeader(new_count)
	with_before = appendBytes(header, before_inner)
	out = appendBytes(with_before, from_outer)
	some = optionalBytes{value: out, ok: true}
	return some.value, some.ok
}

func choiceMutateRanges(data []byte, a_start uint64, a_end uint64, b_start uint64, b_end uint64, use_a bool) ([]byte, bool) {
	var a_before_b bool
	var b_ends_in_a bool
	var a_contains_b bool
	var nested_a optionalBytes
	var b_before_a bool
	var a_ends_in_b bool
	var b_contains_a bool
	var nested_b optionalBytes
	var duplicate_ab optionalBytes
	var use_b bool
	var duplicate_ba optionalBytes
	a_before_b = (a_start <= b_start)
	b_ends_in_a = (b_end <= a_end)
	a_contains_b = (a_before_b && b_ends_in_a)
	if a_contains_b {
		nested_a.value, nested_a.ok = choiceDuplicateNestedPrefix(data, a_start, a_end, b_start, b_end)
		return nested_a.value, nested_a.ok
	}
	b_before_a = (b_start <= a_start)
	a_ends_in_b = (a_end <= b_end)
	b_contains_a = (b_before_a && a_ends_in_b)
	if b_contains_a {
		nested_b.value, nested_b.ok = choiceDuplicateNestedPrefix(data, b_start, b_end, a_start, a_end)
		return nested_b.value, nested_b.ok
	}
	if a_before_b {
		duplicate_ab.value, duplicate_ab.ok = choiceDuplicateRanges(data, a_start, a_end, b_start, b_end, use_a)
		return duplicate_ab.value, duplicate_ab.ok
	}
	use_b = (!use_a)
	duplicate_ba.value, duplicate_ba.ok = choiceDuplicateRanges(data, b_start, b_end, a_start, a_end, use_b)
	return duplicate_ba.value, duplicate_ba.ok
}

func cloneInnerStream(data []byte, offset uint64) ([]byte, bool) {
	var i0 int64
	var i1 int64
	var u0 uint64
	var u5 uint64
	var n int64
	var at int64
	var past_end bool
	var none optionalBytes
	var tag uint64
	var not_clone bool
	var end_ int64
	var malformed bool
	var payload_start int64
	var inner []byte
	var some optionalBytes
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u5 = uint64(5)
	n = int64(len(data))
	at = int64(offset)
	past_end = (at >= n)
	if past_end {
		none = optionalBytes{}
		return none.value, none.ok
	}
	tag = uint64(data[at])
	not_clone = (tag != u5)
	if not_clone {
		none = optionalBytes{}
		return none.value, none.ok
	}
	end_ = choiceEntryEnd(data, at, u0)
	malformed = (end_ == i0)
	if malformed {
		none = optionalBytes{}
		return none.value, none.ok
	}
	payload_start = (at + i1)
	inner = append([]byte(nil), data[payload_start:end_]...)
	some = optionalBytes{value: inner, ok: true}
	return some.value, some.ok
}

func cloneReplaceInner(data []byte, offset uint64, inner []byte) ([]byte, bool) {
	var i0 int64
	var i1 int64
	var u0 uint64
	var u5 uint64
	var n int64
	var at int64
	var past_end bool
	var none optionalBytes
	var tag uint64
	var not_clone bool
	var inner_valid bool
	var inner_invalid bool
	var end_ int64
	var malformed bool
	var prefix_end int64
	var head []byte
	var tail []byte
	var out []byte
	var with_head []byte
	var with_inner []byte
	var rebuilt []byte
	var some optionalBytes
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u5 = uint64(5)
	n = int64(len(data))
	at = int64(offset)
	past_end = (at >= n)
	if past_end {
		none = optionalBytes{}
		return none.value, none.ok
	}
	tag = uint64(data[at])
	not_clone = (tag != u5)
	if not_clone {
		none = optionalBytes{}
		return none.value, none.ok
	}
	inner_valid = choiceStreamValidate(inner)
	inner_invalid = (!inner_valid)
	if inner_invalid {
		none = optionalBytes{}
		return none.value, none.ok
	}
	end_ = choiceEntryEnd(data, at, u0)
	malformed = (end_ == i0)
	if malformed {
		none = optionalBytes{}
		return none.value, none.ok
	}
	prefix_end = (at + i1)
	head = append([]byte(nil), data[i0:prefix_end]...)
	tail = append([]byte(nil), data[end_:n]...)
	out = []byte{}
	with_head = appendBytes(out, head)
	with_inner = appendBytes(with_head, inner)
	rebuilt = appendBytes(with_inner, tail)
	some = optionalBytes{value: rebuilt, ok: true}
	return some.value, some.ok
}

func choiceOffsetAt(data []byte, index uint64) (uint64, bool) {
	var i0 int64
	var i4 int64
	var u0 uint64
	var u1 uint64
	var stream_valid bool
	var stream_invalid bool
	var none optionalU64
	var count uint64
	var past_end bool
	var position int64
	var seen uint64
	var found bool
	var next_ int64
	var malformed bool
	var offset uint64
	var some optionalU64
	i0 = int64(0)
	i4 = int64(4)
	u0 = uint64(0)
	u1 = uint64(1)
	stream_valid = choiceStreamValidate(data)
	stream_invalid = (!stream_valid)
	if stream_invalid {
		none = optionalU64{}
		return none.value, none.ok
	}
	count = choiceStreamCount(data)
	past_end = (index >= count)
	if past_end {
		none = optionalU64{}
		return none.value, none.ok
	}
	position = i4
	seen = u0
	for {
		found = (seen >= index)
		if found {
			break
		}
		next_ = choiceEntryEnd(data, position, u0)
		malformed = (next_ == i0)
		if malformed {
			none = optionalU64{}
			return none.value, none.ok
		}
		position = next_
		seen = (seen + u1)
	}
	offset = uint64(position)
	some = optionalU64{value: offset, ok: true}
	return some.value, some.ok
}

func integerBoundaryEligible(candidate int64, min_value int64, max_value int64) bool {
	var above_min bool
	var below_max bool
	var in_range bool
	var is_min bool
	var is_max bool
	var is_boundary bool
	var not_boundary bool
	var eligible bool
	above_min = (candidate >= min_value)
	below_max = (candidate <= max_value)
	in_range = (above_min && below_max)
	is_min = (candidate == min_value)
	is_max = (candidate == max_value)
	is_boundary = (is_min || is_max)
	not_boundary = (!is_boundary)
	eligible = (in_range && not_boundary)
	return eligible
}

func integerBoundaryCandidate(min_value int64, max_value int64, ordinal uint64) (int64, bool) {
	var i0 int64
	var i1 int64
	var i2 int64
	var u0 uint64
	var u1 uint64
	var u63 uint64
	var want_min bool
	var some_min optionalI64
	var seen uint64
	var same_bounds bool
	var distinct_bounds bool
	var want_max bool
	var some_max optionalI64
	var zero_eligible bool
	var want_zero bool
	var some_zero optionalI64
	var power int64
	var iteration uint64
	var done bool
	var positive_eligible bool
	var want_positive bool
	var some_positive optionalI64
	var negative int64
	var negative_eligible bool
	var want_negative bool
	var some_negative optionalI64
	var has_more bool
	var none optionalI64
	i0 = int64(0)
	i1 = int64(1)
	i2 = int64(2)
	u0 = uint64(0)
	u1 = uint64(1)
	u63 = uint64(63)
	want_min = (ordinal == u0)
	if want_min {
		some_min = optionalI64{value: min_value, ok: true}
		return some_min.value, some_min.ok
	}
	seen = u1
	same_bounds = (min_value == max_value)
	distinct_bounds = (!same_bounds)
	if distinct_bounds {
		want_max = (ordinal == seen)
		if want_max {
			some_max = optionalI64{value: max_value, ok: true}
			return some_max.value, some_max.ok
		}
		seen = (seen + u1)
	}
	zero_eligible = integerBoundaryEligible(i0, min_value, max_value)
	if zero_eligible {
		want_zero = (ordinal == seen)
		if want_zero {
			some_zero = optionalI64{value: i0, ok: true}
			return some_zero.value, some_zero.ok
		}
		seen = (seen + u1)
	}
	power = i1
	iteration = u0
	for {
		done = (iteration >= u63)
		if done {
			break
		}
		positive_eligible = integerBoundaryEligible(power, min_value, max_value)
		if positive_eligible {
			want_positive = (ordinal == seen)
			if want_positive {
				some_positive = optionalI64{value: power, ok: true}
				return some_positive.value, some_positive.ok
			}
			seen = (seen + u1)
		}
		negative = (i0 - power)
		negative_eligible = integerBoundaryEligible(negative, min_value, max_value)
		if negative_eligible {
			want_negative = (ordinal == seen)
			if want_negative {
				some_negative = optionalI64{value: negative, ok: true}
				return some_negative.value, some_negative.ok
			}
			seen = (seen + u1)
		}
		iteration = (iteration + u1)
		has_more = (iteration < u63)
		if has_more {
			power = (power * i2)
		}
	}
	none = optionalI64{}
	return none.value, none.ok
}

func integerSwarmIsWide(min_value int64, max_value int64) bool {
	var min_bits uint64
	var max_bits uint64
	var width uint64
	var u256 uint64
	var wide bool
	min_bits = uint64(min_value)
	max_bits = uint64(max_value)
	width = (max_bits - min_bits)
	u256 = uint64(256)
	wide = (width >= u256)
	return wide
}

func integerSwarmEndpointPermille(entropy uint64) uint64 {
	var mask uint64
	var slot uint64
	var u10 uint64
	var u41 uint64
	var u51 uint64
	var u700 uint64
	var u20 uint64
	var u3 uint64
	var endpoint_heavy bool
	var interesting_heavy bool
	var diffuse_heavy bool
	mask = uint64(255)
	slot = (entropy & mask)
	u10 = uint64(10)
	u41 = uint64(41)
	u51 = uint64(51)
	u700 = uint64(700)
	u20 = uint64(20)
	u3 = uint64(3)
	endpoint_heavy = (slot < u10)
	if endpoint_heavy {
		return u700
	}
	interesting_heavy = (slot < u41)
	if interesting_heavy {
		return u20
	}
	diffuse_heavy = (slot < u51)
	if diffuse_heavy {
		return u20
	}
	return u3
}

func integerSwarmInterestingPermille(entropy uint64) uint64 {
	var mask uint64
	var slot uint64
	var u10 uint64
	var u41 uint64
	var u51 uint64
	var u100 uint64
	var u700 uint64
	var u80 uint64
	var endpoint_heavy bool
	var interesting_heavy bool
	var diffuse_heavy bool
	mask = uint64(255)
	slot = (entropy & mask)
	u10 = uint64(10)
	u41 = uint64(41)
	u51 = uint64(51)
	u100 = uint64(100)
	u700 = uint64(700)
	u80 = uint64(80)
	endpoint_heavy = (slot < u10)
	if endpoint_heavy {
		return u100
	}
	interesting_heavy = (slot < u41)
	if interesting_heavy {
		return u700
	}
	diffuse_heavy = (slot < u51)
	if diffuse_heavy {
		return u100
	}
	return u80
}

func integerSwarmDiffusePermille(entropy uint64) uint64 {
	var mask uint64
	var slot uint64
	var u10 uint64
	var u41 uint64
	var u51 uint64
	var u50 uint64
	var u80 uint64
	var u700 uint64
	var u17 uint64
	var endpoint_heavy bool
	var interesting_heavy bool
	var diffuse_heavy bool
	mask = uint64(255)
	slot = (entropy & mask)
	u10 = uint64(10)
	u41 = uint64(41)
	u51 = uint64(51)
	u50 = uint64(50)
	u80 = uint64(80)
	u700 = uint64(700)
	u17 = uint64(17)
	endpoint_heavy = (slot < u10)
	if endpoint_heavy {
		return u50
	}
	interesting_heavy = (slot < u41)
	if interesting_heavy {
		return u80
	}
	diffuse_heavy = (slot < u51)
	if diffuse_heavy {
		return u700
	}
	return u17
}

func integerSwarmCategory(entropy uint64, roll uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var endpoint uint64
	var in_endpoint bool
	var interesting uint64
	var after_interesting uint64
	var in_interesting bool
	var diffuse uint64
	var after_diffuse uint64
	var in_diffuse bool
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	endpoint = integerSwarmEndpointPermille(entropy)
	in_endpoint = (roll < endpoint)
	if in_endpoint {
		return u0
	}
	interesting = integerSwarmInterestingPermille(entropy)
	after_interesting = (endpoint + interesting)
	in_interesting = (roll < after_interesting)
	if in_interesting {
		return u1
	}
	diffuse = integerSwarmDiffusePermille(entropy)
	after_diffuse = (after_interesting + diffuse)
	in_diffuse = (roll < after_diffuse)
	if in_diffuse {
		return u2
	}
	return u3
}

func integerSwarmEndpointCandidate(min_value int64, max_value int64, ordinal uint64) (int64, bool) {
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var i1 int64
	var is_min bool
	var value optionalI64
	var is_max bool
	var is_min_neighbor bool
	var neighbor int64
	var is_max_neighbor bool
	var none optionalI64
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	i1 = int64(1)
	is_min = (ordinal == u0)
	if is_min {
		value = optionalI64{value: min_value, ok: true}
		return value.value, value.ok
	}
	is_max = (ordinal == u1)
	if is_max {
		value = optionalI64{value: max_value, ok: true}
		return value.value, value.ok
	}
	is_min_neighbor = (ordinal == u2)
	if is_min_neighbor {
		neighbor = (min_value + i1)
		value = optionalI64{value: neighbor, ok: true}
		return value.value, value.ok
	}
	is_max_neighbor = (ordinal == u3)
	if is_max_neighbor {
		neighbor = (max_value - i1)
		value = optionalI64{value: neighbor, ok: true}
		return value.value, value.ok
	}
	none = optionalI64{}
	return none.value, none.ok
}

func integerSwarmDiffuseCount() uint64 {
	var count uint64
	count = uint64(12)
	return count
}

func integerSwarmDiffuseIfInRange(candidate int64, min_value int64, max_value int64) (int64, bool) {
	var above_min bool
	var below_max bool
	var eligible bool
	var value optionalI64
	var none optionalI64
	above_min = (candidate >= min_value)
	below_max = (candidate <= max_value)
	eligible = (above_min && below_max)
	if eligible {
		value = optionalI64{value: candidate, ok: true}
		return value.value, value.ok
	}
	none = optionalI64{}
	return none.value, none.ok
}

func integerSwarmDiffuseCandidate(min_value int64, max_value int64, ordinal uint64) (int64, bool) {
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var u4 uint64
	var u5 uint64
	var u6 uint64
	var u7 uint64
	var u8 uint64
	var u9 uint64
	var u10 uint64
	var u11 uint64
	var i0 int64
	var match_ bool
	var candidate int64
	var value optionalI64
	var m32768 int64
	var m2147483648 int64
	var m1000000 int64
	var m1000000000 int64
	var m6148914691236517205 int64
	var m81985529216486895 int64
	var none optionalI64
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	u4 = uint64(4)
	u5 = uint64(5)
	u6 = uint64(6)
	u7 = uint64(7)
	u8 = uint64(8)
	u9 = uint64(9)
	u10 = uint64(10)
	u11 = uint64(11)
	i0 = int64(0)
	match_ = (ordinal == u0)
	if match_ {
		candidate = int64(32767)
		value.value, value.ok = integerSwarmDiffuseIfInRange(candidate, min_value, max_value)
		return value.value, value.ok
	}
	match_ = (ordinal == u1)
	if match_ {
		m32768 = int64(32768)
		candidate = (i0 - m32768)
		value.value, value.ok = integerSwarmDiffuseIfInRange(candidate, min_value, max_value)
		return value.value, value.ok
	}
	match_ = (ordinal == u2)
	if match_ {
		candidate = int64(2147483647)
		value.value, value.ok = integerSwarmDiffuseIfInRange(candidate, min_value, max_value)
		return value.value, value.ok
	}
	match_ = (ordinal == u3)
	if match_ {
		m2147483648 = int64(2147483648)
		candidate = (i0 - m2147483648)
		value.value, value.ok = integerSwarmDiffuseIfInRange(candidate, min_value, max_value)
		return value.value, value.ok
	}
	match_ = (ordinal == u4)
	if match_ {
		candidate = int64(1000000)
		value.value, value.ok = integerSwarmDiffuseIfInRange(candidate, min_value, max_value)
		return value.value, value.ok
	}
	match_ = (ordinal == u5)
	if match_ {
		m1000000 = int64(1000000)
		candidate = (i0 - m1000000)
		value.value, value.ok = integerSwarmDiffuseIfInRange(candidate, min_value, max_value)
		return value.value, value.ok
	}
	match_ = (ordinal == u6)
	if match_ {
		candidate = int64(1000000000)
		value.value, value.ok = integerSwarmDiffuseIfInRange(candidate, min_value, max_value)
		return value.value, value.ok
	}
	match_ = (ordinal == u7)
	if match_ {
		m1000000000 = int64(1000000000)
		candidate = (i0 - m1000000000)
		value.value, value.ok = integerSwarmDiffuseIfInRange(candidate, min_value, max_value)
		return value.value, value.ok
	}
	match_ = (ordinal == u8)
	if match_ {
		candidate = int64(6148914691236517205)
		value.value, value.ok = integerSwarmDiffuseIfInRange(candidate, min_value, max_value)
		return value.value, value.ok
	}
	match_ = (ordinal == u9)
	if match_ {
		m6148914691236517205 = int64(6148914691236517205)
		candidate = (i0 - m6148914691236517205)
		value.value, value.ok = integerSwarmDiffuseIfInRange(candidate, min_value, max_value)
		return value.value, value.ok
	}
	match_ = (ordinal == u10)
	if match_ {
		candidate = int64(81985529216486895)
		value.value, value.ok = integerSwarmDiffuseIfInRange(candidate, min_value, max_value)
		return value.value, value.ok
	}
	match_ = (ordinal == u11)
	if match_ {
		m81985529216486895 = int64(81985529216486895)
		candidate = (i0 - m81985529216486895)
		value.value, value.ok = integerSwarmDiffuseIfInRange(candidate, min_value, max_value)
		return value.value, value.ok
	}
	none = optionalI64{}
	return none.value, none.ok
}

func integerDefaultShrinkTarget(min_value int64, max_value int64) int64 {
	var i0 int64
	var at_or_below bool
	var at_or_above bool
	var contains_zero bool
	i0 = int64(0)
	at_or_below = (min_value <= i0)
	at_or_above = (max_value >= i0)
	contains_zero = (at_or_below && at_or_above)
	if contains_zero {
		return i0
	}
	return min_value
}

func textWidenedMax(min_size uint64) uint64 {
	var u15 uint64
	var u16 uint64
	var widened uint64
	var below bool
	u15 = uint64(15)
	u16 = uint64(16)
	widened = (min_size + u15)
	below = (widened < u16)
	if below {
		return u16
	}
	return widened
}

func integerBoundaryCount(min_value int64, max_value int64) uint64 {
	var u0 uint64
	var u1 uint64
	var ordinal uint64
	var candidate optionalI64
	var present bool
	var done bool
	u0 = uint64(0)
	u1 = uint64(1)
	ordinal = u0
	for {
		candidate.value, candidate.ok = integerBoundaryCandidate(min_value, max_value, ordinal)
		present = candidate.ok
		done = (!present)
		if done {
			break
		}
		ordinal = (ordinal + u1)
	}
	return ordinal
}

func floatGenerationSpecValid(min_value float64, max_value float64, allow_nan bool, allow_infinity bool, smallest_nonzero float64) bool {
	var false_ bool
	var min_nan bool
	var max_nan bool
	var smallest_nan bool
	var zero float64
	var smallest_negative bool
	var ordered bool
	var simplest optionalF64
	var usable bool
	false_ = false
	min_nan = math.IsNaN(min_value)
	if min_nan {
		return false_
	}
	max_nan = math.IsNaN(max_value)
	if max_nan {
		return false_
	}
	smallest_nan = math.IsNaN(smallest_nonzero)
	if smallest_nan {
		return false_
	}
	zero = float64(0)
	smallest_negative = (smallest_nonzero < zero)
	if smallest_negative {
		return false_
	}
	ordered = signAwareLte(min_value, max_value)
	if ordered {
		simplest.value, simplest.ok = floatSimplest(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero)
		usable = simplest.ok
		return usable
	}
	return false_
}

func floatEntropyBelow(word uint64, count uint64) uint64 {
	var mask uint64
	var u32 uint64
	var word_low uint64
	var word_high uint64
	var count_low uint64
	var count_high uint64
	var low_low uint64
	var low_high uint64
	var high_low uint64
	var high_high uint64
	var carry uint64
	var low_high_low uint64
	var high_low_low uint64
	var middle uint64
	var low_high_high uint64
	var high_low_high uint64
	var middle_high uint64
	var product uint64
	mask = uint64(4294967295)
	u32 = uint64(32)
	word_low = (word & mask)
	word_high = (word >> u32)
	count_low = (count & mask)
	count_high = (count >> u32)
	low_low = (word_low * count_low)
	low_high = (word_low * count_high)
	high_low = (word_high * count_low)
	high_high = (word_high * count_high)
	carry = (low_low >> u32)
	low_high_low = (low_high & mask)
	high_low_low = (high_low & mask)
	middle = (carry + low_high_low)
	middle = (middle + high_low_low)
	low_high_high = (low_high >> u32)
	high_low_high = (high_low >> u32)
	middle_high = (middle >> u32)
	product = (high_high + low_high_high)
	product = (product + high_low_high)
	product = (product + middle_high)
	return product
}

func floatEntropyBetween(word uint64, low uint64, high uint64) uint64 {
	var u1 uint64
	var all uint64
	var span uint64
	var full bool
	var count uint64
	var offset uint64
	var value uint64
	u1 = uint64(1)
	all = uint64(18446744073709551615)
	span = (high - low)
	full = (span == all)
	if full {
		return word
	}
	count = (span + u1)
	offset = floatEntropyBelow(word, count)
	value = (low + offset)
	return value
}

func floatEntropyCoin(word uint64, bit uint64) bool {
	var u1 uint64
	var shifted uint64
	var low uint64
	var set bool
	u1 = uint64(1)
	shifted = (word >> bit)
	low = (shifted & u1)
	set = (low == u1)
	return set
}

func floatEntropyUnit(word uint64) float64 {
	var u11 uint64
	var scale_bits uint64
	var top uint64
	var integer float64
	var scale float64
	var unit float64
	u11 = uint64(11)
	scale_bits = uint64(4368491638549381120)
	top = (word >> u11)
	integer = float64(top)
	scale = math.Float64frombits(scale_bits)
	unit = float64((integer * scale))
	return unit
}

func floatLogUniformMagnitude(low float64, high float64, exponent_word uint64, mantissa_word uint64) float64 {
	var u0 uint64
	var u52 uint64
	var mantissa_mask uint64
	var low_bits uint64
	var high_bits uint64
	var low_exponent uint64
	var high_exponent uint64
	var exponent uint64
	var mantissa_low uint64
	var at_low bool
	var mantissa_high uint64
	var at_high bool
	var mantissa uint64
	var exponent_field uint64
	var bits uint64
	var magnitude float64
	u0 = uint64(0)
	u52 = uint64(52)
	mantissa_mask = uint64(4503599627370495)
	low_bits = math.Float64bits(low)
	high_bits = math.Float64bits(high)
	low_exponent = (low_bits >> u52)
	high_exponent = (high_bits >> u52)
	exponent = floatEntropyBetween(exponent_word, low_exponent, high_exponent)
	mantissa_low = u0
	at_low = (exponent == low_exponent)
	if at_low {
		mantissa_low = (low_bits & mantissa_mask)
	}
	mantissa_high = mantissa_mask
	at_high = (exponent == high_exponent)
	if at_high {
		mantissa_high = (high_bits & mantissa_mask)
	}
	mantissa = floatEntropyBetween(mantissa_word, mantissa_low, mantissa_high)
	exponent_field = (exponent << u52)
	bits = (exponent_field | mantissa)
	magnitude = math.Float64frombits(bits)
	return magnitude
}

func floatBitLength(value uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var rest uint64
	var length uint64
	var done bool
	u0 = uint64(0)
	u1 = uint64(1)
	rest = value
	length = u0
	for {
		done = (rest == u0)
		if done {
			break
		}
		rest = (rest >> u1)
		length = (length + u1)
	}
	return length
}

func floatLogUniformInteger(low uint64, high uint64, binade_word uint64, value_word uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var low_length uint64
	var high_length uint64
	var length uint64
	var first uint64
	var last uint64
	var nonzero bool
	var shift uint64
	var below_first uint64
	var from_ uint64
	var low_above bool
	var to uint64
	var high_below bool
	var value uint64
	u0 = uint64(0)
	u1 = uint64(1)
	low_length = floatBitLength(low)
	high_length = floatBitLength(high)
	length = floatEntropyBetween(binade_word, low_length, high_length)
	first = u0
	last = u0
	nonzero = (length > u0)
	if nonzero {
		shift = (length - u1)
		first = (u1 << shift)
		below_first = (first - u1)
		last = (first + below_first)
	}
	from_ = first
	low_above = (low > from_)
	if low_above {
		from_ = low
	}
	to = last
	high_below = (high < to)
	if high_below {
		to = high
	}
	value = floatEntropyBetween(value_word, from_, to)
	return value
}

func f64NextUp(value float64) float64 {
	var u1 uint64
	var zero float64
	var is_nan bool
	var infinity float64
	var at_top bool
	var is_zero bool
	var smallest float64
	var bits uint64
	var negative bool
	var lower uint64
	var toward_zero float64
	var higher uint64
	var away float64
	u1 = uint64(1)
	zero = float64(0)
	is_nan = math.IsNaN(value)
	if is_nan {
		return value
	}
	infinity = f64Infinity()
	at_top = (value == infinity)
	if at_top {
		return value
	}
	is_zero = (value == zero)
	if is_zero {
		smallest = math.Float64frombits(u1)
		return smallest
	}
	bits = math.Float64bits(value)
	negative = math.Signbit(value)
	if negative {
		lower = (bits - u1)
		toward_zero = math.Float64frombits(lower)
		return toward_zero
	}
	higher = (bits + u1)
	away = math.Float64frombits(higher)
	return away
}

func f64NextDown(value float64) float64 {
	var negated float64
	var up float64
	var down float64
	negated = f64Neg(value)
	up = f64NextUp(negated)
	down = f64Neg(up)
	return down
}

func f64Floor(value float64) float64 {
	var one float64
	var ceiling float64
	var integral bool
	var floor float64
	one = float64(1)
	ceiling = math.Ceil(value)
	integral = (ceiling == value)
	if integral {
		return ceiling
	}
	floor = (ceiling - one)
	return floor
}

func floatCategoryIsPoint(category uint64) bool {
	var u9 uint64
	var u13 uint64
	var from_nan bool
	var to_zero bool
	var point bool
	u9 = uint64(9)
	u13 = uint64(13)
	from_nan = (category >= u9)
	to_zero = (category <= u13)
	point = (from_nan && to_zero)
	return point
}

func floatSwarmWeight(entropy uint64, category uint64) uint64 {
	var u0 uint64
	var u2 uint64
	var u3 uint64
	var u4 uint64
	var u8 uint64
	var u12 uint64
	var u15 uint64
	var u16 uint64
	var u17 uint64
	var u34 uint64
	var u50 uint64
	var u52 uint64
	var u64 uint64
	var u150 uint64
	var u200 uint64
	var u255 uint64
	var u541 uint64
	var u1000 uint64
	var out_of_range bool
	var is_default bool
	var point bool
	var slot uint64
	var ordinary bool
	var focus_field uint64
	var focus uint64
	var level_field uint64
	var level_step uint64
	var level_rise uint64
	var level uint64
	var floor uint64
	var focus_point bool
	var claimed uint64
	var rest uint64
	var is_focus bool
	u0 = uint64(0)
	u2 = uint64(2)
	u3 = uint64(3)
	u4 = uint64(4)
	u8 = uint64(8)
	u12 = uint64(12)
	u15 = uint64(15)
	u16 = uint64(16)
	u17 = uint64(17)
	u34 = uint64(34)
	u50 = uint64(50)
	u52 = uint64(52)
	u64 = uint64(64)
	u150 = uint64(150)
	u200 = uint64(200)
	u255 = uint64(255)
	u541 = uint64(541)
	u1000 = uint64(1000)
	out_of_range = (category > u16)
	if out_of_range {
		return u0
	}
	is_default = (category == u16)
	point = floatCategoryIsPoint(category)
	slot = (entropy & u255)
	ordinary = (slot >= u64)
	if ordinary {
		if is_default {
			return u541
		}
		if point {
			return u17
		}
		return u34
	}
	focus_field = (entropy >> u8)
	focus = (focus_field & u15)
	level_field = (entropy >> u12)
	level_step = (level_field & u3)
	level_rise = (level_step * u150)
	level = (u200 + level_rise)
	if is_default {
		floor = u50
		focus_point = floatCategoryIsPoint(focus)
		if focus_point {
			floor = u52
		}
		claimed = (level + floor)
		rest = (u1000 - claimed)
		return rest
	}
	is_focus = (category == focus)
	if is_focus {
		return level
	}
	if point {
		return u2
	}
	return u4
}

func floatSwarmTotal(entropy uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var u17 uint64
	var total uint64
	var category uint64
	var done bool
	var weight uint64
	u0 = uint64(0)
	u1 = uint64(1)
	u17 = uint64(17)
	total = u0
	category = u0
	for {
		done = (category >= u17)
		if done {
			break
		}
		weight = floatSwarmWeight(entropy, category)
		total = (total + weight)
		category = (category + u1)
	}
	return total
}

func floatSwarmRollSpan() uint64 {
	var span uint64
	span = uint64(1000)
	return span
}

func floatSwarmCategory(entropy uint64, roll uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var u16 uint64
	var reached uint64
	var category uint64
	var done bool
	var weight uint64
	var inside bool
	u0 = uint64(0)
	u1 = uint64(1)
	u16 = uint64(16)
	reached = u0
	category = u0
	for {
		done = (category >= u16)
		if done {
			break
		}
		weight = floatSwarmWeight(entropy, category)
		reached = (reached + weight)
		inside = (roll < reached)
		if inside {
			return category
		}
		category = (category + u1)
	}
	return u16
}

func floatSideLow(min_value float64, max_value float64, smallest_nonzero float64, negative bool) float64 {
	var u1 uint64
	var low float64
	var under_smallest bool
	var tiniest float64
	var under_tiniest bool
	u1 = uint64(1)
	low = min_value
	if negative {
		low = f64Neg(max_value)
	}
	under_smallest = (low < smallest_nonzero)
	if under_smallest {
		low = smallest_nonzero
	}
	tiniest = math.Float64frombits(u1)
	under_tiniest = (low < tiniest)
	if under_tiniest {
		low = tiniest
	}
	return low
}

func floatSideHigh(min_value float64, max_value float64, negative bool) float64 {
	var high float64
	var largest float64
	var over_largest bool
	high = max_value
	if negative {
		high = f64Neg(min_value)
	}
	largest = f64Max()
	over_largest = (high > largest)
	if over_largest {
		high = largest
	}
	return high
}

func floatMagnitudeSample(category uint64, low float64, high float64, coins uint64, first uint64, second uint64) (float64, bool) {
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var u4 uint64
	var u5 uint64
	var u6 uint64
	var u7 uint64
	var u8 uint64
	var u14 uint64
	var u16 uint64
	var u52 uint64
	var mantissa_mask uint64
	var none optionalF64
	var band_low float64
	var band_high float64
	var banded bool
	var near_zero bool
	var bits_low uint64
	var bits_high uint64
	var subnormal bool
	var near_max_for_add bool
	var near_max_for_mul bool
	var near_sqrt_min_positive bool
	var whole_side bool
	var from_ float64
	var band_above bool
	var to float64
	var band_below bool
	var empty bool
	var magnitude float64
	var found optionalF64
	var low_bits uint64
	var high_bits uint64
	var near_one bool
	var one_bits uint64
	var mantissa_bits uint64
	var closeness uint64
	var span_shift uint64
	var span uint64
	var above_from uint64
	var low_over_one bool
	var above_end uint64
	var above_to uint64
	var high_under_above bool
	var double_span uint64
	var below_reach uint64
	var below_from uint64
	var low_over_below bool
	var below_to uint64
	var high_under_below bool
	var above_open bool
	var below_open bool
	var below_first bool
	var take_below bool
	var above_first bool
	var prefer_above bool
	var above_only bool
	var bits uint64
	var value float64
	var integer bool
	var one float64
	var limit_bits uint64
	var limit float64
	var under_one bool
	var over_limit bool
	var from_integer uint64
	var to_integer uint64
	var picked uint64
	var half_integer bool
	var zero float64
	var half float64
	var low_less_half float64
	var under_zero bool
	var high_less_half float64
	var whole float64
	var binade_edge bool
	var low_exponent uint64
	var low_mantissa uint64
	var from_exponent uint64
	var subnormal_low bool
	var inside_binade bool
	var past_power bool
	var to_exponent uint64
	var exponent uint64
	var power uint64
	var predecessor uint64
	var want_predecessor bool
	var predecessor_in_range bool
	var use_predecessor bool
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	u4 = uint64(4)
	u5 = uint64(5)
	u6 = uint64(6)
	u7 = uint64(7)
	u8 = uint64(8)
	u14 = uint64(14)
	u16 = uint64(16)
	u52 = uint64(52)
	mantissa_mask = uint64(4503599627370495)
	none = optionalF64{}
	band_low = low
	band_high = high
	banded = false
	near_zero = (category == u1)
	if near_zero {
		bits_low = uint64(4503599627370496)
		bits_high = uint64(4591870180066957721)
		band_low = math.Float64frombits(bits_low)
		band_high = math.Float64frombits(bits_high)
		banded = true
	}
	subnormal = (category == u2)
	if subnormal {
		band_low = math.Float64frombits(u1)
		band_high = math.Float64frombits(mantissa_mask)
		banded = true
	}
	near_max_for_add = (category == u6)
	if near_max_for_add {
		bits_low = uint64(9214364837600034816)
		band_low = math.Float64frombits(bits_low)
		band_high = f64Max()
		banded = true
	}
	near_max_for_mul = (category == u7)
	if near_max_for_mul {
		bits_low = uint64(6908521828386340864)
		bits_high = uint64(6917529027641081855)
		band_low = math.Float64frombits(bits_low)
		band_high = math.Float64frombits(bits_high)
		banded = true
	}
	near_sqrt_min_positive = (category == u8)
	if near_sqrt_min_positive {
		bits_low = uint64(2296835809958952960)
		bits_high = uint64(2310346608841064447)
		band_low = math.Float64frombits(bits_low)
		band_high = math.Float64frombits(bits_high)
		banded = true
	}
	whole_side = (category == u16)
	if whole_side {
		banded = true
	}
	if banded {
		from_ = low
		band_above = (band_low > from_)
		if band_above {
			from_ = band_low
		}
		to = high
		band_below = (band_high < to)
		if band_below {
			to = band_high
		}
		empty = (from_ > to)
		if empty {
			return none.value, none.ok
		}
		magnitude = floatLogUniformMagnitude(from_, to, first, second)
		found = optionalF64{value: magnitude, ok: true}
		return found.value, found.ok
	}
	low_bits = math.Float64bits(low)
	high_bits = math.Float64bits(high)
	near_one = (category == u3)
	if near_one {
		one_bits = uint64(4607182418800017408)
		mantissa_bits = uint64(52)
		closeness = floatEntropyBetween(first, u1, mantissa_bits)
		span_shift = (mantissa_bits - closeness)
		span = (u1 << span_shift)
		above_from = one_bits
		low_over_one = (low_bits > above_from)
		if low_over_one {
			above_from = low_bits
		}
		above_end = (one_bits + span)
		above_to = (above_end - u1)
		high_under_above = (high_bits < above_to)
		if high_under_above {
			above_to = high_bits
		}
		double_span = (span + span)
		below_reach = (double_span - u1)
		below_from = (one_bits - below_reach)
		low_over_below = (low_bits > below_from)
		if low_over_below {
			below_from = low_bits
		}
		below_to = (one_bits - u1)
		high_under_below = (high_bits < below_to)
		if high_under_below {
			below_to = high_bits
		}
		above_open = (above_from <= above_to)
		below_open = (below_from <= below_to)
		below_first = floatEntropyCoin(coins, u1)
		take_below = below_open
		above_first = (!below_first)
		prefer_above = (above_first && above_open)
		if prefer_above {
			take_below = false
		}
		above_only = (!below_open)
		if above_only {
			take_below = false
		}
		if take_below {
			bits = floatEntropyBetween(second, below_from, below_to)
			value = math.Float64frombits(bits)
			found = optionalF64{value: value, ok: true}
			return found.value, found.ok
		}
		if above_open {
			bits = floatEntropyBetween(second, above_from, above_to)
			value = math.Float64frombits(bits)
			found = optionalF64{value: value, ok: true}
			return found.value, found.ok
		}
		return none.value, none.ok
	}
	integer = (category == u4)
	if integer {
		one = float64(1)
		limit_bits = uint64(4845873199050653696)
		limit = math.Float64frombits(limit_bits)
		from_ = math.Ceil(low)
		under_one = (from_ < one)
		if under_one {
			from_ = one
		}
		to = f64Floor(high)
		over_limit = (to > limit)
		if over_limit {
			to = limit
		}
		empty = (from_ > to)
		if empty {
			return none.value, none.ok
		}
		from_integer = uint64(from_)
		to_integer = uint64(to)
		picked = floatLogUniformInteger(from_integer, to_integer, first, second)
		value = float64(picked)
		found = optionalF64{value: value, ok: true}
		return found.value, found.ok
	}
	half_integer = (category == u5)
	if half_integer {
		zero = float64(0)
		half = float64(0.5)
		limit_bits = uint64(4841369599423283198)
		limit = math.Float64frombits(limit_bits)
		low_less_half = (low - half)
		from_ = math.Ceil(low_less_half)
		under_zero = (from_ < zero)
		if under_zero {
			from_ = zero
		}
		high_less_half = (high - half)
		to = f64Floor(high_less_half)
		over_limit = (to > limit)
		if over_limit {
			to = limit
		}
		empty = (from_ > to)
		if empty {
			return none.value, none.ok
		}
		from_integer = uint64(from_)
		to_integer = uint64(to)
		picked = floatLogUniformInteger(from_integer, to_integer, first, second)
		whole = float64(picked)
		value = (whole + half)
		found = optionalF64{value: value, ok: true}
		return found.value, found.ok
	}
	binade_edge = (category == u14)
	if binade_edge {
		low_exponent = (low_bits >> u52)
		low_mantissa = (low_bits & mantissa_mask)
		from_exponent = low_exponent
		subnormal_low = (low_exponent == u0)
		inside_binade = (low_mantissa != u0)
		past_power = (subnormal_low || inside_binade)
		if past_power {
			from_exponent = (low_exponent + u1)
		}
		to_exponent = (high_bits >> u52)
		empty = (from_exponent > to_exponent)
		if empty {
			return none.value, none.ok
		}
		exponent = floatEntropyBetween(first, from_exponent, to_exponent)
		power = (exponent << u52)
		bits = power
		predecessor = (power - u1)
		want_predecessor = floatEntropyCoin(coins, u1)
		predecessor_in_range = (predecessor >= low_bits)
		use_predecessor = (want_predecessor && predecessor_in_range)
		if use_predecessor {
			bits = predecessor
		}
		value = math.Float64frombits(bits)
		found = optionalF64{value: value, ok: true}
		return found.value, found.ok
	}
	return none.value, none.ok
}

func floatSignedSample(category uint64, min_value float64, max_value float64, smallest_nonzero float64, coins uint64, first uint64, second uint64) (float64, bool) {
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var negative bool
	var tried uint64
	var done bool
	var low float64
	var high float64
	var open bool
	var magnitude optionalF64
	var present bool
	var value float64
	var found optionalF64
	var none optionalF64
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	negative = floatEntropyCoin(coins, u0)
	tried = u0
	for {
		done = (tried >= u2)
		if done {
			break
		}
		low = floatSideLow(min_value, max_value, smallest_nonzero, negative)
		high = floatSideHigh(min_value, max_value, negative)
		open = (low <= high)
		if open {
			magnitude.value, magnitude.ok = floatMagnitudeSample(category, low, high, coins, first, second)
			present = magnitude.ok
			if present {
				value = magnitude.value
				if negative {
					value = f64Neg(value)
				}
				found = optionalF64{value: value, ok: true}
				return found.value, found.ok
			}
		}
		negative = (!negative)
		tried = (tried + u1)
	}
	none = optionalF64{}
	return none.value, none.ok
}

func floatPointOptionCount(category uint64) uint64 {
	var u0 uint64
	var u2 uint64
	var u6 uint64
	var u10 uint64
	var u13 uint64
	var endpoint bool
	var from_infinity bool
	var to_zero bool
	var paired bool
	u0 = uint64(0)
	u2 = uint64(2)
	u6 = uint64(6)
	u10 = uint64(10)
	u13 = uint64(13)
	endpoint = (category == u0)
	if endpoint {
		return u6
	}
	from_infinity = (category >= u10)
	to_zero = (category <= u13)
	paired = (from_infinity && to_zero)
	if paired {
		return u2
	}
	return u0
}

func floatPointOption(category uint64, min_value float64, max_value float64, index uint64) float64 {
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var u4 uint64
	var u10 uint64
	var u11 uint64
	var u12 uint64
	var one float64
	var endpoint bool
	var want_min bool
	var want_max bool
	var want_above_min bool
	var above_min float64
	var want_min_plus_one bool
	var min_plus_one float64
	var want_max_less_one bool
	var max_less_one float64
	var below_max float64
	var magnitude float64
	var infinity bool
	var max_magnitude bool
	var max_exact_integer bool
	var limit_bits uint64
	var negative bool
	var negated float64
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	u4 = uint64(4)
	u10 = uint64(10)
	u11 = uint64(11)
	u12 = uint64(12)
	one = float64(1)
	endpoint = (category == u0)
	if endpoint {
		want_min = (index == u0)
		if want_min {
			return min_value
		}
		want_max = (index == u1)
		if want_max {
			return max_value
		}
		want_above_min = (index == u2)
		if want_above_min {
			above_min = f64NextUp(min_value)
			return above_min
		}
		want_min_plus_one = (index == u3)
		if want_min_plus_one {
			min_plus_one = (min_value + one)
			return min_plus_one
		}
		want_max_less_one = (index == u4)
		if want_max_less_one {
			max_less_one = (max_value - one)
			return max_less_one
		}
		below_max = f64NextDown(max_value)
		return below_max
	}
	magnitude = float64(0)
	infinity = (category == u10)
	if infinity {
		magnitude = f64Infinity()
	}
	max_magnitude = (category == u11)
	if max_magnitude {
		magnitude = f64Max()
	}
	max_exact_integer = (category == u12)
	if max_exact_integer {
		limit_bits = uint64(4845873199050653696)
		magnitude = math.Float64frombits(limit_bits)
	}
	negative = (index == u1)
	if negative {
		negated = f64Neg(magnitude)
		return negated
	}
	return magnitude
}

func floatPointOptionAdmitted(category uint64, min_value float64, max_value float64, allow_nan bool, allow_infinity bool, smallest_nonzero float64, index uint64) bool {
	var u0 uint64
	var u1 uint64
	var false_ bool
	var option float64
	var valid bool
	var invalid bool
	var option_bits uint64
	var earlier uint64
	var done bool
	var previous float64
	var previous_bits uint64
	var repeat bool
	var true_ bool
	u0 = uint64(0)
	u1 = uint64(1)
	false_ = false
	option = floatPointOption(category, min_value, max_value, index)
	valid = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, option)
	invalid = (!valid)
	if invalid {
		return false_
	}
	option_bits = math.Float64bits(option)
	earlier = u0
	for {
		done = (earlier >= index)
		if done {
			break
		}
		previous = floatPointOption(category, min_value, max_value, earlier)
		previous_bits = math.Float64bits(previous)
		repeat = (previous_bits == option_bits)
		if repeat {
			return false_
		}
		earlier = (earlier + u1)
	}
	true_ = true
	return true_
}

func floatPointCandidate(category uint64, min_value float64, max_value float64, allow_nan bool, allow_infinity bool, smallest_nonzero float64, word uint64) (float64, bool) {
	var u0 uint64
	var u1 uint64
	var count uint64
	var admitted uint64
	var index uint64
	var done bool
	var admit bool
	var none optionalF64
	var nothing bool
	var wanted uint64
	var seen uint64
	var hit bool
	var option float64
	var found optionalF64
	u0 = uint64(0)
	u1 = uint64(1)
	count = floatPointOptionCount(category)
	admitted = u0
	index = u0
	for {
		done = (index >= count)
		if done {
			break
		}
		admit = floatPointOptionAdmitted(category, min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, index)
		if admit {
			admitted = (admitted + u1)
		}
		index = (index + u1)
	}
	none = optionalF64{}
	nothing = (admitted == u0)
	if nothing {
		return none.value, none.ok
	}
	wanted = floatEntropyBelow(word, admitted)
	seen = u0
	index = u0
	for {
		done = (index >= count)
		if done {
			break
		}
		admit = floatPointOptionAdmitted(category, min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, index)
		if admit {
			hit = (seen == wanted)
			if hit {
				option = floatPointOption(category, min_value, max_value, index)
				found = optionalF64{value: option, ok: true}
				return found.value, found.ok
			}
			seen = (seen + u1)
		}
		index = (index + u1)
	}
	return none.value, none.ok
}

func floatNanCandidate(allow_nan bool, coins uint64, first uint64) (float64, bool) {
	var u1 uint64
	var u63 uint64
	var exponent_field uint64
	var quiet uint64
	var refused bool
	var none optionalF64
	var sign_bit uint64
	var sign uint64
	var payload uint64
	var canonical bool
	var random bool
	var payload_top uint64
	var low_payload uint64
	var bits uint64
	var value float64
	var found optionalF64
	u1 = uint64(1)
	u63 = uint64(63)
	exponent_field = uint64(9218868437227405312)
	quiet = uint64(2251799813685248)
	refused = (!allow_nan)
	if refused {
		none = optionalF64{}
		return none.value, none.ok
	}
	sign_bit = (coins & u1)
	sign = (sign_bit << u63)
	payload = quiet
	canonical = floatEntropyCoin(coins, u1)
	random = (!canonical)
	if random {
		payload_top = (quiet - u1)
		low_payload = floatEntropyBetween(first, u1, payload_top)
		payload = (quiet | low_payload)
	}
	bits = (sign | exponent_field)
	bits = (bits | payload)
	value = math.Float64frombits(bits)
	found = optionalF64{value: value, ok: true}
	return found.value, found.ok
}

func floatOddDenominator(word uint64) uint64 {
	var u1 uint64
	var u500 uint64
	var half uint64
	var double_ uint64
	var odd uint64
	u1 = uint64(1)
	u500 = uint64(500)
	half = floatEntropyBetween(word, u1, u500)
	double_ = (half + half)
	odd = (double_ + u1)
	return odd
}

func floatDivides(numerator float64, denominator float64) bool {
	var quotient float64
	var ceiling float64
	var integral bool
	quotient = (numerator / denominator)
	ceiling = math.Ceil(quotient)
	integral = (ceiling == quotient)
	return integral
}

func floatNonDyadicCandidate(min_value float64, max_value float64, allow_nan bool, allow_infinity bool, smallest_nonzero float64, coins uint64, first uint64, second uint64, third uint64, fourth uint64) (float64, bool) {
	var u0 uint64
	var u1 uint64
	var u8 uint64
	var u10 uint64
	var u100 uint64
	var u300 uint64
	var five float64
	var one float64
	var magnitude float64
	var decimal bool
	var digits uint64
	var scale uint64
	var raised uint64
	var done bool
	var numerator_top uint64
	var numerator_integer uint64
	var numerator float64
	var five_divides bool
	var denominator float64
	var fraction bool
	var denominator_integer uint64
	var whole bool
	var negated float64
	var first_value float64
	var second_value float64
	var negative_first bool
	var first_valid bool
	var found optionalF64
	var second_valid bool
	var none optionalF64
	var size float64
	var size_nan bool
	var size_infinite bool
	var ratio float64
	var offset float64
	var value float64
	u0 = uint64(0)
	u1 = uint64(1)
	u8 = uint64(8)
	u10 = uint64(10)
	u100 = uint64(100)
	u300 = uint64(300)
	five = float64(5)
	one = float64(1)
	magnitude = float64(0)
	decimal = floatEntropyCoin(coins, u1)
	if decimal {
		digits = floatEntropyBetween(first, u1, u8)
		scale = u1
		raised = u0
		for {
			done = (raised >= digits)
			if done {
				break
			}
			scale = (scale * u10)
			raised = (raised + u1)
		}
		numerator_top = (scale * u100)
		numerator_integer = floatEntropyBetween(second, u1, numerator_top)
		numerator = float64(numerator_integer)
		five_divides = floatDivides(numerator, five)
		if five_divides {
			numerator = (numerator + one)
		}
		denominator = float64(scale)
		magnitude = (numerator / denominator)
	}
	fraction = (!decimal)
	if fraction {
		denominator_integer = floatOddDenominator(first)
		numerator_integer = floatEntropyBetween(second, u1, u300)
		numerator = float64(numerator_integer)
		denominator = float64(denominator_integer)
		whole = floatDivides(numerator, denominator)
		if whole {
			numerator = (numerator + one)
		}
		magnitude = (numerator / denominator)
	}
	negated = f64Neg(magnitude)
	first_value = magnitude
	second_value = negated
	negative_first = floatEntropyCoin(coins, u0)
	if negative_first {
		first_value = negated
		second_value = magnitude
	}
	first_valid = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, first_value)
	if first_valid {
		found = optionalF64{value: first_value, ok: true}
		return found.value, found.ok
	}
	second_valid = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, second_value)
	if second_valid {
		found = optionalF64{value: second_value, ok: true}
		return found.value, found.ok
	}
	none = optionalF64{}
	size = (max_value - min_value)
	size_nan = math.IsNaN(size)
	if size_nan {
		return none.value, none.ok
	}
	size_infinite = math.IsInf(size, 0)
	if size_infinite {
		return none.value, none.ok
	}
	denominator_integer = floatOddDenominator(third)
	numerator_top = (denominator_integer - u1)
	numerator_integer = floatEntropyBetween(fourth, u1, numerator_top)
	numerator = float64(numerator_integer)
	denominator = float64(denominator_integer)
	ratio = (numerator / denominator)
	offset = float64((size * ratio))
	value = (min_value + offset)
	found = optionalF64{value: value, ok: true}
	return found.value, found.ok
}

func floatCategoryCandidate(category uint64, min_value float64, max_value float64, allow_nan bool, allow_infinity bool, smallest_nonzero float64, coins uint64, first uint64, second uint64, third uint64, fourth uint64) (float64, bool) {
	var u0 uint64
	var u8 uint64
	var u9 uint64
	var u14 uint64
	var u15 uint64
	var candidate optionalF64
	var options uint64
	var pointed bool
	var not_endpoint bool
	var through_bands bool
	var band bool
	var binade_edge bool
	var signed_ bool
	var nan bool
	var non_dyadic bool
	var present bool
	var value float64
	var valid bool
	var none optionalF64
	u0 = uint64(0)
	u8 = uint64(8)
	u9 = uint64(9)
	u14 = uint64(14)
	u15 = uint64(15)
	candidate = optionalF64{}
	options = floatPointOptionCount(category)
	pointed = (options > u0)
	if pointed {
		candidate.value, candidate.ok = floatPointCandidate(category, min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, first)
	}
	not_endpoint = (category > u0)
	through_bands = (category <= u8)
	band = (not_endpoint && through_bands)
	binade_edge = (category == u14)
	signed_ = (band || binade_edge)
	if signed_ {
		candidate.value, candidate.ok = floatSignedSample(category, min_value, max_value, smallest_nonzero, coins, first, second)
	}
	nan = (category == u9)
	if nan {
		candidate.value, candidate.ok = floatNanCandidate(allow_nan, coins, first)
	}
	non_dyadic = (category == u15)
	if non_dyadic {
		candidate.value, candidate.ok = floatNonDyadicCandidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, coins, first, second, third, fourth)
	}
	present = candidate.ok
	if present {
		value = candidate.value
		valid = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, value)
		if valid {
			return candidate.value, candidate.ok
		}
	}
	none = optionalF64{}
	return none.value, none.ok
}

func floatDefaultCandidate(min_value float64, max_value float64, allow_nan bool, allow_infinity bool, smallest_nonzero float64, coins uint64, first uint64, second uint64, third uint64) float64 {
	var u0 uint64
	var u1 uint64
	var u16 uint64
	var one float64
	var log_uniform bool
	var sign_coins uint64
	var sampled optionalF64
	var present bool
	var value float64
	var valid bool
	var largest float64
	var lowest float64
	var low float64
	var under bool
	var high float64
	var over bool
	var unit float64
	var rest float64
	var low_share float64
	var high_share float64
	var simplest optionalF64
	var has_simplest bool
	var fallback float64
	u0 = uint64(0)
	u1 = uint64(1)
	u16 = uint64(16)
	one = float64(1)
	log_uniform = floatEntropyCoin(coins, u0)
	if log_uniform {
		sign_coins = (coins >> u1)
		sampled.value, sampled.ok = floatSignedSample(u16, min_value, max_value, smallest_nonzero, sign_coins, first, second)
		present = sampled.ok
		if present {
			value = sampled.value
			valid = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, value)
			if valid {
				return value
			}
		}
	}
	largest = f64Max()
	lowest = f64Neg(largest)
	low = min_value
	under = (low < lowest)
	if under {
		low = lowest
	}
	high = max_value
	over = (high > largest)
	if over {
		high = largest
	}
	unit = floatEntropyUnit(third)
	rest = (one - unit)
	low_share = float64((low * rest))
	high_share = float64((high * unit))
	value = (low_share + high_share)
	valid = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, value)
	if valid {
		return value
	}
	simplest.value, simplest.ok = floatSimplest(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero)
	has_simplest = simplest.ok
	if has_simplest {
		fallback = simplest.value
		return fallback
	}
	return value
}

func textBiasedRankSpan() uint64 {
	var span uint64
	span = uint64(128)
	return span
}

func textBiasThreshold() uint64 {
	var threshold uint64
	threshold = uint64(900)
	return threshold
}

func textRankSpan(alphabet []byte, biased bool) uint64 {
	var size uint64
	var span uint64
	var biased_span uint64
	var narrower bool
	size = alphabetSize(alphabet)
	span = size
	if biased {
		biased_span = textBiasedRankSpan()
		narrower = (biased_span < size)
		if narrower {
			span = biased_span
		}
	}
	return span
}

func generationRequestKind(attempts uint64, pending_mutations uint64, mutation_streak uint64) uint64 {
	var u0 uint64
	var simplest uint64
	var mutation uint64
	var random uint64
	var first bool
	var has_mutation bool
	var brood uint64
	var fresh_due bool
	u0 = uint64(0)
	simplest = uint64(1)
	mutation = uint64(2)
	random = uint64(3)
	first = (attempts == u0)
	if first {
		return simplest
	}
	has_mutation = (pending_mutations > u0)
	if has_mutation {
		brood = mutationAttemptCount()
		fresh_due = (mutation_streak >= brood)
		if fresh_due {
			return random
		}
		return mutation
	}
	return random
}

func generationEnqueueMutations(request_kind uint64, valid uint64) bool {
	var random uint64
	var u10 uint64
	var is_random bool
	var corpus_ready bool
	var enqueue bool
	random = uint64(3)
	u10 = uint64(10)
	is_random = (request_kind == random)
	corpus_ready = (valid >= u10)
	enqueue = (is_random && corpus_ready)
	return enqueue
}

func generationShouldContinue(valid uint64, examples uint64) bool {
	var continue_ bool
	continue_ = (valid < examples)
	return continue_
}

func generationNoveltyAction(request_kind uint64, exact_seen bool) uint64 {
	var skip uint64
	var execute uint64
	var execute_and_record uint64
	var mutation uint64
	var is_mutation bool
	skip = uint64(0)
	execute = uint64(1)
	execute_and_record = uint64(2)
	mutation = uint64(2)
	is_mutation = (request_kind == mutation)
	if is_mutation {
		if exact_seen {
			return skip
		}
		return execute_and_record
	}
	return execute
}

func generationRareLabelPermille() uint64 {
	var permille uint64
	permille = uint64(250)
	return permille
}

func generationLabelRare(observed uint64, valid uint64, min_permille uint64) bool {
	var true_ bool
	var false_ bool
	var thousand uint64
	var max_scaled uint64
	var floor uint64
	var threshold uint64
	var below_floor bool
	var over_rate bool
	var short_ bool
	var lhs_overflows bool
	var rhs_overflows bool
	var lhs uint64
	var rhs uint64
	var rare bool
	true_ = true
	false_ = false
	thousand = uint64(1000)
	max_scaled = uint64(18446744073709551)
	floor = generationRareLabelPermille()
	threshold = min_permille
	below_floor = (threshold < floor)
	if below_floor {
		threshold = floor
	}
	over_rate = (threshold > thousand)
	if over_rate {
		short_ = (observed < valid)
		return short_
	}
	lhs_overflows = (observed > max_scaled)
	if lhs_overflows {
		return false_
	}
	rhs_overflows = (valid > max_scaled)
	if rhs_overflows {
		return true_
	}
	lhs = (observed * thousand)
	rhs = (valid * threshold)
	rare = (lhs < rhs)
	return rare
}

func generationMutationAdmission(request_kind uint64, valid uint64, pending uint64, held_rare_label bool) uint64 {
	var none uint64
	var back uint64
	var front uint64
	var brood uint64
	var full bool
	var base bool
	none = uint64(0)
	back = uint64(1)
	front = uint64(2)
	brood = mutationAttemptCount()
	full = (pending >= brood)
	if full {
		return none
	}
	base = generationEnqueueMutations(request_kind, valid)
	if base {
		if held_rare_label {
			return front
		}
		return back
	}
	if held_rare_label {
		return front
	}
	return none
}

func mutationIndex(count uint64, entropy uint64) (uint64, bool) {
	var u0 uint64
	var u32 uint64
	var u4294967296 uint64
	var empty bool
	var too_large bool
	var invalid bool
	var none optionalU64
	var slot uint64
	var scaled uint64
	var index uint64
	var some optionalU64
	u0 = uint64(0)
	u32 = uint64(32)
	u4294967296 = uint64(4294967296)
	empty = (count == u0)
	too_large = (count > u4294967296)
	invalid = (empty || too_large)
	if invalid {
		none = optionalU64{}
		return none.value, none.ok
	}
	slot = (entropy >> u32)
	scaled = (slot * count)
	index = (scaled >> u32)
	some = optionalU64{value: index, ok: true}
	return some.value, some.ok
}

func mutationDistinctIndex(count uint64, first uint64, entropy uint64) (uint64, bool) {
	var u1 uint64
	var too_small bool
	var first_invalid bool
	var invalid bool
	var none optionalU64
	var remaining uint64
	var raw_opt optionalU64
	var raw uint64
	var past_hole bool
	var index uint64
	var some optionalU64
	u1 = uint64(1)
	too_small = (count <= u1)
	first_invalid = (first >= count)
	invalid = (too_small || first_invalid)
	if invalid {
		none = optionalU64{}
		return none.value, none.ok
	}
	remaining = (count - u1)
	raw_opt.value, raw_opt.ok = mutationIndex(remaining, entropy)
	raw = raw_opt.value
	past_hole = (raw >= first)
	index = raw
	if past_hole {
		index = (raw + u1)
	}
	some = optionalU64{value: index, ok: true}
	return some.value, some.ok
}

func mutationUseFirst(entropy uint64) bool {
	var u0 uint64
	var u1 uint64
	var low uint64
	var first bool
	u0 = uint64(0)
	u1 = uint64(1)
	low = (entropy & u1)
	first = (low == u0)
	return first
}

func mutationAttemptCount() uint64 {
	var u6 uint64
	u6 = uint64(6)
	return u6
}

func mutationAttemptRandomized(attempt uint64) bool {
	var u2 uint64
	var randomized bool
	u2 = uint64(2)
	randomized = (attempt >= u2)
	return randomized
}

func statefulFeatureGenerationRaw() uint64 {
	var u0 uint64
	u0 = uint64(0)
	return u0
}

func statefulFeatureProbability(raw uint64) float64 {
	var u254 uint64
	var f255 float64
	var bounded uint64
	var too_large bool
	var value float64
	var probability float64
	u254 = uint64(254)
	f255 = float64(255)
	bounded = raw
	too_large = (bounded > u254)
	if too_large {
		bounded = u254
	}
	value = float64(bounded)
	probability = (value / f255)
	return probability
}

func statefulFeatureForced(candidate_count uint64, is_candidate bool, decided bool, disabled bool) (bool, bool) {
	var u1 uint64
	var false_ bool
	var last bool
	var must_enable bool
	var forced optionalBool
	var free optionalBool
	u1 = uint64(1)
	false_ = false
	last = (candidate_count == u1)
	must_enable = (last && is_candidate)
	if must_enable {
		forced = optionalBool{value: false_, ok: true}
		return forced.value, forced.ok
	}
	if decided {
		forced = optionalBool{value: disabled, ok: true}
		return forced.value, forced.ok
	}
	free = optionalBool{}
	return free.value, free.ok
}

func statefulRuleProbeCount() uint64 {
	var u3 uint64
	u3 = uint64(3)
	return u3
}

func statefulFallbackMaxGood(rule_count uint64, known_bad_count uint64) uint64 {
	var u0 uint64
	var valid bool
	var remaining uint64
	u0 = uint64(0)
	valid = (known_bad_count < rule_count)
	if valid {
		remaining = (rule_count - known_bad_count)
		return remaining
	}
	return u0
}

func statefulFallbackAccept(allowed_count uint64, speculative uint64) bool {
	var accept bool
	accept = (allowed_count > speculative)
	return accept
}

func statefulChoiceProtocol(data []byte, start int64, remaining uint64) uint64 {
	var i0 int64
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var first int64
	var third int64
	var seen uint64
	var pos int64
	var left uint64
	var enough bool
	var exhausted bool
	var end_ int64
	var bad_entry bool
	var key optionalU64
	var named bool
	var positional bool
	var is_first bool
	var is_third bool
	var first_choice optionalBool
	var first_boolean bool
	var first_value bool
	var tag uint64
	var integer bool
	i0 = int64(0)
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	first = i0
	third = i0
	seen = u0
	pos = start
	left = remaining
	for {
		enough = (seen >= u3)
		if enough {
			break
		}
		exhausted = (left == u0)
		if exhausted {
			return u1
		}
		end_ = choiceEntryEnd(data, pos, u0)
		bad_entry = (end_ == i0)
		if bad_entry {
			return u1
		}
		key.value, key.ok = keyedEntryKey(data, pos)
		named = key.ok
		positional = (!named)
		if positional {
			is_first = (seen == u0)
			if is_first {
				first = pos
			}
			is_third = (seen == u2)
			if is_third {
				third = pos
			}
			seen = (seen + u1)
		}
		pos = end_
		left = (left - u1)
	}
	first_choice.value, first_choice.ok = deserializeBooleanChoice(data, first)
	first_boolean = first_choice.ok
	if first_boolean {
		first_value = first_choice.value
		if first_value {
			return u3
		}
	}
	tag = uint64(data[third])
	integer = (tag == u0)
	if integer {
		return u2
	}
	return u1
}

func statefulStopForced(steps_drawn uint64, steps_rejected uint64, step_count uint64) (bool, bool) {
	var u0 uint64
	var u10 uint64
	var u1000 uint64
	var u_max uint64
	var mul_ceiling uint64
	var true_ bool
	var false_ bool
	var successful uint64
	var progressed bool
	var done bool
	var halt optionalBool
	var attempts uint64
	var fits bool
	var stalled bool
	var below_floor bool
	var spent bool
	var first bool
	var go_ optionalBool
	var free optionalBool
	u0 = uint64(0)
	u10 = uint64(10)
	u1000 = uint64(1000)
	u_max = uint64(18446744073709551615)
	mul_ceiling = uint64(1844674407370955161)
	true_ = true
	false_ = false
	successful = u0
	progressed = (steps_drawn > steps_rejected)
	if progressed {
		successful = (steps_drawn - steps_rejected)
	}
	done = (successful >= step_count)
	if done {
		halt = optionalBool{value: true_, ok: true}
		return halt.value, halt.ok
	}
	attempts = u_max
	fits = (step_count <= mul_ceiling)
	if fits {
		attempts = (step_count * u10)
	}
	stalled = (successful == u0)
	if stalled {
		below_floor = (attempts < u1000)
		if below_floor {
			attempts = u1000
		}
	}
	spent = (steps_drawn >= attempts)
	if spent {
		halt = optionalBool{value: true_, ok: true}
		return halt.value, halt.ok
	}
	first = (steps_drawn == u0)
	if first {
		go_ = optionalBool{value: false_, ok: true}
		return go_.value, go_.ok
	}
	free = optionalBool{}
	return free.value, free.ok
}

func statefulContinueForced(steps_drawn uint64, steps_rejected uint64, step_count uint64) (bool, bool) {
	var stop optionalBool
	var decided bool
	var halt bool
	var go_ bool
	var forced optionalBool
	var free optionalBool
	stop.value, stop.ok = statefulStopForced(steps_drawn, steps_rejected, step_count)
	decided = stop.ok
	if decided {
		halt = stop.value
		go_ = (!halt)
		forced = optionalBool{value: go_, ok: true}
		return forced.value, forced.ok
	}
	free = optionalBool{}
	return free.value, free.ok
}

func statefulContinueProbability() float64 {
	var f1 float64
	var f2_32 float64
	var stop float64
	var probability float64
	f1 = float64(1)
	f2_32 = float64(4294967296)
	stop = (f1 / f2_32)
	probability = (f1 - stop)
	return probability
}

func statefulPoolPosition(active_count uint64, newest_offset uint64) (uint64, bool) {
	var u1 uint64
	var valid bool
	var invalid bool
	var none optionalU64
	var last uint64
	var position uint64
	var some optionalU64
	u1 = uint64(1)
	valid = (newest_offset < active_count)
	invalid = (!valid)
	if invalid {
		none = optionalU64{}
		return none.value, none.ok
	}
	last = (active_count - u1)
	position = (last - newest_offset)
	some = optionalU64{value: position, ok: true}
	return some.value, some.ok
}

func statefulFreshIdWindow(max_plus_one uint64) uint64 {
	var u1 uint64
	var window uint64
	u1 = uint64(1)
	window = (max_plus_one + u1)
	return window
}

func statefulSetChoiceWindow(max_plus_one uint64) uint64 {
	var window uint64
	window = max_plus_one
	return window
}

func statefulFreshIdFallbackStep(candidate uint64, used_id uint64) uint64 {
	var u1 uint64
	var next_ uint64
	var taken bool
	u1 = uint64(1)
	next_ = candidate
	taken = (used_id == candidate)
	if taken {
		next_ = (candidate + u1)
	}
	return next_
}

func statefulFreshIdAccepts(value uint64, window_hi uint64, in_use bool) bool {
	var in_window bool
	var unused bool
	var accepts bool
	in_window = (value <= window_hi)
	unused = (!in_use)
	accepts = (in_window && unused)
	return accepts
}

func statefulSetChoiceAccepts(value uint64, window_hi uint64) bool {
	var accepts bool
	accepts = (value <= window_hi)
	return accepts
}

func statefulSetChoiceBetter(candidate uint64, raw uint64, best uint64) bool {
	var at_or_below bool
	var above_best bool
	var better bool
	at_or_below = (candidate <= raw)
	above_best = (candidate > best)
	better = (at_or_below && above_best)
	return better
}

func outcomeIsFailure(outcome uint64) bool {
	var interesting uint64
	var is_failure bool
	interesting = uint64(4)
	is_failure = (outcome == interesting)
	return is_failure
}

func shrinkAccept(outcome uint64, origin_matches bool) bool {
	var is_failure bool
	var accept bool
	is_failure = outcomeIsFailure(outcome)
	accept = (is_failure && origin_matches)
	return accept
}

func shrinkCostProgress(candidate_cost uint64, current_cost uint64, stream_simpler bool) bool {
	var false_ bool
	var true_ bool
	var cheaper bool
	var dearer bool
	false_ = false
	true_ = true
	cheaper = (candidate_cost < current_cost)
	if cheaper {
		return true_
	}
	dearer = (candidate_cost > current_cost)
	if dearer {
		return false_
	}
	return stream_simpler
}

func generationRejectionLimit(base uint64, valid uint64) uint64 {
	var u100 uint64
	var max uint64
	var max_valid uint64
	var scale_overflows bool
	var earned uint64
	var remaining uint64
	var add_overflows bool
	var limit uint64
	u100 = uint64(100)
	max = uint64(18446744073709551615)
	max_valid = uint64(184467440737095516)
	scale_overflows = (valid > max_valid)
	if scale_overflows {
		return max
	}
	earned = (valid * u100)
	remaining = (max - earned)
	add_overflows = (base > remaining)
	if add_overflows {
		return max
	}
	limit = (base + earned)
	return limit
}

func healthFilterTooMuch(rejections uint64, valid uint64, suppressed bool) bool {
	var u10 uint64
	var u50 uint64
	var many_rejections bool
	var few_valid bool
	var active bool
	var unhealthy bool
	var report bool
	u10 = uint64(10)
	u50 = uint64(50)
	many_rejections = (rejections >= u50)
	few_valid = (valid < u10)
	active = (!suppressed)
	unhealthy = (many_rejections && few_valid)
	report = (unhealthy && active)
	return report
}

func choiceBudgetExhausted(draw_count uint64) bool {
	var buffer_size uint64
	var exhausted bool
	buffer_size = uint64(8192)
	exhausted = (draw_count >= buffer_size)
	return exhausted
}

func healthDataTooLarge(overruns uint64, valid uint64, suppressed bool) bool {
	var u10 uint64
	var u20 uint64
	var many_overruns bool
	var few_valid bool
	var active bool
	var unhealthy bool
	var report bool
	u10 = uint64(10)
	u20 = uint64(20)
	many_overruns = (overruns >= u20)
	few_valid = (valid < u10)
	active = (!suppressed)
	unhealthy = (many_overruns && few_valid)
	report = (unhealthy && active)
	return report
}

func healthLargeBaseExample(overran bool, outcome uint64, draw_count uint64, suppressed bool) bool {
	var u0 uint64
	var half_buffer uint64
	var valid bool
	var large bool
	var large_valid bool
	var unhealthy bool
	var active bool
	var report bool
	u0 = uint64(0)
	half_buffer = uint64(4096)
	valid = (outcome == u0)
	large = (draw_count > half_buffer)
	large_valid = (valid && large)
	unhealthy = (overran || large_valid)
	active = (!suppressed)
	report = (unhealthy && active)
	return report
}

func healthTooSlow(generation_us uint64, valid uint64, suppressed bool) bool {
	var u10 uint64
	var one_second_us uint64
	var slow bool
	var few_valid bool
	var active bool
	var unhealthy bool
	var report bool
	u10 = uint64(10)
	one_second_us = uint64(1000000)
	slow = (generation_us > one_second_us)
	few_valid = (valid < u10)
	active = (!suppressed)
	unhealthy = (slow && few_valid)
	report = (unhealthy && active)
	return report
}

func coverageFailed(observed uint64, valid uint64, min_permille uint64) bool {
	var true_ bool
	var false_ bool
	var zero uint64
	var thousand uint64
	var max_scaled uint64
	var never bool
	var no_rate bool
	var lhs_overflows bool
	var over_rate bool
	var failed_clamped bool
	var rhs_overflows bool
	var lhs uint64
	var rhs uint64
	var failed bool
	true_ = true
	false_ = false
	zero = uint64(0)
	thousand = uint64(1000)
	max_scaled = uint64(18446744073709551)
	never = (observed == zero)
	if never {
		return true_
	}
	no_rate = (min_permille == zero)
	if no_rate {
		return false_
	}
	lhs_overflows = (observed > max_scaled)
	if lhs_overflows {
		return false_
	}
	over_rate = (min_permille > thousand)
	if over_rate {
		failed_clamped = (observed < valid)
		return failed_clamped
	}
	rhs_overflows = (valid > max_scaled)
	if rhs_overflows {
		return true_
	}
	lhs = (observed * thousand)
	rhs = (min_permille * valid)
	failed = (lhs < rhs)
	return failed
}

func shrinkCandidateAt(current uint64, ordinal uint64) (uint64, bool) {
	var u0 uint64
	var u1 uint64
	var u4 uint64
	var u5 uint64
	var empty bool
	var none_empty optionalU64
	var within_current bool
	var within_prefix bool
	var linear bool
	var some_linear optionalU64
	var prefix_exhausts bool
	var none_prefix optionalU64
	var seen uint64
	var lower uint64
	var gap uint64
	var exhausted bool
	var half uint64
	var probe uint64
	var want_probe bool
	var some_probe optionalU64
	var none optionalU64
	u0 = uint64(0)
	u1 = uint64(1)
	u4 = uint64(4)
	u5 = uint64(5)
	empty = (current == u0)
	if empty {
		none_empty = optionalU64{}
		return none_empty.value, none_empty.ok
	}
	within_current = (ordinal < current)
	within_prefix = (ordinal < u5)
	linear = (within_current && within_prefix)
	if linear {
		some_linear = optionalU64{value: ordinal, ok: true}
		return some_linear.value, some_linear.ok
	}
	prefix_exhausts = (current <= u5)
	if prefix_exhausts {
		none_prefix = optionalU64{}
		return none_prefix.value, none_prefix.ok
	}
	seen = u5
	lower = u4
	for {
		gap = (current - lower)
		exhausted = (gap <= u1)
		if exhausted {
			break
		}
		half = (gap >> u1)
		probe = (lower + half)
		want_probe = (ordinal == seen)
		if want_probe {
			some_probe = optionalU64{value: probe, ok: true}
			return some_probe.value, some_probe.ok
		}
		seen = (seen + u1)
		lower = probe
	}
	none = optionalU64{}
	return none.value, none.ok
}

func shrinkNextShrinks(phase uint64, feedback uint64, shrinks uint64) uint64 {
	var u1 uint64
	var phase_baseline uint64
	var phase_verify uint64
	var feedback_accepted uint64
	var in_baseline bool
	var in_verify bool
	var accepted bool
	var bumped uint64
	u1 = uint64(1)
	phase_baseline = uint64(0)
	phase_verify = uint64(3)
	feedback_accepted = uint64(2)
	in_baseline = (phase == phase_baseline)
	if in_baseline {
		return shrinks
	}
	in_verify = (phase == phase_verify)
	if in_verify {
		return shrinks
	}
	accepted = (feedback == feedback_accepted)
	if accepted {
		bumped = (shrinks + u1)
		return bumped
	}
	return shrinks
}

func shrinkSweepAgain(improved bool, shrinks uint64, max_shrinks uint64) bool {
	var u0 uint64
	var false_ bool
	var true_ bool
	var idle bool
	var uncapped bool
	var under bool
	u0 = uint64(0)
	false_ = false
	true_ = true
	idle = (!improved)
	if idle {
		return false_
	}
	uncapped = (max_shrinks == u0)
	if uncapped {
		return true_
	}
	under = (shrinks < max_shrinks)
	return under
}

func shrinkNextStall(stall uint64, feedback uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var feedback_skip uint64
	var feedback_accepted uint64
	var accepted bool
	var skipped bool
	var bumped uint64
	u0 = uint64(0)
	u1 = uint64(1)
	feedback_skip = uint64(0)
	feedback_accepted = uint64(2)
	accepted = (feedback == feedback_accepted)
	if accepted {
		return u0
	}
	skipped = (feedback == feedback_skip)
	if skipped {
		return stall
	}
	bumped = (stall + u1)
	return bumped
}

func shrinkNextSpanCursor(phase uint64, feedback uint64, span_cursor uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var phase_delete uint64
	var feedback_accepted uint64
	var in_delete bool
	var accepted bool
	var advanced uint64
	u0 = uint64(0)
	u1 = uint64(1)
	phase_delete = uint64(1)
	feedback_accepted = uint64(2)
	in_delete = (phase == phase_delete)
	if in_delete {
		accepted = (feedback == feedback_accepted)
		if accepted {
			return u0
		}
		advanced = (span_cursor + u1)
		return advanced
	}
	return u0
}

func shrinkNextPaidCursor(phase uint64, feedback uint64, paid_cursor uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var phase_paid uint64
	var feedback_skip uint64
	var feedback_accepted uint64
	var in_paid bool
	var accepted bool
	var skipped bool
	var advanced uint64
	u0 = uint64(0)
	u1 = uint64(1)
	phase_paid = uint64(12)
	feedback_skip = uint64(0)
	feedback_accepted = uint64(2)
	in_paid = (phase == phase_paid)
	if in_paid {
		accepted = (feedback == feedback_accepted)
		if accepted {
			return u0
		}
		skipped = (feedback == feedback_skip)
		if skipped {
			advanced = (paid_cursor + u1)
			return advanced
		}
		return paid_cursor
	}
	return u0
}

func shrinkDeletePaidVariant(ordinal uint64, widened bool) (int64, bool) {
	var u0 uint64
	var u1 uint64
	var i0 int64
	var i1 int64
	var step uint64
	var plain bool
	var some_plain optionalI64
	var down bool
	var minus_one int64
	var some_down optionalI64
	var up bool
	var some_up optionalI64
	var none optionalI64
	u0 = uint64(0)
	u1 = uint64(1)
	i0 = int64(0)
	i1 = int64(1)
	step = ordinal
	if widened {
		plain = (ordinal == u0)
		if plain {
			some_plain = optionalI64{value: i0, ok: true}
			return some_plain.value, some_plain.ok
		}
		step = (ordinal - u1)
	}
	down = (step == u0)
	if down {
		minus_one = (i0 - i1)
		some_down = optionalI64{value: minus_one, ok: true}
		return some_down.value, some_down.ok
	}
	up = (step == u1)
	if up {
		some_up = optionalI64{value: i1, ok: true}
		return some_up.value, some_up.ok
	}
	none = optionalI64{}
	return none.value, none.ok
}

func shrinkNextRaiseCursor(phase uint64, feedback uint64, raise_cursor uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var phase_raise uint64
	var feedback_skip uint64
	var feedback_accepted uint64
	var in_raise bool
	var accepted bool
	var skipped bool
	var advanced uint64
	u0 = uint64(0)
	u1 = uint64(1)
	phase_raise = uint64(13)
	feedback_skip = uint64(0)
	feedback_accepted = uint64(2)
	in_raise = (phase == phase_raise)
	if in_raise {
		accepted = (feedback == feedback_accepted)
		if accepted {
			return u0
		}
		skipped = (feedback == feedback_skip)
		if skipped {
			advanced = (raise_cursor + u1)
			return advanced
		}
		return raise_cursor
	}
	return u0
}

func shrinkRaiseCandidate(current_index uint64, max_index uint64, ordinal uint64) (uint64, bool) {
	var u0 uint64
	var u1 uint64
	var below_max bool
	var at_max bool
	var none_max optionalU64
	var next_ uint64
	var first bool
	var some_next optionalU64
	var second bool
	var beyond_next bool
	var jumps bool
	var some_max optionalU64
	var none optionalU64
	u0 = uint64(0)
	u1 = uint64(1)
	below_max = (current_index < max_index)
	at_max = (!below_max)
	if at_max {
		none_max = optionalU64{}
		return none_max.value, none_max.ok
	}
	next_ = (current_index + u1)
	first = (ordinal == u0)
	if first {
		some_next = optionalU64{value: next_, ok: true}
		return some_next.value, some_next.ok
	}
	second = (ordinal == u1)
	beyond_next = (next_ < max_index)
	jumps = (second && beyond_next)
	if jumps {
		some_max = optionalU64{value: max_index, ok: true}
		return some_max.value, some_max.ok
	}
	none = optionalU64{}
	return none.value, none.ok
}

func shrinkNextPosition(phase uint64, feedback uint64, position uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var phase_lower uint64
	var feedback_skip uint64
	var feedback_accepted uint64
	var in_lower bool
	var accepted bool
	var skipped bool
	var advanced uint64
	u0 = uint64(0)
	u1 = uint64(1)
	phase_lower = uint64(2)
	feedback_skip = uint64(0)
	feedback_accepted = uint64(2)
	in_lower = (phase == phase_lower)
	if in_lower {
		accepted = (feedback == feedback_accepted)
		if accepted {
			return u0
		}
		skipped = (feedback == feedback_skip)
		if skipped {
			advanced = (position + u1)
			return advanced
		}
		return position
	}
	return u0
}

func shrinkNextShortenCursor(phase uint64, feedback uint64, shorten_cursor uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var phase_shorten uint64
	var feedback_skip uint64
	var feedback_accepted uint64
	var in_shorten bool
	var accepted bool
	var skipped bool
	var advanced uint64
	u0 = uint64(0)
	u1 = uint64(1)
	phase_shorten = uint64(11)
	feedback_skip = uint64(0)
	feedback_accepted = uint64(2)
	in_shorten = (phase == phase_shorten)
	if in_shorten {
		accepted = (feedback == feedback_accepted)
		if accepted {
			return u0
		}
		skipped = (feedback == feedback_skip)
		if skipped {
			advanced = (shorten_cursor + u1)
			return advanced
		}
		return shorten_cursor
	}
	return u0
}

func shrinkNextCharacterCursor(phase uint64, feedback uint64, character_cursor uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var phase_character uint64
	var feedback_skip uint64
	var feedback_accepted uint64
	var in_character bool
	var accepted bool
	var skipped bool
	var advanced uint64
	u0 = uint64(0)
	u1 = uint64(1)
	phase_character = uint64(10)
	feedback_skip = uint64(0)
	feedback_accepted = uint64(2)
	in_character = (phase == phase_character)
	if in_character {
		accepted = (feedback == feedback_accepted)
		if accepted {
			return u0
		}
		skipped = (feedback == feedback_skip)
		if skipped {
			advanced = (character_cursor + u1)
			return advanced
		}
		return character_cursor
	}
	return u0
}

func shrinkNextDescendantCursor(phase uint64, feedback uint64, descendant_cursor uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var phase_descendant uint64
	var feedback_skip uint64
	var feedback_accepted uint64
	var in_descendant bool
	var accepted bool
	var skipped bool
	var advanced uint64
	u0 = uint64(0)
	u1 = uint64(1)
	phase_descendant = uint64(9)
	feedback_skip = uint64(0)
	feedback_accepted = uint64(2)
	in_descendant = (phase == phase_descendant)
	if in_descendant {
		accepted = (feedback == feedback_accepted)
		if accepted {
			return u0
		}
		skipped = (feedback == feedback_skip)
		if skipped {
			advanced = (descendant_cursor + u1)
			return advanced
		}
		return descendant_cursor
	}
	return u0
}

func shrinkDescendantWorthwhile(ancestor_choices uint64, descendant_choices uint64) bool {
	var u0 uint64
	var nonempty bool
	var shorter bool
	var worthwhile bool
	u0 = uint64(0)
	nonempty = (descendant_choices > u0)
	shorter = (descendant_choices < ancestor_choices)
	worthwhile = (nonempty && shorter)
	return worthwhile
}

func shrinkNextTrivialCursor(phase uint64, feedback uint64, trivial_cursor uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var phase_trivial uint64
	var feedback_accepted uint64
	var in_trivial bool
	var accepted bool
	var advanced uint64
	u0 = uint64(0)
	u1 = uint64(1)
	phase_trivial = uint64(8)
	feedback_accepted = uint64(2)
	in_trivial = (phase == phase_trivial)
	if in_trivial {
		accepted = (feedback == feedback_accepted)
		if accepted {
			return u0
		}
		advanced = (trivial_cursor + u1)
		return advanced
	}
	return u0
}

func shrinkTrivialWorthwhile(rewritable uint64) bool {
	var u0 uint64
	var worthwhile bool
	u0 = uint64(0)
	worthwhile = (rewritable > u0)
	return worthwhile
}

func shrinkNextReorderCursor(phase uint64, feedback uint64, reorder_cursor uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var phase_reorder uint64
	var feedback_accepted uint64
	var in_reorder bool
	var accepted bool
	var advanced uint64
	u0 = uint64(0)
	u1 = uint64(1)
	phase_reorder = uint64(7)
	feedback_accepted = uint64(2)
	in_reorder = (phase == phase_reorder)
	if in_reorder {
		accepted = (feedback == feedback_accepted)
		if accepted {
			return u0
		}
		advanced = (reorder_cursor + u1)
		return advanced
	}
	return u0
}

func shrinkNextPairCursor(phase uint64, feedback uint64, pair_cursor uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var phase_pair uint64
	var feedback_skip uint64
	var feedback_accepted uint64
	var in_pair bool
	var accepted bool
	var skipped bool
	var advanced uint64
	u0 = uint64(0)
	u1 = uint64(1)
	phase_pair = uint64(6)
	feedback_skip = uint64(0)
	feedback_accepted = uint64(2)
	in_pair = (phase == phase_pair)
	if in_pair {
		accepted = (feedback == feedback_accepted)
		if accepted {
			return u0
		}
		skipped = (feedback == feedback_skip)
		if skipped {
			advanced = (pair_cursor + u1)
			return advanced
		}
		return pair_cursor
	}
	return u0
}

func shrinkPairCount(draw_count uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var empty bool
	var pairs uint64
	u0 = uint64(0)
	u1 = uint64(1)
	empty = (draw_count < u1)
	if empty {
		return u0
	}
	pairs = (draw_count - u1)
	return pairs
}

func shrinkNextDuplicateCursor(phase uint64, feedback uint64, duplicate_cursor uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var phase_duplicate uint64
	var feedback_skip uint64
	var feedback_accepted uint64
	var in_duplicate bool
	var accepted bool
	var skipped bool
	var advanced uint64
	u0 = uint64(0)
	u1 = uint64(1)
	phase_duplicate = uint64(5)
	feedback_skip = uint64(0)
	feedback_accepted = uint64(2)
	in_duplicate = (phase == phase_duplicate)
	if in_duplicate {
		accepted = (feedback == feedback_accepted)
		if accepted {
			return u0
		}
		skipped = (feedback == feedback_skip)
		if skipped {
			advanced = (duplicate_cursor + u1)
			return advanced
		}
		return duplicate_cursor
	}
	return u0
}

func shrinkNextOrdinal(phase uint64, feedback uint64, ordinal uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var phase_lower uint64
	var phase_duplicate uint64
	var phase_pair uint64
	var phase_descendant uint64
	var phase_character uint64
	var phase_shorten uint64
	var phase_paid uint64
	var phase_raise uint64
	var feedback_rejected uint64
	var is_lower bool
	var is_duplicate bool
	var is_pair bool
	var is_descendant bool
	var is_character bool
	var is_shorten bool
	var is_paid bool
	var is_raise bool
	var walks_schedule bool
	var walks_ordinal bool
	var walks_span bool
	var walks_string bool
	var walks_shorten bool
	var walks_paid bool
	var in_lower bool
	var rejected bool
	var advanced uint64
	u0 = uint64(0)
	u1 = uint64(1)
	phase_lower = uint64(2)
	phase_duplicate = uint64(5)
	phase_pair = uint64(6)
	phase_descendant = uint64(9)
	phase_character = uint64(10)
	phase_shorten = uint64(11)
	phase_paid = uint64(12)
	phase_raise = uint64(13)
	feedback_rejected = uint64(1)
	is_lower = (phase == phase_lower)
	is_duplicate = (phase == phase_duplicate)
	is_pair = (phase == phase_pair)
	is_descendant = (phase == phase_descendant)
	is_character = (phase == phase_character)
	is_shorten = (phase == phase_shorten)
	is_paid = (phase == phase_paid)
	is_raise = (phase == phase_raise)
	walks_schedule = (is_lower || is_duplicate)
	walks_ordinal = (walks_schedule || is_pair)
	walks_span = (walks_ordinal || is_descendant)
	walks_string = (walks_span || is_character)
	walks_shorten = (walks_string || is_shorten)
	walks_paid = (walks_shorten || is_paid)
	in_lower = (walks_paid || is_raise)
	if in_lower {
		rejected = (feedback == feedback_rejected)
		if rejected {
			advanced = (ordinal + u1)
			return advanced
		}
		return u0
	}
	return u0
}

func shrinkNextPhase(phase uint64, feedback uint64, span_cursor uint64, duplicate_cursor uint64, position uint64, pair_cursor uint64, reorder_cursor uint64, trivial_cursor uint64, descendant_cursor uint64, character_cursor uint64, shorten_cursor uint64, paid_cursor uint64, raise_cursor uint64, shrinks uint64, stall uint64, max_shrinks uint64, max_stall uint64, draw_count uint64, span_count uint64) uint64 {
	var u0 uint64
	var phase_baseline uint64
	var phase_delete uint64
	var phase_lower uint64
	var phase_verify uint64
	var phase_done uint64
	var phase_duplicate uint64
	var phase_pair uint64
	var phase_reorder uint64
	var phase_trivial uint64
	var phase_descendant uint64
	var phase_character uint64
	var phase_shorten uint64
	var phase_paid uint64
	var phase_raise uint64
	var feedback_accepted uint64
	var accepted bool
	var next_ uint64
	var in_baseline bool
	var in_delete bool
	var in_paid bool
	var in_raise bool
	var in_descendant bool
	var in_verify bool
	var settled bool
	var shrinks_next uint64
	var stall_next uint64
	var shrinks_capped bool
	var over_shrinks bool
	var stop_shrinks bool
	var stall_capped bool
	var over_stall bool
	var stop_stall bool
	var lands_delete bool
	var span_next uint64
	var spans_left bool
	var spans_done bool
	var lands_paid bool
	var paid_next uint64
	var paid_left bool
	var paid_done bool
	var lands_trivial bool
	var trivial_next uint64
	var trivial_left bool
	var trivial_done bool
	var lands_descendant bool
	var descendant_next uint64
	var descendants_left bool
	var descendants_done bool
	var lands_reorder bool
	var reorder_next uint64
	var siblings_left bool
	var siblings_done bool
	var lands_duplicate bool
	var duplicate_next uint64
	var duplicate_left bool
	var duplicate_done bool
	var lands_lower bool
	var position_next uint64
	var draws_left bool
	var draws_done bool
	var lands_pair bool
	var pair_next uint64
	var pair_count uint64
	var pairs_left bool
	var pairs_done bool
	var lands_shorten bool
	var shorten_next uint64
	var shorten_left bool
	var shorten_done bool
	var lands_character bool
	var character_next uint64
	var characters_left bool
	var characters_done bool
	var lands_raise bool
	var raise_next uint64
	var raisable uint64
	var raises_left bool
	var raises_done bool
	u0 = uint64(0)
	phase_baseline = uint64(0)
	phase_delete = uint64(1)
	phase_lower = uint64(2)
	phase_verify = uint64(3)
	phase_done = uint64(4)
	phase_duplicate = uint64(5)
	phase_pair = uint64(6)
	phase_reorder = uint64(7)
	phase_trivial = uint64(8)
	phase_descendant = uint64(9)
	phase_character = uint64(10)
	phase_shorten = uint64(11)
	phase_paid = uint64(12)
	phase_raise = uint64(13)
	feedback_accepted = uint64(2)
	accepted = (feedback == feedback_accepted)
	next_ = phase
	in_baseline = (phase == phase_baseline)
	if in_baseline {
		next_ = phase_lower
		if accepted {
			next_ = phase_delete
		}
	}
	in_delete = (phase == phase_delete)
	if in_delete {
		if accepted {
			next_ = phase_baseline
		}
	}
	in_paid = (phase == phase_paid)
	if in_paid {
		if accepted {
			next_ = phase_baseline
		}
	}
	in_raise = (phase == phase_raise)
	if in_raise {
		if accepted {
			next_ = phase_baseline
		}
	}
	in_descendant = (phase == phase_descendant)
	if in_descendant {
		if accepted {
			next_ = phase_baseline
		}
	}
	in_verify = (phase == phase_verify)
	if in_verify {
		next_ = phase_done
	}
	settled = (next_ != phase_done)
	if settled {
		shrinks_next = shrinkNextShrinks(phase, feedback, shrinks)
		stall_next = shrinkNextStall(stall, feedback)
		shrinks_capped = (max_shrinks != u0)
		over_shrinks = (shrinks_next >= max_shrinks)
		stop_shrinks = (shrinks_capped && over_shrinks)
		if stop_shrinks {
			next_ = phase_verify
		}
		stall_capped = (max_stall != u0)
		over_stall = (stall_next >= max_stall)
		stop_stall = (stall_capped && over_stall)
		if stop_stall {
			next_ = phase_verify
		}
	}
	lands_delete = (next_ == phase_delete)
	if lands_delete {
		span_next = shrinkNextSpanCursor(phase, feedback, span_cursor)
		spans_left = (span_next < span_count)
		spans_done = (!spans_left)
		if spans_done {
			next_ = phase_paid
		}
	}
	lands_paid = (next_ == phase_paid)
	if lands_paid {
		paid_next = shrinkNextPaidCursor(phase, feedback, paid_cursor)
		paid_left = (paid_next < span_count)
		paid_done = (!paid_left)
		if paid_done {
			next_ = phase_trivial
		}
	}
	lands_trivial = (next_ == phase_trivial)
	if lands_trivial {
		trivial_next = shrinkNextTrivialCursor(phase, feedback, trivial_cursor)
		trivial_left = (trivial_next < span_count)
		trivial_done = (!trivial_left)
		if trivial_done {
			next_ = phase_descendant
		}
	}
	lands_descendant = (next_ == phase_descendant)
	if lands_descendant {
		descendant_next = shrinkNextDescendantCursor(phase, feedback, descendant_cursor)
		descendants_left = (descendant_next < span_count)
		descendants_done = (!descendants_left)
		if descendants_done {
			next_ = phase_reorder
		}
	}
	lands_reorder = (next_ == phase_reorder)
	if lands_reorder {
		reorder_next = shrinkNextReorderCursor(phase, feedback, reorder_cursor)
		siblings_left = (reorder_next < span_count)
		siblings_done = (!siblings_left)
		if siblings_done {
			next_ = phase_duplicate
		}
	}
	lands_duplicate = (next_ == phase_duplicate)
	if lands_duplicate {
		duplicate_next = shrinkNextDuplicateCursor(phase, feedback, duplicate_cursor)
		duplicate_left = (duplicate_next < draw_count)
		duplicate_done = (!duplicate_left)
		if duplicate_done {
			next_ = phase_lower
		}
	}
	lands_lower = (next_ == phase_lower)
	if lands_lower {
		position_next = shrinkNextPosition(phase, feedback, position)
		draws_left = (position_next < draw_count)
		draws_done = (!draws_left)
		if draws_done {
			next_ = phase_pair
		}
	}
	lands_pair = (next_ == phase_pair)
	if lands_pair {
		pair_next = shrinkNextPairCursor(phase, feedback, pair_cursor)
		pair_count = shrinkPairCount(draw_count)
		pairs_left = (pair_next < pair_count)
		pairs_done = (!pairs_left)
		if pairs_done {
			next_ = phase_shorten
		}
	}
	lands_shorten = (next_ == phase_shorten)
	if lands_shorten {
		shorten_next = shrinkNextShortenCursor(phase, feedback, shorten_cursor)
		shorten_left = (shorten_next < draw_count)
		shorten_done = (!shorten_left)
		if shorten_done {
			next_ = phase_character
		}
	}
	lands_character = (next_ == phase_character)
	if lands_character {
		character_next = shrinkNextCharacterCursor(phase, feedback, character_cursor)
		characters_left = (character_next < draw_count)
		characters_done = (!characters_left)
		if characters_done {
			next_ = phase_raise
		}
	}
	lands_raise = (next_ == phase_raise)
	if lands_raise {
		raise_next = shrinkNextRaiseCursor(phase, feedback, raise_cursor)
		raisable = shrinkPairCount(draw_count)
		raises_left = (raise_next < raisable)
		raises_done = (!raises_left)
		if raises_done {
			next_ = phase_verify
		}
	}
	return next_
}

func shrinkRequestKind(phase uint64, span_cursor uint64, duplicate_cursor uint64, position uint64, pair_cursor uint64, reorder_cursor uint64, trivial_cursor uint64, descendant_cursor uint64, character_cursor uint64, shorten_cursor uint64, paid_cursor uint64, raise_cursor uint64, draw_count uint64, span_count uint64) uint64 {
	var phase_baseline uint64
	var phase_delete uint64
	var phase_lower uint64
	var phase_verify uint64
	var phase_duplicate uint64
	var phase_pair uint64
	var phase_reorder uint64
	var phase_trivial uint64
	var phase_descendant uint64
	var phase_character uint64
	var phase_shorten uint64
	var phase_paid uint64
	var phase_raise uint64
	var request_done uint64
	var request_baseline uint64
	var request_delete uint64
	var request_lower uint64
	var request_verify uint64
	var request_duplicates uint64
	var request_pair uint64
	var request_reorder uint64
	var request_trivial uint64
	var request_descendant uint64
	var request_character uint64
	var request_shorten uint64
	var request_paid uint64
	var request_raise uint64
	var is_baseline bool
	var is_delete bool
	var spans_left bool
	var is_paid bool
	var paid_left bool
	var is_trivial bool
	var trivial_left bool
	var is_descendant bool
	var descendants_left bool
	var is_reorder bool
	var siblings_left bool
	var is_duplicate bool
	var groups_left bool
	var is_lower bool
	var draws_left bool
	var is_pair bool
	var pair_count uint64
	var pairs_left bool
	var is_shorten bool
	var shorten_left bool
	var is_character bool
	var characters_left bool
	var is_raise bool
	var raisable uint64
	var raises_left bool
	var is_verify bool
	phase_baseline = uint64(0)
	phase_delete = uint64(1)
	phase_lower = uint64(2)
	phase_verify = uint64(3)
	phase_duplicate = uint64(5)
	phase_pair = uint64(6)
	phase_reorder = uint64(7)
	phase_trivial = uint64(8)
	phase_descendant = uint64(9)
	phase_character = uint64(10)
	phase_shorten = uint64(11)
	phase_paid = uint64(12)
	phase_raise = uint64(13)
	request_done = uint64(0)
	request_baseline = uint64(1)
	request_delete = uint64(2)
	request_lower = uint64(3)
	request_verify = uint64(4)
	request_duplicates = uint64(5)
	request_pair = uint64(6)
	request_reorder = uint64(7)
	request_trivial = uint64(8)
	request_descendant = uint64(9)
	request_character = uint64(10)
	request_shorten = uint64(11)
	request_paid = uint64(12)
	request_raise = uint64(13)
	is_baseline = (phase == phase_baseline)
	if is_baseline {
		return request_baseline
	}
	is_delete = (phase == phase_delete)
	if is_delete {
		spans_left = (span_cursor < span_count)
		if spans_left {
			return request_delete
		}
		return request_done
	}
	is_paid = (phase == phase_paid)
	if is_paid {
		paid_left = (paid_cursor < span_count)
		if paid_left {
			return request_paid
		}
		return request_done
	}
	is_trivial = (phase == phase_trivial)
	if is_trivial {
		trivial_left = (trivial_cursor < span_count)
		if trivial_left {
			return request_trivial
		}
		return request_done
	}
	is_descendant = (phase == phase_descendant)
	if is_descendant {
		descendants_left = (descendant_cursor < span_count)
		if descendants_left {
			return request_descendant
		}
		return request_done
	}
	is_reorder = (phase == phase_reorder)
	if is_reorder {
		siblings_left = (reorder_cursor < span_count)
		if siblings_left {
			return request_reorder
		}
		return request_done
	}
	is_duplicate = (phase == phase_duplicate)
	if is_duplicate {
		groups_left = (duplicate_cursor < draw_count)
		if groups_left {
			return request_duplicates
		}
		return request_done
	}
	is_lower = (phase == phase_lower)
	if is_lower {
		draws_left = (position < draw_count)
		if draws_left {
			return request_lower
		}
		return request_done
	}
	is_pair = (phase == phase_pair)
	if is_pair {
		pair_count = shrinkPairCount(draw_count)
		pairs_left = (pair_cursor < pair_count)
		if pairs_left {
			return request_pair
		}
		return request_done
	}
	is_shorten = (phase == phase_shorten)
	if is_shorten {
		shorten_left = (shorten_cursor < draw_count)
		if shorten_left {
			return request_shorten
		}
		return request_done
	}
	is_character = (phase == phase_character)
	if is_character {
		characters_left = (character_cursor < draw_count)
		if characters_left {
			return request_character
		}
		return request_done
	}
	is_raise = (phase == phase_raise)
	if is_raise {
		raisable = shrinkPairCount(draw_count)
		raises_left = (raise_cursor < raisable)
		if raises_left {
			return request_raise
		}
		return request_done
	}
	is_verify = (phase == phase_verify)
	if is_verify {
		return request_verify
	}
	return request_done
}

func shrinkRequestSpan(span_cursor uint64, span_count uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var spans_left bool
	var exhausted bool
	var last uint64
	var index uint64
	u0 = uint64(0)
	u1 = uint64(1)
	spans_left = (span_cursor < span_count)
	exhausted = (!spans_left)
	if exhausted {
		return u0
	}
	last = (span_count - u1)
	index = (last - span_cursor)
	return index
}

func shrinkDuplicateWorthwhile(members uint64, current_index uint64) bool {
	var u0 uint64
	var u2 uint64
	var shared bool
	var lowerable bool
	var worthwhile bool
	u0 = uint64(0)
	u2 = uint64(2)
	shared = (members >= u2)
	lowerable = (current_index != u0)
	worthwhile = (shared && lowerable)
	return worthwhile
}

func shrinkPairBase(left_index uint64, right_index uint64) uint64 {
	var right_smaller bool
	right_smaller = (right_index < left_index)
	if right_smaller {
		return right_index
	}
	return left_index
}

func shrinkPairWorthwhile(left_index uint64, right_index uint64) bool {
	var u0 uint64
	var base uint64
	var worthwhile bool
	u0 = uint64(0)
	base = shrinkPairBase(left_index, right_index)
	worthwhile = (base != u0)
	return worthwhile
}

func shrinkPairCandidate(index uint64, base uint64, candidate uint64) (uint64, bool) {
	var rising bool
	var none_rising optionalU64
	var shift uint64
	var too_small bool
	var none_small optionalU64
	var shifted uint64
	var some_shifted optionalU64
	rising = (candidate > base)
	if rising {
		none_rising = optionalU64{}
		return none_rising.value, none_rising.ok
	}
	shift = (base - candidate)
	too_small = (index < shift)
	if too_small {
		none_small = optionalU64{}
		return none_small.value, none_small.ok
	}
	shifted = (index - shift)
	some_shifted = optionalU64{value: shifted, ok: true}
	return some_shifted.value, some_shifted.ok
}

func shrinkCandidateCount(current uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var ordinal uint64
	var candidate optionalU64
	var present bool
	var done bool
	u0 = uint64(0)
	u1 = uint64(1)
	ordinal = u0
	for {
		candidate.value, candidate.ok = shrinkCandidateAt(current, ordinal)
		present = candidate.ok
		done = (!present)
		if done {
			break
		}
		ordinal = (ordinal + u1)
	}
	return ordinal
}

func lengthPContinue(min_size int64, max_size int64, has_max bool) float64 {
	var f0_5 float64
	var f1 float64
	var f2 float64
	var f5 float64
	var f0_99 float64
	var minimum uint64
	var min_f float64
	var infinity float64
	var max_f float64
	var bounded uint64
	var doubled float64
	var plus_five float64
	var floor float64
	var five_larger bool
	var span float64
	var midpoint float64
	var average float64
	var midpoint_smaller bool
	var desired_extra float64
	var max_extra float64
	var saturated bool
	var unbounded bool
	var denominator float64
	var ratio float64
	var p float64
	f0_5 = float64(0.5)
	f1 = float64(1)
	f2 = float64(2)
	f5 = float64(5)
	f0_99 = float64(0.99)
	minimum = uint64(min_size)
	min_f = float64(minimum)
	infinity = f64Infinity()
	max_f = infinity
	if has_max {
		bounded = uint64(max_size)
		max_f = float64(bounded)
	}
	doubled = float64((min_f * f2))
	plus_five = (min_f + f5)
	floor = doubled
	five_larger = (plus_five > doubled)
	if five_larger {
		floor = plus_five
	}
	span = (min_f + max_f)
	midpoint = float64((span * f0_5))
	average = floor
	midpoint_smaller = (midpoint < floor)
	if midpoint_smaller {
		average = midpoint
	}
	desired_extra = (average - min_f)
	max_extra = (max_f - min_f)
	saturated = (desired_extra >= max_extra)
	if saturated {
		return f0_99
	}
	unbounded = math.IsInf(max_f, 0)
	if unbounded {
		denominator = (f1 + desired_extra)
		ratio = (f1 / denominator)
		p = (f1 - ratio)
		return p
	}
	denominator = (f2 + desired_extra)
	ratio = (f1 / denominator)
	p = (f1 - ratio)
	return p
}

func manyDrawsChoice(min_size int64, max_size int64, has_max bool) bool {
	var fixed bool
	var bounded_and_fixed bool
	var varies bool
	fixed = (min_size == max_size)
	bounded_and_fixed = (has_max && fixed)
	varies = (!bounded_and_fixed)
	return varies
}

func manyForcedValue(force_stop bool, count int64, min_size int64, max_size int64, has_max bool, p_continue float64) (bool, bool) {
	var true_ bool
	var false_ bool
	var stop optionalBool
	var below_minimum bool
	var go_ optionalBool
	var at_maximum bool
	var f0 float64
	var f1 float64
	var never bool
	var always bool
	var free optionalBool
	true_ = true
	false_ = false
	if force_stop {
		stop = optionalBool{value: false_, ok: true}
		return stop.value, stop.ok
	}
	below_minimum = (count < min_size)
	if below_minimum {
		go_ = optionalBool{value: true_, ok: true}
		return go_.value, go_.ok
	}
	if has_max {
		at_maximum = (count >= max_size)
		if at_maximum {
			stop = optionalBool{value: false_, ok: true}
			return stop.value, stop.ok
		}
	}
	f0 = float64(0)
	f1 = float64(1)
	never = (p_continue <= f0)
	if never {
		stop = optionalBool{value: false_, ok: true}
		return stop.value, stop.ok
	}
	always = (p_continue >= f1)
	if always {
		go_ = optionalBool{value: true_, ok: true}
		return go_.value, go_.ok
	}
	free = optionalBool{}
	return free.value, free.ok
}

func filterRetryAllowed(attempts uint64) bool {
	var u3 uint64
	var allowed bool
	u3 = uint64(3)
	allowed = (attempts < u3)
	return allowed
}

func manyRejectionBudget(count int64) uint64 {
	var i0 int64
	var u2 uint64
	var u3 uint64
	var u_max uint64
	var half_max uint64
	var zero int64
	var accepted int64
	var positive bool
	var magnitude uint64
	var ceiling uint64
	var doubled uint64
	var fits bool
	var budget uint64
	var below_floor bool
	i0 = int64(0)
	u2 = uint64(2)
	u3 = uint64(3)
	u_max = uint64(18446744073709551615)
	half_max = uint64(9223372036854775807)
	zero = i0
	accepted = zero
	positive = (count > i0)
	if positive {
		accepted = count
	}
	magnitude = uint64(accepted)
	ceiling = u_max
	doubled = ceiling
	fits = (magnitude <= half_max)
	if fits {
		doubled = (magnitude * u2)
	}
	budget = doubled
	below_floor = (doubled < u3)
	if below_floor {
		budget = u3
	}
	return budget
}

func manyRejectAction(rejections uint64, count int64, min_size int64) uint64 {
	var retry_ uint64
	var stop uint64
	var invalid uint64
	var budget uint64
	var over_budget bool
	var within_budget bool
	var below_minimum bool
	retry_ = uint64(0)
	stop = uint64(1)
	invalid = uint64(2)
	budget = manyRejectionBudget(count)
	over_budget = (rejections > budget)
	within_budget = (!over_budget)
	if within_budget {
		return retry_
	}
	below_minimum = (count < min_size)
	if below_minimum {
		return invalid
	}
	return stop
}

func replayInteger(stream []byte, offset int64, min_value []byte, max_value []byte, target []byte) []byte {
	var entry optionalBytes
	var present bool
	var recorded []byte
	var usable bool
	var punned []byte
	entry.value, entry.ok = deserializeIntegerChoice(stream, offset)
	present = entry.ok
	if present {
		recorded = entry.value
		usable = integerValidate(min_value, max_value, recorded)
		if usable {
			return recorded
		}
	}
	punned = integerUnit(min_value, max_value, target)
	return punned
}

func replayBoolean(stream []byte, offset int64) bool {
	var entry optionalBool
	var present bool
	var recorded bool
	var punned bool
	entry.value, entry.ok = deserializeBooleanChoice(stream, offset)
	present = entry.ok
	if present {
		recorded = entry.value
		return recorded
	}
	punned = true
	return punned
}

func replayFloat(stream []byte, offset int64, min_value float64, max_value float64, allow_nan bool, allow_infinity bool, smallest_nonzero float64) (float64, bool) {
	var entry optionalF64
	var present bool
	var recorded float64
	var usable bool
	var found optionalF64
	var punned optionalF64
	entry.value, entry.ok = deserializeFloatChoice(stream, offset)
	present = entry.ok
	if present {
		recorded = entry.value
		usable = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, recorded)
		if usable {
			found = optionalF64{value: recorded, ok: true}
			return found.value, found.ok
		}
	}
	punned.value, punned.ok = floatUnit(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero)
	return punned.value, punned.ok
}

func replayBytes(stream []byte, offset int64, min_size int64, max_size int64) []byte {
	var entry optionalBytes
	var present bool
	var recorded []byte
	var usable bool
	var punned []byte
	entry.value, entry.ok = deserializeBytesChoice(stream, offset)
	present = entry.ok
	if present {
		recorded = entry.value
		usable = bytesValidate(min_size, max_size, recorded)
		if usable {
			return recorded
		}
	}
	punned = bytesUnit(min_size, max_size)
	return punned
}

func replayString(stream []byte, offset int64, alphabet []byte, min_size int64, max_size int64) []byte {
	var entry optionalBytes
	var present bool
	var recorded []byte
	var usable bool
	var punned []byte
	entry.value, entry.ok = deserializeStringChoice(stream, offset)
	present = entry.ok
	if present {
		recorded = entry.value
		usable = stringValidate(alphabet, min_size, max_size, recorded)
		if usable {
			return recorded
		}
	}
	punned = stringUnit(alphabet, min_size, max_size)
	return punned
}

func replayIntegerPuns(stream []byte, offset int64, min_value []byte, max_value []byte) bool {
	var false_ bool
	var true_ bool
	var entry optionalBytes
	var present bool
	var recorded []byte
	var usable bool
	false_ = false
	true_ = true
	entry.value, entry.ok = deserializeIntegerChoice(stream, offset)
	present = entry.ok
	if present {
		recorded = entry.value
		usable = integerValidate(min_value, max_value, recorded)
		if usable {
			return false_
		}
	}
	return true_
}

func replayBooleanPuns(stream []byte, offset int64) bool {
	var entry optionalBool
	var present bool
	var punned bool
	entry.value, entry.ok = deserializeBooleanChoice(stream, offset)
	present = entry.ok
	punned = (!present)
	return punned
}

func replayFloatPuns(stream []byte, offset int64, min_value float64, max_value float64, allow_nan bool, allow_infinity bool, smallest_nonzero float64) bool {
	var false_ bool
	var true_ bool
	var entry optionalF64
	var present bool
	var recorded float64
	var usable bool
	false_ = false
	true_ = true
	entry.value, entry.ok = deserializeFloatChoice(stream, offset)
	present = entry.ok
	if present {
		recorded = entry.value
		usable = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, recorded)
		if usable {
			return false_
		}
	}
	return true_
}

func replayBytesPuns(stream []byte, offset int64, min_size int64, max_size int64) bool {
	var false_ bool
	var true_ bool
	var entry optionalBytes
	var present bool
	var recorded []byte
	var usable bool
	false_ = false
	true_ = true
	entry.value, entry.ok = deserializeBytesChoice(stream, offset)
	present = entry.ok
	if present {
		recorded = entry.value
		usable = bytesValidate(min_size, max_size, recorded)
		if usable {
			return false_
		}
	}
	return true_
}

func replayStringPuns(stream []byte, offset int64, alphabet []byte, min_size int64, max_size int64) bool {
	var false_ bool
	var true_ bool
	var entry optionalBytes
	var present bool
	var recorded []byte
	var usable bool
	false_ = false
	true_ = true
	entry.value, entry.ok = deserializeStringChoice(stream, offset)
	present = entry.ok
	if present {
		recorded = entry.value
		usable = stringValidate(alphabet, min_size, max_size, recorded)
		if usable {
			return false_
		}
	}
	return true_
}

func keyAppendInt(encoded []byte, value_raw []byte) []byte {
	var tag uint64
	var out []byte
	var value []byte
	var length_i64 int64
	var length uint64
	var length_bytes []byte
	var with_length []byte
	var component []byte
	var result []byte
	tag = uint64(0)
	out = []byte{}
	out = append(out, byte(tag))
	value = canonicalizeSignedLe(value_raw)
	length_i64 = int64(len(value))
	length = uint64(length_i64)
	length_bytes = u32ToLe(length)
	with_length = appendBytes(out, length_bytes)
	component = appendBytes(with_length, value)
	result = appendBytes(encoded, component)
	return result
}

func keyAppendText(encoded []byte, value []byte) []byte {
	var tag uint64
	var out []byte
	var length_i64 int64
	var length uint64
	var length_bytes []byte
	var with_length []byte
	var component []byte
	var result []byte
	tag = uint64(1)
	out = []byte{}
	out = append(out, byte(tag))
	length_i64 = int64(len(value))
	length = uint64(length_i64)
	length_bytes = u32ToLe(length)
	with_length = appendBytes(out, length_bytes)
	component = appendBytes(with_length, value)
	result = appendBytes(encoded, component)
	return result
}

func keyHash(encoded []byte) uint64 {
	var i0 int64
	var i1 int64
	var u0 uint64
	var u1 uint64
	var offset_basis uint64
	var prime uint64
	var length int64
	var hash uint64
	var pos int64
	var done bool
	var byte_ uint64
	var reserved bool
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	u1 = uint64(1)
	offset_basis = uint64(14695981039346656037)
	prime = uint64(1099511628211)
	length = int64(len(encoded))
	hash = offset_basis
	pos = i0
	for {
		done = (pos >= length)
		if done {
			break
		}
		byte_ = uint64(encoded[pos])
		hash = (hash ^ byte_)
		hash = (hash * prime)
		pos = (pos + i1)
	}
	reserved = (hash == u0)
	if reserved {
		return u1
	}
	return hash
}

func targetingScheduleStep(max_test_cases uint64) uint64 {
	var u1 uint64
	var u10 uint64
	var u50 uint64
	var ten []byte
	var max_big []byte
	var tenth_big []byte
	var tenth_opt optionalU64
	var tenth uint64
	var cap_ uint64
	var over_fifty bool
	var half uint64
	var step uint64
	var half_wins bool
	var below_floor bool
	u1 = uint64(1)
	u10 = uint64(10)
	u50 = uint64(50)
	ten = u64ToBigintLe(u10)
	max_big = u64ToBigintLe(max_test_cases)
	tenth_big = positiveDivLeNonzero(max_big, ten)
	tenth_opt.value, tenth_opt.ok = bigintToU64Le(tenth_big)
	tenth = tenth_opt.value
	cap_ = tenth
	over_fifty = (cap_ > u50)
	if over_fifty {
		cap_ = u50
	}
	half = (max_test_cases >> u1)
	step = (cap_ + u1)
	half_wins = (half > step)
	if half_wins {
		step = half
	}
	below_floor = (step < u10)
	if below_floor {
		step = u10
	}
	return step
}

func targetingShouldFire(valid uint64, next_at uint64) bool {
	var fire bool
	fire = (valid >= next_at)
	return fire
}

func targetingNextThreshold(next_at uint64, step uint64) uint64 {
	var max uint64
	var room uint64
	var overflows bool
	var sum uint64
	max = uint64(18446744073709551615)
	room = (max - step)
	overflows = (next_at > room)
	if overflows {
		return max
	}
	sum = (next_at + step)
	return sum
}

func targetingTrialBudget(max_test_cases uint64) uint64 {
	var u10 uint64
	var max uint64
	var room uint64
	var overflows bool
	var budget uint64
	u10 = uint64(10)
	max = uint64(18446744073709551615)
	room = uint64(1844674407370955161)
	overflows = (max_test_cases > room)
	if overflows {
		return max
	}
	budget = (max_test_cases * u10)
	return budget
}

func targetingImprovementAllowance(round uint64) uint64 {
	var u10 uint64
	var u61 uint64
	var max uint64
	var too_far bool
	var allowance uint64
	u10 = uint64(10)
	u61 = uint64(61)
	max = uint64(18446744073709551615)
	too_far = (round >= u61)
	if too_far {
		return max
	}
	allowance = (u10 << round)
	return allowance
}

func targetingRecord(has_best bool, best_score float64, score float64) bool {
	var true_ bool
	var none_yet bool
	var improves bool
	true_ = true
	none_yet = (!has_best)
	if none_yet {
		return true_
	}
	improves = (score > best_score)
	return improves
}

func targetingAccept(new_score float64, current_score float64, new_count uint64, current_count uint64) uint64 {
	var reject uint64
	var lateral uint64
	var improvement uint64
	var worse bool
	var strict bool
	var grew bool
	reject = uint64(0)
	lateral = uint64(1)
	improvement = uint64(2)
	worse = (new_score < current_score)
	if worse {
		return reject
	}
	strict = (new_score > current_score)
	if strict {
		return improvement
	}
	grew = (new_count > current_count)
	if grew {
		return reject
	}
	return lateral
}

func targetingStepCap() uint64 {
	var cap_ uint64
	cap_ = uint64(1048576)
	return cap_
}

func signedLeFromI64(value int64) []byte {
	var i0 int64
	var negative bool
	var negated int64
	var negated_u uint64
	var magnitude []byte
	var out []byte
	var value_u uint64
	var out_pos []byte
	i0 = int64(0)
	negative = (value < i0)
	if negative {
		negated = (i0 - value)
		negated_u = uint64(negated)
		magnitude = u64ToBigintLe(negated_u)
		out = signedNegateLe(magnitude)
		return out
	}
	value_u = uint64(value)
	out_pos = u64ToBigintLe(value_u)
	return out_pos
}

func targetingStepInteger(min_value []byte, max_value []byte, value []byte, delta int64) ([]byte, bool) {
	var delta_big []byte
	var stepped []byte
	var valid bool
	var out optionalBytes
	var none optionalBytes
	delta_big = signedLeFromI64(delta)
	stepped = signedAddLe(value, delta_big)
	valid = integerValidate(min_value, max_value, stepped)
	if valid {
		out = optionalBytes{value: stepped, ok: true}
		return out.value, out.ok
	}
	none = optionalBytes{}
	return none.value, none.ok
}

func targetingStepBoolean(value bool, delta int64) (bool, bool) {
	var i0 int64
	var i1 int64
	var false_ bool
	var true_ bool
	var is_zero bool
	var same optionalBool
	var negative bool
	var magnitude int64
	var is_down bool
	var down optionalBool
	var none_down optionalBool
	var is_up bool
	var up optionalBool
	var none optionalBool
	i0 = int64(0)
	i1 = int64(1)
	false_ = false
	true_ = true
	is_zero = (delta == i0)
	if is_zero {
		same = optionalBool{value: value, ok: true}
		return same.value, same.ok
	}
	negative = (delta < i0)
	if negative {
		magnitude = (i0 - delta)
		is_down = (magnitude == i1)
		if is_down {
			down = optionalBool{value: false_, ok: true}
			return down.value, down.ok
		}
		none_down = optionalBool{}
		return none_down.value, none_down.ok
	}
	is_up = (delta == i1)
	if is_up {
		up = optionalBool{value: true_, ok: true}
		return up.value, up.ok
	}
	none = optionalBool{}
	return none.value, none.ok
}

func targetingStepFloat(min_value float64, max_value float64, allow_nan bool, allow_infinity bool, smallest_nonzero float64, value float64, delta int64) (float64, bool) {
	var i0 int64
	var negative bool
	var magnitude_i int64
	var magnitude_u uint64
	var delta_f float64
	var stepped float64
	var valid bool
	var out optionalF64
	var none optionalF64
	i0 = int64(0)
	negative = (delta < i0)
	magnitude_i = delta
	if negative {
		magnitude_i = (i0 - delta)
	}
	magnitude_u = uint64(magnitude_i)
	delta_f = float64(magnitude_u)
	if negative {
		delta_f = f64Neg(delta_f)
	}
	stepped = (value + delta_f)
	valid = floatValidate(min_value, max_value, allow_nan, allow_infinity, smallest_nonzero, stepped)
	if valid {
		out = optionalF64{value: stepped, ok: true}
		return out.value, out.ok
	}
	none = optionalF64{}
	return none.value, none.ok
}

func targetingStepBytes(min_size int64, max_size int64, value []byte, delta int64) ([]byte, bool) {
	var i0 int64
	var i1 int64
	var u0 uint64
	var n int64
	var le []byte
	var pos int64
	var at_start bool
	var byte_ uint64
	var delta_big []byte
	var sum []byte
	var zero []byte
	var sign int64
	var negative bool
	var none_negative optionalBytes
	var sum_len int64
	var mag_len int64
	var empty bool
	var last_at int64
	var last uint64
	var nonzero bool
	var width int64
	var narrow bool
	var out []byte
	var at int64
	var done bool
	var in_magnitude bool
	var digit uint64
	var valid bool
	var some optionalBytes
	var none optionalBytes
	i0 = int64(0)
	i1 = int64(1)
	u0 = uint64(0)
	n = int64(len(value))
	le = []byte{}
	pos = n
	for {
		at_start = (pos <= i0)
		if at_start {
			break
		}
		pos = (pos - i1)
		byte_ = uint64(value[pos])
		le = append(le, byte(byte_))
	}
	le = append(le, byte(u0))
	delta_big = signedLeFromI64(delta)
	sum = signedAddLe(le, delta_big)
	zero = bigintZero()
	sign = signedCompareLe(sum, zero)
	negative = (sign < i0)
	if negative {
		none_negative = optionalBytes{}
		return none_negative.value, none_negative.ok
	}
	sum_len = int64(len(sum))
	mag_len = sum_len
	for {
		empty = (mag_len <= i0)
		if empty {
			break
		}
		last_at = (mag_len - i1)
		last = uint64(sum[last_at])
		nonzero = (last != u0)
		if nonzero {
			break
		}
		mag_len = last_at
	}
	width = mag_len
	narrow = (width < n)
	if narrow {
		width = n
	}
	out = []byte{}
	at = width
	for {
		done = (at <= i0)
		if done {
			break
		}
		at = (at - i1)
		in_magnitude = (at < mag_len)
		digit = u0
		if in_magnitude {
			digit = uint64(sum[at])
		}
		out = append(out, byte(digit))
	}
	valid = bytesValidate(min_size, max_size, out)
	if valid {
		some = optionalBytes{value: out, ok: true}
		return some.value, some.ok
	}
	none = optionalBytes{}
	return none.value, none.ok
}

func findIntegerProbe(mode uint64, a uint64, b uint64) (uint64, bool) {
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var done bool
	var none optionalU64
	var linear bool
	var probe_a optionalU64
	var grow bool
	var probe_b optionalU64
	var gap uint64
	var half uint64
	var mid uint64
	var probe_mid optionalU64
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	done = (mode >= u3)
	if done {
		none = optionalU64{}
		return none.value, none.ok
	}
	linear = (mode < u1)
	if linear {
		probe_a = optionalU64{value: a, ok: true}
		return probe_a.value, probe_a.ok
	}
	grow = (mode < u2)
	if grow {
		probe_b = optionalU64{value: b, ok: true}
		return probe_b.value, probe_b.ok
	}
	gap = (b - a)
	half = (gap >> u1)
	mid = (a + half)
	probe_mid = optionalU64{value: mid, ok: true}
	return probe_mid.value, probe_mid.ok
}

func findIntegerResult(mode uint64, a uint64, b uint64) (uint64, bool) {
	var u3 uint64
	var done bool
	var answer optionalU64
	var none optionalU64
	u3 = uint64(3)
	done = (mode >= u3)
	if done {
		answer = optionalU64{value: a, ok: true}
		return answer.value, answer.ok
	}
	none = optionalU64{}
	return none.value, none.ok
}

func findIntegerBracketMode(lo uint64, hi uint64) uint64 {
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var next_ uint64
	var open bool
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	next_ = (lo + u1)
	open = (next_ < hi)
	if open {
		return u2
	}
	return u3
}

func findIntegerNextMode(mode uint64, a uint64, b uint64, ok bool) uint64 {
	var u0 uint64
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var u4 uint64
	var half_max uint64
	var done bool
	var linear bool
	var failed bool
	var still_small bool
	var grow bool
	var overflows bool
	var bracketed uint64
	var gap uint64
	var half uint64
	var mid uint64
	var above uint64
	var below uint64
	u0 = uint64(0)
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	u4 = uint64(4)
	half_max = uint64(9223372036854775807)
	done = (mode >= u3)
	if done {
		return u3
	}
	linear = (mode < u1)
	if linear {
		failed = (!ok)
		if failed {
			return u3
		}
		still_small = (a < u4)
		if still_small {
			return u0
		}
		return u1
	}
	grow = (mode < u2)
	if grow {
		if ok {
			overflows = (b > half_max)
			if overflows {
				return u3
			}
			return u1
		}
		bracketed = findIntegerBracketMode(a, b)
		return bracketed
	}
	gap = (b - a)
	half = (gap >> u1)
	mid = (a + half)
	if ok {
		above = findIntegerBracketMode(mid, b)
		return above
	}
	below = findIntegerBracketMode(a, mid)
	return below
}

func findIntegerNextA(mode uint64, a uint64, b uint64, ok bool) uint64 {
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var u4 uint64
	var done bool
	var linear bool
	var failed bool
	var previous uint64
	var still_small bool
	var next_ uint64
	var grow bool
	var gap uint64
	var half uint64
	var mid uint64
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	u4 = uint64(4)
	done = (mode >= u3)
	if done {
		return a
	}
	linear = (mode < u1)
	if linear {
		failed = (!ok)
		if failed {
			previous = (a - u1)
			return previous
		}
		still_small = (a < u4)
		if still_small {
			next_ = (a + u1)
			return next_
		}
		return u4
	}
	grow = (mode < u2)
	if grow {
		if ok {
			return b
		}
		return a
	}
	if ok {
		gap = (b - a)
		half = (gap >> u1)
		mid = (a + half)
		return mid
	}
	return a
}

func findIntegerNextB(mode uint64, a uint64, b uint64, ok bool) uint64 {
	var u1 uint64
	var u2 uint64
	var u3 uint64
	var u4 uint64
	var u5 uint64
	var half_max uint64
	var done bool
	var linear bool
	var failed bool
	var still_small bool
	var grow bool
	var overflows bool
	var doubled uint64
	var gap uint64
	var half uint64
	var mid uint64
	u1 = uint64(1)
	u2 = uint64(2)
	u3 = uint64(3)
	u4 = uint64(4)
	u5 = uint64(5)
	half_max = uint64(9223372036854775807)
	done = (mode >= u3)
	if done {
		return b
	}
	linear = (mode < u1)
	if linear {
		failed = (!ok)
		if failed {
			return b
		}
		still_small = (a < u4)
		if still_small {
			return b
		}
		return u5
	}
	grow = (mode < u2)
	if grow {
		if ok {
			overflows = (b > half_max)
			if overflows {
				return b
			}
			doubled = (b * u2)
			return doubled
		}
		return b
	}
	if ok {
		return b
	}
	gap = (b - a)
	half = (gap >> u1)
	mid = (a + half)
	return mid
}

func recursiveLargeTargetProbability() float64 {
	var probability float64
	probability = float64(0.75)
	return probability
}

func recursiveTargetDrawn(max_leaves uint64) bool {
	var u2 uint64
	var drawn bool
	u2 = uint64(2)
	drawn = (max_leaves >= u2)
	return drawn
}

func recursiveRetryAllowed(attempt uint64) bool {
	var u9 uint64
	var allowed bool
	u9 = uint64(9)
	allowed = (attempt < u9)
	return allowed
}

func recursiveEffectiveTarget(target uint64, attempt uint64) uint64 {
	var u1 uint64
	var shifted uint64
	var below_one bool
	u1 = uint64(1)
	shifted = (target >> attempt)
	below_one = (shifted < u1)
	if below_one {
		return u1
	}
	return shifted
}

func recursiveArityEstimate(closed_children uint64, closed_branches uint64) float64 {
	var prior_children float64
	var prior_branches float64
	var children float64
	var branches float64
	var numerator float64
	var denominator float64
	var estimate float64
	prior_children = float64(0.5)
	prior_branches = float64(0.25)
	children = float64(closed_children)
	branches = float64(closed_branches)
	numerator = (prior_children + children)
	denominator = (prior_branches + branches)
	estimate = (numerator / denominator)
	return estimate
}

func recursiveBranchProbability(mean_arity float64, max_leaves uint64) float64 {
	var f1 float64
	var f0_95 float64
	var u2 uint64
	var u4294967296 uint64
	var arity float64
	var arity_too_small bool
	var budget uint64
	var budget_too_small bool
	var budget_too_large bool
	var budget_f float64
	var numerator float64
	var scaled_arity float64
	var denominator float64
	var probability float64
	var over_cap bool
	f1 = float64(1)
	f0_95 = float64(0.95)
	u2 = uint64(2)
	u4294967296 = uint64(4294967296)
	arity = mean_arity
	arity_too_small = (arity < f1)
	if arity_too_small {
		arity = f1
	}
	budget = max_leaves
	budget_too_small = (budget < u2)
	if budget_too_small {
		budget = u2
	}
	budget_too_large = (budget > u4294967296)
	if budget_too_large {
		budget = u4294967296
	}
	budget_f = float64(budget)
	numerator = (budget_f - f1)
	scaled_arity = float64((budget_f * arity))
	denominator = (scaled_arity - f1)
	probability = (numerator / denominator)
	over_cap = (probability > f0_95)
	if over_cap {
		return f0_95
	}
	return probability
}

func recursivePricedProbability(closed_children uint64, closed_branches uint64, attempt uint64, target uint64) float64 {
	var arity float64
	var attempt_f float64
	var priced_arity float64
	var effective uint64
	var probability float64
	arity = recursiveArityEstimate(closed_children, closed_branches)
	attempt_f = float64(attempt)
	priced_arity = (arity + attempt_f)
	effective = recursiveEffectiveTarget(target, attempt)
	probability = recursiveBranchProbability(priced_arity, effective)
	return probability
}

func recursiveProfileBoundary(target float64, arity float64, tail_probability float64, max_depth uint64) uint64 {
	var u0 uint64
	var u1 uint64
	var u4096 uint64
	var f1 float64
	var grow float64
	var horizon uint64
	var too_far bool
	var tail_numerator float64
	var tail_times_arity float64
	var tail_denominator float64
	var lo float64
	var grow_times_arity float64
	var grow_stop float64
	var lift float64
	var hi float64
	var boundary uint64
	var reached bool
	var at_horizon bool
	var done bool
	var next_lift float64
	var next_ float64
	var gain float64
	var epsilon float64
	var plateau bool
	var lo_lift float64
	u0 = uint64(0)
	u1 = uint64(1)
	u4096 = uint64(4096)
	f1 = float64(1)
	grow = float64(0.95)
	horizon = max_depth
	too_far = (horizon > u4096)
	if too_far {
		horizon = u4096
	}
	tail_numerator = (f1 - tail_probability)
	tail_times_arity = float64((tail_probability * arity))
	tail_denominator = (f1 - tail_times_arity)
	lo = (tail_numerator / tail_denominator)
	grow_times_arity = float64((grow * arity))
	grow_stop = (f1 - grow)
	lift = float64((grow_times_arity * lo))
	hi = (grow_stop + lift)
	boundary = u0
	for {
		reached = (hi >= target)
		at_horizon = (boundary >= horizon)
		done = (reached || at_horizon)
		if done {
			break
		}
		next_lift = float64((grow_times_arity * hi))
		next_ = (grow_stop + next_lift)
		gain = (next_ - hi)
		epsilon = float64(0.000000001)
		plateau = (gain < epsilon)
		if plateau {
			break
		}
		lo_lift = float64((grow_times_arity * lo))
		lo = (grow_stop + lo_lift)
		hi = next_
		boundary = (boundary + u1)
	}
	return boundary
}

func recursiveProfileExpected(target float64, arity float64, tail_probability float64, max_depth uint64) float64 {
	var u0 uint64
	var u1 uint64
	var f1 float64
	var grow float64
	var zero_depth bool
	var boundary uint64
	var tail_numerator float64
	var tail_times_arity float64
	var tail_denominator float64
	var lo float64
	var grow_times_arity float64
	var grow_stop float64
	var lo_lift float64
	var hi float64
	var at uint64
	var done bool
	var next_lo_lift float64
	var next_hi_lift float64
	var hi_below bool
	var lo_above bool
	u0 = uint64(0)
	u1 = uint64(1)
	f1 = float64(1)
	grow = float64(0.95)
	zero_depth = (max_depth == u0)
	if zero_depth {
		return f1
	}
	boundary = recursiveProfileBoundary(target, arity, tail_probability, max_depth)
	tail_numerator = (f1 - tail_probability)
	tail_times_arity = float64((tail_probability * arity))
	tail_denominator = (f1 - tail_times_arity)
	lo = (tail_numerator / tail_denominator)
	grow_times_arity = float64((grow * arity))
	grow_stop = (f1 - grow)
	lo_lift = float64((grow_times_arity * lo))
	hi = (grow_stop + lo_lift)
	at = u0
	for {
		done = (at >= boundary)
		if done {
			break
		}
		next_lo_lift = float64((grow_times_arity * lo))
		lo = (grow_stop + next_lo_lift)
		next_hi_lift = float64((grow_times_arity * hi))
		hi = (grow_stop + next_hi_lift)
		at = (at + u1)
	}
	hi_below = (hi <= target)
	if hi_below {
		return hi
	}
	lo_above = (lo >= target)
	if lo_above {
		return lo
	}
	return target
}

func recursiveDecisionProbability(max_depth uint64, depth uint64, target uint64, attempt uint64, closed_children uint64, closed_branches uint64) float64 {
	var f0 float64
	var u1 uint64
	var at_cap bool
	var effective uint64
	var single bool
	var priced float64
	var arity float64
	var attempt_f float64
	var priced_arity float64
	var u2 uint64
	var tail float64
	var target_f float64
	var boundary uint64
	var before bool
	var grow float64
	var on_boundary bool
	var f1 float64
	var tail_numerator float64
	var tail_times_arity float64
	var tail_denominator float64
	var lo float64
	var grow_times_arity float64
	var grow_stop float64
	var lo_lift float64
	var hi float64
	var at uint64
	var done bool
	var next_lo_lift float64
	var next_hi_lift float64
	var hi_below bool
	var lo_above bool
	var gap float64
	var offset float64
	var fraction float64
	var probability_gap float64
	var scaled float64
	var interpolated float64
	f0 = float64(0)
	u1 = uint64(1)
	at_cap = (depth >= max_depth)
	if at_cap {
		return f0
	}
	effective = recursiveEffectiveTarget(target, attempt)
	single = (effective == u1)
	if single {
		priced = recursivePricedProbability(closed_children, closed_branches, attempt, target)
		return priced
	}
	arity = recursiveArityEstimate(closed_children, closed_branches)
	attempt_f = float64(attempt)
	priced_arity = (arity + attempt_f)
	u2 = uint64(2)
	tail = recursiveBranchProbability(priced_arity, u2)
	target_f = float64(effective)
	boundary = recursiveProfileBoundary(target_f, arity, tail, max_depth)
	before = (depth < boundary)
	if before {
		grow = float64(0.95)
		return grow
	}
	on_boundary = (depth == boundary)
	if on_boundary {
		f1 = float64(1)
		grow = float64(0.95)
		tail_numerator = (f1 - tail)
		tail_times_arity = float64((tail * arity))
		tail_denominator = (f1 - tail_times_arity)
		lo = (tail_numerator / tail_denominator)
		grow_times_arity = float64((grow * arity))
		grow_stop = (f1 - grow)
		lo_lift = float64((grow_times_arity * lo))
		hi = (grow_stop + lo_lift)
		at = uint64(0)
		for {
			done = (at >= boundary)
			if done {
				break
			}
			next_lo_lift = float64((grow_times_arity * lo))
			lo = (grow_stop + next_lo_lift)
			next_hi_lift = float64((grow_times_arity * hi))
			hi = (grow_stop + next_hi_lift)
			at = (at + u1)
		}
		hi_below = (hi <= target_f)
		if hi_below {
			return grow
		}
		lo_above = (lo >= target_f)
		if lo_above {
			return tail
		}
		gap = (hi - lo)
		offset = (target_f - lo)
		fraction = (offset / gap)
		probability_gap = (grow - tail)
		scaled = float64((probability_gap * fraction))
		interpolated = (tail + scaled)
		return interpolated
	}
	return tail
}

func recursiveFinishMispriced(max_depth uint64, target uint64, attempt uint64, leaves uint64, closed_children uint64, closed_branches uint64, start_probability float64, reprices uint64) bool {
	var u1 uint64
	var u2 uint64
	var maxed bool
	var false_ bool
	var effective uint64
	var single bool
	var repriced float64
	var difference float64
	var threshold float64
	var mispriced bool
	var half_floor uint64
	var remainder uint64
	var half_ceil uint64
	var too_small bool
	var arity float64
	var target_f float64
	var zero float64
	var reachable float64
	var f2 float64
	var twice_reachable float64
	var reachable_enough bool
	u1 = uint64(1)
	u2 = uint64(2)
	maxed = (reprices >= u2)
	if maxed {
		false_ = false
		return false_
	}
	effective = recursiveEffectiveTarget(target, attempt)
	single = (effective == u1)
	if single {
		repriced = recursivePricedProbability(closed_children, closed_branches, attempt, target)
		difference = (repriced - start_probability)
		threshold = float64(0.05)
		mispriced = (difference > threshold)
		return mispriced
	}
	half_floor = (effective >> u1)
	remainder = (effective & u1)
	half_ceil = (half_floor + remainder)
	too_small = (leaves < half_ceil)
	if too_small {
		arity = recursiveArityEstimate(closed_children, closed_branches)
		target_f = float64(effective)
		zero = float64(0)
		reachable = recursiveProfileExpected(target_f, arity, zero, max_depth)
		f2 = float64(2)
		twice_reachable = float64((reachable * f2))
		reachable_enough = (twice_reachable >= target_f)
		return reachable_enough
	}
	false_ = false
	return false_
}
