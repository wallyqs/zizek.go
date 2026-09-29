// Generated from conformance/shell/draw_ops.zir by the zizek emitter. Do not edit by hand.

package zizek

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func dpI64(text string) int64 {
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		panic(err)
	}
	return value
}

func dpInt(text string) int {
	value, err := strconv.Atoi(text)
	if err != nil {
		panic(err)
	}
	return value
}

func dpF64Bits(text string) float64 {
	bits, err := strconv.ParseUint(text, 16, 64)
	if err != nil {
		panic(err)
	}
	return math.Float64frombits(bits)
}

func dpAlphabet(text string) []byte {
	if text == "" {
		return nil
	}
	characters, err := hex.DecodeString(text)
	if err != nil {
		panic(err)
	}
	value, err := Alphabet(string(characters))
	if err != nil {
		panic(err)
	}
	return value
}

func dpU64Hex(text string) uint64 {
	value, err := strconv.ParseUint(text, 16, 64)
	if err != nil {
		panic(err)
	}
	return value
}

type dpOptionalIntValue struct {
	value int
	present bool
}

func dpOptionalInt(text string) dpOptionalIntValue {
	if text == "-" {
		return dpOptionalIntValue{}
	}
	return dpOptionalIntValue{value: dpInt(text), present: true}
}

// TraceDraw runs one op of a draw script against tc and returns what it
// drew, encoded as "<tag>:<value>". The vocabulary is declared in
// conformance/shell/draw_ops.zir; this dispatch is generated from it.
func TraceDraw(tc *TestCase, op string) (string, error) {
	head, rest, _ := strings.Cut(op, ":")
	fields := strings.Split(rest, ":")
	_ = fields
	switch head {
	case "i":
		value, err := tc.Integer(dpI64(fields[0]), dpI64(fields[1]), 0)
		if err != nil {
			return "", err
		}
		return "i:" + strconv.FormatInt(value, 10), nil
	case "b":
		value, err := tc.Boolean()
		if err != nil {
			return "", err
		}
		if value {
			return "b:1", nil
		}
		return "b:0", nil
	case "y":
		value, err := tc.Bytes(dpInt(fields[0]), dpInt(fields[1]))
		if err != nil {
			return "", err
		}
		return "y:" + hex.EncodeToString(value), nil
	case "f":
		value, err := tc.Float(dpF64Bits(fields[0]), dpF64Bits(fields[1]), false, false, math.SmallestNonzeroFloat64)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("f:%016x", math.Float64bits(value)), nil
	case "s":
		value, err := tc.String(dpInt(fields[0]), dpInt(fields[1]), dpAlphabet(fields[2]))
		if err != nil {
			return "", err
		}
		return "s:" + hex.EncodeToString([]byte(value)), nil
	case "k":
		value, err := tc.IntegerKeyed(dpI64(fields[0]), dpI64(fields[1]), 0, dpU64Hex(fields[2]))
		if err != nil {
			return "", err
		}
		return "k:" + strconv.FormatInt(value, 10), nil
	case "l":
		bounds := dpOptionalInt(fields[1])
		element := Integers().MinValue(dpI64(fields[2])).MaxValue(dpI64(fields[3]))
		value, err := Collect(tc, element, dpInt(fields[0]), bounds.value, bounds.present, "list.element")
		if err != nil {
			return "", err
		}
		encoded := make([]string, len(value))
		for at, item := range value {
			encoded[at] = strconv.FormatInt(item, 10)
		}
		return "l:" + strings.Join(encoded, ","), nil
	case "q":
		bounds := dpOptionalInt(fields[1])
		element := Integers().MinValue(dpI64(fields[2])).MaxValue(dpI64(fields[3]))
		value, err := CollectUnique(tc, element, func(item int64) int64 { return item }, dpInt(fields[0]), bounds.value, bounds.present, "set.element")
		if err != nil {
			return "", err
		}
		encoded := make([]string, len(value))
		for at, item := range value {
			encoded[at] = strconv.FormatInt(item, 10)
		}
		return "q:" + strings.Join(encoded, ","), nil
	case "c":
		child, err := tc.CloneStream()
		if err != nil { return "", err }
		value, err := child.Integer(dpI64(fields[0]), dpI64(fields[1]), 0)
		if err != nil { return "", err }
		return "c:" + strconv.FormatInt(value, 10), nil
	}
	return "", fmt.Errorf("unknown draw op %q", head)
}
