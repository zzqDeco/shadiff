package dbhook

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"sync"
	"time"

	"shadiff/internal/logger"
	"shadiff/internal/model"
)

// MongoHook is a MongoDB protocol proxy that parses the OP_MSG Wire Protocol
type MongoHook struct {
	listenAddr  string
	targetAddr  string
	listener    net.Listener
	sideEffects chan model.SideEffect
	done        chan struct{}
	wg          sync.WaitGroup
}

// MongoDB Wire Protocol constants
const (
	opMsgOpCode = 2013 // OP_MSG
)

func NewMongoHook(listenAddr, targetAddr string) *MongoHook {
	return &MongoHook{
		listenAddr:  listenAddr,
		targetAddr:  targetAddr,
		sideEffects: make(chan model.SideEffect, 1000),
		done:        make(chan struct{}),
	}
}

func (h *MongoHook) Type() string { return "mongo" }

func (h *MongoHook) SideEffects() <-chan model.SideEffect {
	return h.sideEffects
}

func (h *MongoHook) Start(ctx context.Context) error {
	var err error
	h.listener, err = net.Listen("tcp", h.listenAddr)
	if err != nil {
		return err
	}

	logger.DBHookEvent("started", "mongo", "listen", h.listenAddr, "target", h.targetAddr)

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		for {
			select {
			case <-h.done:
				return
			default:
			}

			conn, err := h.listener.Accept()
			if err != nil {
				select {
				case <-h.done:
					return
				default:
					logger.Error("mongo accept error", err)
					continue
				}
			}

			h.wg.Add(1)
			go func() {
				defer h.wg.Done()
				h.handleConn(conn)
			}()
		}
	}()

	return nil
}

func (h *MongoHook) Stop() error {
	close(h.done)
	if h.listener != nil {
		h.listener.Close()
	}
	h.wg.Wait()
	close(h.sideEffects)
	return nil
}

func (h *MongoHook) handleConn(clientConn net.Conn) {
	defer clientConn.Close()

	serverConn, err := net.DialTimeout("tcp", h.targetAddr, 10*time.Second)
	if err != nil {
		logger.Error("mongo connect target failed", err)
		return
	}
	defer serverConn.Close()

	var wg sync.WaitGroup

	// Server -> Client (passthrough)
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(clientConn, serverConn)
	}()

	// Client -> Server (sniff)
	wg.Add(1)
	go func() {
		defer wg.Done()
		h.sniffClientToServer(clientConn, serverConn)
	}()

	wg.Wait()
}

func (h *MongoHook) sniffClientToServer(client, server net.Conn) {
	for {
		// MongoDB Wire Protocol header: 4-byte message length (little-endian)
		header := make([]byte, 16)
		if _, err := io.ReadFull(client, header); err != nil {
			return
		}

		msgLen := int(binary.LittleEndian.Uint32(header[0:4]))
		// requestID := binary.LittleEndian.Uint32(header[4:8])
		// responseTo := binary.LittleEndian.Uint32(header[8:12])
		opCode := int(binary.LittleEndian.Uint32(header[12:16]))

		// Read remaining message body
		remaining := msgLen - 16
		if remaining < 0 || remaining > 16*1024*1024 {
			// Invalid message length, forward header then fall back to passthrough
			server.Write(header)
			io.Copy(server, client)
			return
		}

		body := make([]byte, remaining)
		if _, err := io.ReadFull(client, body); err != nil {
			server.Write(header)
			return
		}

		// Forward the complete message
		server.Write(header)
		server.Write(body)

		// Try to parse OP_MSG
		if opCode == opMsgOpCode {
			h.parseOpMsg(body)
		}
	}
}

// parseOpMsg parses a MongoDB OP_MSG message
func (h *MongoHook) parseOpMsg(body []byte) {
	if len(body) < 5 {
		return
	}

	// flagBits (4 bytes) + sections
	// _ = binary.LittleEndian.Uint32(body[0:4])
	offset := 4

	for offset < len(body) {
		if offset >= len(body) {
			break
		}

		kind := body[offset]
		offset++

		switch kind {
		case 0: // Body section (single BSON document)
			if offset+4 > len(body) {
				return
			}
			docLen := int(binary.LittleEndian.Uint32(body[offset : offset+4]))
			if docLen < 5 || offset+docLen > len(body) {
				return
			}

			doc := body[offset : offset+docLen]
			h.extractMongoCommand(doc)
			offset += docLen

		case 1: // Document Sequence section
			if offset+4 > len(body) {
				return
			}
			secLen := int(binary.LittleEndian.Uint32(body[offset : offset+4]))
			if secLen < 4 || offset+secLen > len(body) {
				return
			}
			offset += secLen

		default:
			return
		}
	}
}

// extractMongoCommand extracts MongoDB command information from a BSON document.
// Simplified implementation: converts BSON to JSON-parseable format to extract command type and parameters.
func (h *MongoHook) extractMongoCommand(bsonDoc []byte) {
	// Simplified handling: try basic BSON parsing to extract the first key-value pair.
	// Full BSON parsing requires the bson library; here we do basic extraction.
	doc := simpleBSONToMap(bsonDoc)
	if doc == nil {
		return
	}

	effect := model.SideEffect{
		Type:      model.SideEffectDB,
		DBType:    "mongo",
		Timestamp: time.Now().UnixMilli(),
	}

	// Extract database name
	if db, ok := doc["$db"]; ok {
		if dbStr, ok := db.(string); ok {
			effect.Database = dbStr
		}
	}

	// Identify command type and collection name
	mongoCommands := []string{"find", "insert", "update", "delete", "aggregate", "count", "distinct", "findAndModify"}
	for _, cmd := range mongoCommands {
		if coll, ok := doc[cmd]; ok {
			effect.Operation = cmd
			if collStr, ok := coll.(string); ok {
				effect.Collection = collStr
			}
			break
		}
	}

	if effect.Operation == "" {
		return // Not a CRUD command, skip
	}

	// Extract filter conditions
	if filter, ok := doc["filter"]; ok {
		effect.Filter = filter
	}

	// Extract update operations
	if update, ok := doc["updates"]; ok {
		effect.Update = update
	}

	// Extract inserted documents
	if docs, ok := doc["documents"]; ok {
		effect.Documents = docs
	}

	select {
	case h.sideEffects <- effect:
	default:
		logger.Warn("mongo side effect channel full, dropping")
	}
}

// simpleBSONToMap is a simplified BSON parser that extracts key-value pairs with string type.
// A full implementation should use go.mongodb.org/mongo-driver/bson; this does basic extraction.
func simpleBSONToMap(data []byte) map[string]any {
	if len(data) < 5 {
		return nil
	}

	result := make(map[string]any)
	docLen := int(binary.LittleEndian.Uint32(data[0:4]))
	if docLen > len(data) {
		return nil
	}

	offset := 4
	for offset < docLen-1 {
		if offset >= len(data) {
			break
		}

		elemType := data[offset]
		offset++

		// Read key (C string)
		keyEnd := offset
		for keyEnd < len(data) && data[keyEnd] != 0 {
			keyEnd++
		}
		if keyEnd >= len(data) {
			break
		}
		key := string(data[offset:keyEnd])
		offset = keyEnd + 1

		switch elemType {
		case 0x02: // UTF-8 string
			if offset+4 > len(data) {
				return result
			}
			strLen := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
			offset += 4
			if offset+strLen > len(data) || strLen < 1 {
				return result
			}
			result[key] = string(data[offset : offset+strLen-1])
			offset += strLen

		case 0x03, 0x04: // Document or Array
			if offset+4 > len(data) {
				return result
			}
			subDocLen := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
			if offset+subDocLen > len(data) {
				return result
			}
			// Try to convert sub-document to a JSON-friendly format
			subMap := simpleBSONToMap(data[offset : offset+subDocLen])
			if subMap != nil {
				result[key] = subMap
			}
			offset += subDocLen

		case 0x10: // int32
			if offset+4 > len(data) {
				return result
			}
			result[key] = int(binary.LittleEndian.Uint32(data[offset : offset+4]))
			offset += 4

		case 0x12: // int64
			if offset+8 > len(data) {
				return result
			}
			result[key] = int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
			offset += 8

		case 0x01: // double
			if offset+8 > len(data) {
				return result
			}
			offset += 8 // skip double

		case 0x08: // boolean
			if offset >= len(data) {
				return result
			}
			result[key] = data[offset] != 0
			offset++

		case 0x0A: // null
			result[key] = nil

		case 0x07: // ObjectId (12 bytes)
			if offset+12 > len(data) {
				return result
			}
			offset += 12

		default:
			// Unknown type, cannot continue parsing
			return result
		}
	}

	return result
}

// MongoCommandToJSON converts a MongoDB command to a readable JSON string (for logging and reporting)
func MongoCommandToJSON(effect model.SideEffect) string {
	cmd := map[string]any{
		"operation":  effect.Operation,
		"collection": effect.Collection,
		"database":   effect.Database,
	}
	if effect.Filter != nil {
		cmd["filter"] = effect.Filter
	}
	if effect.Update != nil {
		cmd["update"] = effect.Update
	}
	if effect.Documents != nil {
		cmd["documents"] = effect.Documents
	}
	data, _ := json.Marshal(cmd)
	return string(data)
}
