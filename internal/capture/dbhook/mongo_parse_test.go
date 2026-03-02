package dbhook

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"shadiff/internal/model"
)

// buildBSONString creates a BSON UTF-8 string element: type(1) + key(cstring) + strlen(4) + value + null
func buildBSONString(key, value string) []byte {
	var b []byte
	b = append(b, 0x02) // type: string
	b = append(b, []byte(key)...)
	b = append(b, 0x00) // null terminator for key
	strBytes := append([]byte(value), 0x00)
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(strBytes)))
	b = append(b, lenBuf...)
	b = append(b, strBytes...)
	return b
}

// buildBSONInt32 creates a BSON int32 element: type(1) + key(cstring) + int32(4)
func buildBSONInt32(key string, val int32) []byte {
	var b []byte
	b = append(b, 0x10) // type: int32
	b = append(b, []byte(key)...)
	b = append(b, 0x00) // null terminator for key
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(val))
	b = append(b, buf...)
	return b
}

// buildBSONDoc wraps elements into a BSON document: length(4) + elements + null terminator(1)
func buildBSONDoc(elements ...[]byte) []byte {
	var body []byte
	for _, elem := range elements {
		body = append(body, elem...)
	}
	body = append(body, 0x00) // document null terminator
	docLen := 4 + len(body)
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(docLen))
	return append(lenBuf, body...)
}

func TestSimpleBSONToMap_ValidStringDocument(t *testing.T) {
	doc := buildBSONDoc(
		buildBSONString("find", "users"),
		buildBSONString("$db", "mydb"),
	)

	result := simpleBSONToMap(doc)
	if result == nil {
		t.Fatal("expected non-nil map")
	}

	if v, ok := result["find"]; !ok {
		t.Fatal("expected key 'find'")
	} else if v != "users" {
		t.Fatalf("expected 'users', got %v", v)
	}

	if v, ok := result["$db"]; !ok {
		t.Fatal("expected key '$db'")
	} else if v != "mydb" {
		t.Fatalf("expected 'mydb', got %v", v)
	}
}

func TestSimpleBSONToMap_Int32(t *testing.T) {
	doc := buildBSONDoc(
		buildBSONString("count", "orders"),
		buildBSONInt32("limit", 42),
	)

	result := simpleBSONToMap(doc)
	if result == nil {
		t.Fatal("expected non-nil map")
	}

	if v, ok := result["limit"]; !ok {
		t.Fatal("expected key 'limit'")
	} else if v != int(42) {
		t.Fatalf("expected 42, got %v (type %T)", v, v)
	}
}

func TestSimpleBSONToMap_Nil(t *testing.T) {
	result := simpleBSONToMap(nil)
	if result != nil {
		t.Fatalf("expected nil for nil input, got %v", result)
	}
}

func TestSimpleBSONToMap_Empty(t *testing.T) {
	result := simpleBSONToMap([]byte{})
	if result != nil {
		t.Fatalf("expected nil for empty input, got %v", result)
	}
}

func TestSimpleBSONToMap_TooShort(t *testing.T) {
	result := simpleBSONToMap([]byte{0x01, 0x02, 0x03})
	if result != nil {
		t.Fatalf("expected nil for short input, got %v", result)
	}
}

func TestSimpleBSONToMap_InvalidLength(t *testing.T) {
	// Document claims length 100 but actual data is only 5 bytes
	data := make([]byte, 5)
	binary.LittleEndian.PutUint32(data[0:4], 100)
	data[4] = 0x00

	result := simpleBSONToMap(data)
	if result != nil {
		t.Fatalf("expected nil for invalid length, got %v", result)
	}
}

func TestSimpleBSONToMap_EmptyDocument(t *testing.T) {
	// Minimal valid BSON: length 5 = 4 (length) + 1 (null terminator)
	doc := buildBSONDoc()
	result := simpleBSONToMap(doc)
	if result == nil {
		t.Fatal("expected non-nil map for empty document")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty map, got %v", result)
	}
}

func TestSimpleBSONToMap_Boolean(t *testing.T) {
	// Build boolean element manually: type(0x08) + key + null + value(1 byte)
	var elem []byte
	elem = append(elem, 0x08) // type: boolean
	elem = append(elem, []byte("active")...)
	elem = append(elem, 0x00) // null terminator for key
	elem = append(elem, 0x01) // true

	doc := buildBSONDoc(elem)
	result := simpleBSONToMap(doc)
	if result == nil {
		t.Fatal("expected non-nil map")
	}
	if v, ok := result["active"]; !ok {
		t.Fatal("expected key 'active'")
	} else if v != true {
		t.Fatalf("expected true, got %v", v)
	}
}

func TestSimpleBSONToMap_Null(t *testing.T) {
	// Build null element: type(0x0A) + key + null terminator
	var elem []byte
	elem = append(elem, 0x0A) // type: null
	elem = append(elem, []byte("empty")...)
	elem = append(elem, 0x00)

	doc := buildBSONDoc(elem)
	result := simpleBSONToMap(doc)
	if result == nil {
		t.Fatal("expected non-nil map")
	}
	if _, ok := result["empty"]; !ok {
		t.Fatal("expected key 'empty'")
	}
	if result["empty"] != nil {
		t.Fatalf("expected nil value, got %v", result["empty"])
	}
}

func TestSimpleBSONToMap_SubDocument(t *testing.T) {
	// Build a sub-document element (type 0x03)
	subDoc := buildBSONDoc(
		buildBSONString("name", "test"),
	)

	var elem []byte
	elem = append(elem, 0x03) // type: document
	elem = append(elem, []byte("filter")...)
	elem = append(elem, 0x00)
	elem = append(elem, subDoc...)

	doc := buildBSONDoc(elem)
	result := simpleBSONToMap(doc)
	if result == nil {
		t.Fatal("expected non-nil map")
	}

	sub, ok := result["filter"]
	if !ok {
		t.Fatal("expected key 'filter'")
	}
	subMap, ok := sub.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", sub)
	}
	if subMap["name"] != "test" {
		t.Fatalf("expected sub-document name='test', got %v", subMap["name"])
	}
}

func TestMongoCommandToJSON_BasicCommand(t *testing.T) {
	effect := model.SideEffect{
		Type:       model.SideEffectDB,
		DBType:     "mongo",
		Operation:  "find",
		Collection: "users",
		Database:   "testdb",
	}

	result := MongoCommandToJSON(effect)
	if result == "" {
		t.Fatal("expected non-empty JSON string")
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("expected valid JSON, got error: %v", err)
	}

	if parsed["operation"] != "find" {
		t.Fatalf("expected operation 'find', got %v", parsed["operation"])
	}
	if parsed["collection"] != "users" {
		t.Fatalf("expected collection 'users', got %v", parsed["collection"])
	}
	if parsed["database"] != "testdb" {
		t.Fatalf("expected database 'testdb', got %v", parsed["database"])
	}
}

func TestMongoCommandToJSON_WithFilter(t *testing.T) {
	effect := model.SideEffect{
		Type:       model.SideEffectDB,
		DBType:     "mongo",
		Operation:  "find",
		Collection: "users",
		Database:   "testdb",
		Filter:     map[string]any{"age": 25},
	}

	result := MongoCommandToJSON(effect)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("expected valid JSON, got error: %v", err)
	}

	if parsed["filter"] == nil {
		t.Fatal("expected filter in JSON output")
	}
}

func TestMongoCommandToJSON_WithNilOptionalFields(t *testing.T) {
	effect := model.SideEffect{
		Operation:  "insert",
		Collection: "logs",
		Database:   "app",
	}

	result := MongoCommandToJSON(effect)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("expected valid JSON, got error: %v", err)
	}

	// filter, update, documents should be absent when nil
	if _, ok := parsed["filter"]; ok {
		t.Fatal("expected no 'filter' key when Filter is nil")
	}
	if _, ok := parsed["update"]; ok {
		t.Fatal("expected no 'update' key when Update is nil")
	}
	if _, ok := parsed["documents"]; ok {
		t.Fatal("expected no 'documents' key when Documents is nil")
	}
}
