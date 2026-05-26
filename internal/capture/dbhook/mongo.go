package dbhook

import (
	"encoding/binary"
	"encoding/json"
	"time"

	"shadiff/internal/dbtype"
	"shadiff/internal/model"
)

// MongoHook is a MongoDB protocol proxy that parses the OP_MSG Wire Protocol.
type MongoHook struct {
	*tcpProxy
}

// MongoDB Wire Protocol constants.
const (
	opMsgOpCode        = 2013 // OP_MSG
	maxMongoMessageLen = 16 * 1024 * 1024
)

type mongoParser struct {
	buf []byte
}

func NewMongoHook(listenAddr, targetAddr string) *MongoHook {
	return &MongoHook{
		tcpProxy: newTCPProxy(dbtype.Mongo, listenAddr, targetAddr, func() protocolParser {
			return &mongoParser{}
		}),
	}
}

func (p *mongoParser) Feed(data []byte) []model.SideEffect {
	p.buf = append(p.buf, data...)

	var effects []model.SideEffect
	for len(p.buf) >= 16 {
		msgLen := int(binary.LittleEndian.Uint32(p.buf[0:4]))
		if msgLen < 16 || msgLen > maxMongoMessageLen {
			p.buf = nil
			return effects
		}
		if len(p.buf) < msgLen {
			break
		}

		opCode := int(binary.LittleEndian.Uint32(p.buf[12:16]))
		if opCode == opMsgOpCode {
			effects = append(effects, parseOpMsg(p.buf[16:msgLen])...)
		}
		p.buf = p.buf[msgLen:]
	}

	if len(p.buf) > maxMongoMessageLen {
		p.buf = nil
	}
	return effects
}

// parseOpMsg parses a MongoDB OP_MSG body.
func parseOpMsg(body []byte) []model.SideEffect {
	if len(body) < 5 {
		return nil
	}

	var effects []model.SideEffect
	offset := 4 // flagBits
	for offset < len(body) {
		kind := body[offset]
		offset++

		switch kind {
		case 0: // Body section (single BSON document)
			if offset+4 > len(body) {
				return effects
			}
			docLen := int(binary.LittleEndian.Uint32(body[offset : offset+4]))
			if docLen < 5 || offset+docLen > len(body) {
				return effects
			}
			if effect, ok := extractMongoCommand(body[offset : offset+docLen]); ok {
				effects = append(effects, effect)
			}
			offset += docLen
		case 1: // Document Sequence section
			if offset+4 > len(body) {
				return effects
			}
			secLen := int(binary.LittleEndian.Uint32(body[offset : offset+4]))
			if secLen < 4 || offset+secLen > len(body) {
				return effects
			}
			offset += secLen
		default:
			return effects
		}
	}

	return effects
}

// extractMongoCommand extracts MongoDB command information from a BSON document.
func extractMongoCommand(bsonDoc []byte) (model.SideEffect, bool) {
	doc := simpleBSONToMap(bsonDoc)
	if doc == nil {
		return model.SideEffect{}, false
	}

	payload := model.MongoSideEffect{}
	if db, ok := doc["$db"]; ok {
		if dbStr, ok := db.(string); ok {
			payload.Database = dbStr
		}
	}

	mongoCommands := []string{"find", "insert", "update", "delete", "aggregate", "count", "distinct", "findAndModify"}
	for _, cmd := range mongoCommands {
		if coll, ok := doc[cmd]; ok {
			payload.Operation = cmd
			if collStr, ok := coll.(string); ok {
				payload.Collection = collStr
			}
			break
		}
	}
	if payload.Operation == "" {
		return model.SideEffect{}, false
	}

	if filter, ok := doc["filter"]; ok {
		payload.Filter = filter
	}
	if update, ok := doc["updates"]; ok {
		payload.Update = update
	}
	if docs, ok := doc["documents"]; ok {
		payload.Documents = docs
	}

	return model.NewMongoSideEffect(payload, time.Now().UnixMilli()), true
}

// simpleBSONToMap is a simplified BSON parser that extracts JSON-friendly values.
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
			if subMap := simpleBSONToMap(data[offset : offset+subDocLen]); subMap != nil {
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
			offset += 8
		case 0x08: // boolean
			if offset >= len(data) {
				return result
			}
			result[key] = data[offset] != 0
			offset++
		case 0x0A: // null
			result[key] = nil
		case 0x07: // ObjectId
			if offset+12 > len(data) {
				return result
			}
			offset += 12
		default:
			return result
		}
	}

	return result
}

// MongoCommandToJSON converts a MongoDB command to readable JSON for logging and reporting.
func MongoCommandToJSON(effect model.SideEffect) string {
	payload := effect.Mongo()
	if payload == nil {
		payload = &model.MongoSideEffect{}
	}
	cmd := map[string]any{
		"operation":  payload.Operation,
		"collection": payload.Collection,
		"database":   payload.Database,
	}
	if payload.Filter != nil {
		cmd["filter"] = payload.Filter
	}
	if payload.Update != nil {
		cmd["update"] = payload.Update
	}
	if payload.Documents != nil {
		cmd["documents"] = payload.Documents
	}
	data, _ := json.Marshal(cmd)
	return string(data)
}
