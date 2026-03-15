package logbench

import (
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"testing"
)

func jsonRoundtrip(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(encodeValue(v))
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	return result
}

func TestEncodeValue_Primitives(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want any
	}{
		{"string", "hello", "hello"},
		{"int", 42, 42},
		{"float64", 3.14, 3.14},
		{"bool", true, true},
		{"nil", nil, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := encodeValue(tc.in)
			b1, _ := json.Marshal(got)
			b2, _ := json.Marshal(tc.want)
			if string(b1) != string(b2) {
				t.Errorf("encodeValue(%v) = %s, want %s", tc.in, b1, b2)
			}
		})
	}
}

func TestEncodeValue_NaN(t *testing.T) {
	result := jsonRoundtrip(t, math.NaN())
	if result["_type"] != "@go/NaN" {
		t.Errorf("expected @go/NaN, got %v", result["_type"])
	}
}

func TestEncodeValue_Infinity(t *testing.T) {
	pos := jsonRoundtrip(t, math.Inf(1))
	if pos["_type"] != "@go/Infinity" || pos["_value"] != float64(1) {
		t.Errorf("expected @go/Infinity +1, got %v", pos)
	}

	neg := jsonRoundtrip(t, math.Inf(-1))
	if neg["_type"] != "@go/Infinity" || neg["_value"] != float64(-1) {
		t.Errorf("expected @go/Infinity -1, got %v", neg)
	}
}

func TestEncodeValue_Complex(t *testing.T) {
	result := jsonRoundtrip(t, complex(1, 2))
	if result["_type"] != "@go/complex128" {
		t.Errorf("expected @go/complex128, got %v", result["_type"])
	}

	result32 := jsonRoundtrip(t, complex64(complex(1, 2)))
	if result32["_type"] != "@go/complex64" {
		t.Errorf("expected @go/complex64, got %v", result32["_type"])
	}
}

func TestEncodeValue_Error(t *testing.T) {
	err := errors.New("something broke")
	result := jsonRoundtrip(t, err)
	if result["_type"] != "@go/error" {
		t.Errorf("expected @go/error, got %v", result["_type"])
	}
	val := result["_value"].(map[string]any)
	if val["message"] != "something broke" {
		t.Errorf("expected message 'something broke', got %v", val["message"])
	}
}

func TestEncodeValue_Function(t *testing.T) {
	fn := func() {}
	result := jsonRoundtrip(t, fn)
	if result["_type"] != "@go/Function" {
		t.Errorf("expected @go/Function, got %v", result["_type"])
	}
}

func TestEncodeValue_BigInt(t *testing.T) {
	bi := big.NewInt(123456789012345)
	result := jsonRoundtrip(t, bi)
	if result["_type"] != "@go/big.Int" {
		t.Errorf("expected @go/big.Int, got %v", result["_type"])
	}
	if result["_value"] != "123456789012345" {
		t.Errorf("expected value '123456789012345', got %v", result["_value"])
	}
}

func TestEncodeValue_BigFloat(t *testing.T) {
	bf := big.NewFloat(3.14159)
	result := jsonRoundtrip(t, bf)
	if result["_type"] != "@go/big.Float" {
		t.Errorf("expected @go/big.Float, got %v", result["_type"])
	}
}

func TestEncodeValue_Bytes(t *testing.T) {
	result := jsonRoundtrip(t, []byte("hello"))
	if result["_type"] != "@go/bytes" {
		t.Errorf("expected @go/bytes, got %v", result["_type"])
	}
	if result["_value"] != "aGVsbG8=" {
		t.Errorf("expected base64 'aGVsbG8=', got %v", result["_value"])
	}
}

func TestEncodeValue_Chan(t *testing.T) {
	ch := make(chan int)
	result := jsonRoundtrip(t, ch)
	if result["_type"] != "@go/chan" {
		t.Errorf("expected @go/chan, got %v", result["_type"])
	}
}

type testStruct struct {
	Name    string
	Age     int
	private string
}

func TestEncodeValue_Struct(t *testing.T) {
	s := testStruct{Name: "Alice", Age: 30, private: "hidden"}
	result := jsonRoundtrip(t, s)
	if result["_type"] != "@go/Struct" {
		t.Errorf("expected @go/Struct, got %v", result["_type"])
	}
	val := result["_value"].(map[string]any)
	if val["__name__"] != "testStruct" {
		t.Errorf("expected __name__ 'testStruct', got %v", val["__name__"])
	}
	if val["Name"] != "Alice" {
		t.Errorf("expected Name 'Alice', got %v", val["Name"])
	}
	if _, ok := val["private"]; ok {
		t.Error("unexported field should not be included")
	}
}

func TestPrepareContent_Single(t *testing.T) {
	result := prepareContent([]any{"hello"})
	if result != "hello" {
		t.Errorf("single content should unwrap, got %v", result)
	}
}

func TestPrepareContent_Multiple(t *testing.T) {
	result := prepareContent([]any{"hello", 42})
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("multiple content should be array, got %T", result)
	}
	if len(arr) != 2 {
		t.Errorf("expected 2 items, got %d", len(arr))
	}
}

func TestEncodeValue_Map(t *testing.T) {
	m := map[string]any{"key": math.NaN()}
	result := encodeValue(m)
	rm := result.(map[string]any)
	inner, ok := rm["key"].(typedEnvelope)
	if !ok || inner.Type != "@go/NaN" {
		t.Errorf("expected nested NaN encoding, got %v", rm["key"])
	}
}

func TestEncodeValue_Slice(t *testing.T) {
	s := []any{1, math.Inf(1), "hello"}
	result := encodeValue(s)
	rs := result.([]any)
	if len(rs) != 3 {
		t.Fatalf("expected 3 items, got %d", len(rs))
	}
	env, ok := rs[1].(typedEnvelope)
	if !ok || env.Type != "@go/Infinity" {
		t.Errorf("expected Infinity encoding at index 1, got %v", rs[1])
	}
}
