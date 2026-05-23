package orchestrator

import (
	"testing"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

func schema(fields map[string]asmtypes.FieldDef) map[string]asmtypes.FieldDef { return fields }

func TestBlackboard_DefaultValues(t *testing.T) {
	bb := NewBlackboard(schema(map[string]asmtypes.FieldDef{
		"status": {Type: "string", Default: "pending"},
		"score":  {Type: "number", Default: float64(0)},
	}), nil)

	data := bb.Read()
	if data["status"] != "pending" {
		t.Errorf("status default: got %v, want %q", data["status"], "pending")
	}
	if data["score"] != float64(0) {
		t.Errorf("score default: got %v, want 0", data["score"])
	}
}

func TestBlackboard_InitialValuesOverrideDefaults(t *testing.T) {
	bb := NewBlackboard(schema(map[string]asmtypes.FieldDef{
		"status": {Type: "string", Default: "pending"},
	}), map[string]interface{}{"status": "active"})

	if v, _ := bb.Get("status"); v != "active" {
		t.Errorf("expected initial value 'active', got %v", v)
	}
}

func TestBlackboard_Write_Valid(t *testing.T) {
	bb := NewBlackboard(schema(map[string]asmtypes.FieldDef{
		"name": {Type: "string"},
	}), nil)

	if err := bb.Write(map[string]interface{}{"name": "alice"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if v, _ := bb.Get("name"); v != "alice" {
		t.Errorf("got %v, want %q", v, "alice")
	}
}

func TestBlackboard_Write_TypeMismatch(t *testing.T) {
	bb := NewBlackboard(schema(map[string]asmtypes.FieldDef{
		"count": {Type: "number"},
	}), nil)

	err := bb.Write(map[string]interface{}{"count": "not-a-number"})
	if err == nil {
		t.Fatal("expected type error, got nil")
	}
}

func TestBlackboard_Write_BoolType(t *testing.T) {
	bb := NewBlackboard(schema(map[string]asmtypes.FieldDef{
		"approved": {Type: "bool"},
	}), nil)

	if err := bb.Write(map[string]interface{}{"approved": true}); err != nil {
		t.Fatalf("Write bool: %v", err)
	}
	// wrong type
	if err := bb.Write(map[string]interface{}{"approved": "yes"}); err == nil {
		t.Fatal("expected type error for string->bool, got nil")
	}
}

func TestBlackboard_Write_ExtraFieldsAllowed(t *testing.T) {
	// Fields not in schema should be accepted (agents may write metadata).
	bb := NewBlackboard(schema(map[string]asmtypes.FieldDef{
		"status": {Type: "string"},
	}), nil)

	if err := bb.Write(map[string]interface{}{"extra_key": "surprise"}); err != nil {
		t.Fatalf("expected extra fields to be allowed, got: %v", err)
	}
	if v, ok := bb.Get("extra_key"); !ok || v != "surprise" {
		t.Errorf("extra_key: got %v/%v, want 'surprise'/true", v, ok)
	}
}

func TestBlackboard_RequiredField_Missing(t *testing.T) {
	bb := NewBlackboard(schema(map[string]asmtypes.FieldDef{
		"invoice_id": {Type: "string", Required: true},
	}), nil)

	// Writing an unrelated key should still fail because invoice_id is required and absent.
	err := bb.Write(map[string]interface{}{"other": "value"})
	if err == nil {
		t.Fatal("expected error for missing required field, got nil")
	}
}

func TestBlackboard_RequiredField_Present(t *testing.T) {
	bb := NewBlackboard(schema(map[string]asmtypes.FieldDef{
		"invoice_id": {Type: "string", Required: true},
	}), map[string]interface{}{"invoice_id": "INV-001"})

	// With required field present, write should succeed.
	if err := bb.Write(map[string]interface{}{"extra": "ok"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBlackboard_Read_ReturnsCopy(t *testing.T) {
	bb := NewBlackboard(nil, map[string]interface{}{"key": "original"})
	data := bb.Read()
	data["key"] = "mutated"

	data2 := bb.Read()
	if data2["key"] != "original" {
		t.Error("Read() must return a copy — mutation affected stored data")
	}
}

func TestBlackboard_Get_MissingKey(t *testing.T) {
	bb := NewBlackboard(nil, nil)
	_, ok := bb.Get("absent")
	if ok {
		t.Error("Get should return ok=false for absent key")
	}
}

func TestBlackboard_NumberTypes(t *testing.T) {
	bb := NewBlackboard(schema(map[string]asmtypes.FieldDef{
		"n": {Type: "number"},
	}), nil)

	numeric := []interface{}{
		int(1), int8(1), int16(1), int32(1), int64(1),
		uint(1), uint8(1), uint16(1), uint32(1), uint64(1),
		float32(1.5), float64(1.5),
	}
	for _, v := range numeric {
		if err := bb.Write(map[string]interface{}{"n": v}); err != nil {
			t.Errorf("Write(%T): unexpected error: %v", v, err)
		}
	}
}
