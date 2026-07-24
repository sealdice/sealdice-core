package sealcrypto

import (
	"errors"
	"strings"

	"github.com/dop251/goja"
)

func ensureTextEncodingGlobals(rt *goja.Runtime) {
	ensureTextEncoderGlobal(rt)
	ensureTextDecoderGlobal(rt)
}

func ensureTextEncoderGlobal(rt *goja.Runtime) {
	if v := rt.Get("TextEncoder"); isValuePresent(v) {
		return
	}
	_ = rt.Set("TextEncoder", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This
		_ = obj.Set("encoding", "utf-8")
		_ = obj.Set("encode", func(fc goja.FunctionCall) goja.Value {
			input := ""
			if v := fc.Argument(0); isValuePresent(v) {
				input = v.String()
			}
			return bytesToUint8Array(rt, []byte(input))
		})
		return obj
	})
}

func ensureTextDecoderGlobal(rt *goja.Runtime) {
	if v := rt.Get("TextDecoder"); isValuePresent(v) {
		return
	}
	_ = rt.Set("TextDecoder", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This
		label := "utf-8"
		if v := call.Argument(0); isValuePresent(v) {
			s := strings.TrimSpace(strings.ToLower(v.String()))
			if s != "" && s != "utf-8" && s != "utf8" {
				panic(rt.NewTypeError("TextDecoder only supports utf-8"))
			}
		}
		_ = obj.Set("encoding", label)
		_ = obj.Set("fatal", false)
		_ = obj.Set("ignoreBOM", false)
		_ = obj.Set("decode", func(fc goja.FunctionCall) goja.Value {
			input := fc.Argument(0)
			if goja.IsUndefined(input) || goja.IsNull(input) {
				return rt.ToValue("")
			}
			data, err := bufferSourceBytes(rt, input, true, false)
			if err != nil {
				panic(rt.NewTypeError("TextDecoder.decode: " + err.Error()))
			}
			return rt.ToValue(string(data))
		})
		return obj
	})
}

func bytesToUint8Array(rt *goja.Runtime, data []byte) goja.Value {
	ctor, ok := goja.AssertConstructor(rt.Get("Uint8Array"))
	if !ok {
		ab := rt.NewArrayBuffer(data)
		return rt.ToValue(ab)
	}
	buf := make([]byte, len(data))
	copy(buf, data)
	typed, err := ctor(nil, rt.ToValue(rt.NewArrayBuffer(buf)))
	if err != nil {
		panic(rt.NewGoError(errors.New("failed to construct Uint8Array")))
	}
	return typed
}
