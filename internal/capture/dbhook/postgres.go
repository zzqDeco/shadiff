package dbhook

import (
	"encoding/binary"
	"time"

	"shadiff/internal/dbtype"
	"shadiff/internal/model"
)

// PostgresHook is a PostgreSQL protocol proxy that parses Simple/Extended Query messages.
type PostgresHook struct {
	*tcpProxy
}

// PostgreSQL frontend message types.
const (
	pgMsgQuery            = 'Q' // Simple Query
	pgMsgParse            = 'P' // Extended Query: Parse
	maxPostgresMessageLen = 16 * 1024 * 1024
)

type postgresParser struct {
	buf     []byte
	startup bool
}

func NewPostgresHook(listenAddr, targetAddr string) *PostgresHook {
	return &PostgresHook{
		tcpProxy: newTCPProxy(dbtype.Postgres, listenAddr, targetAddr, func() protocolParser {
			return &postgresParser{startup: true}
		}),
	}
}

func (p *postgresParser) Feed(data []byte) []model.SideEffect {
	p.buf = append(p.buf, data...)

	var effects []model.SideEffect
	for {
		if p.startup {
			if len(p.buf) < 4 {
				break
			}
			msgLen := int(binary.BigEndian.Uint32(p.buf[:4]))
			if msgLen < 8 || msgLen > maxPostgresMessageLen {
				p.buf = nil
				p.startup = false
				break
			}
			if len(p.buf) < msgLen {
				break
			}
			p.buf = p.buf[msgLen:]
			p.startup = false
			continue
		}

		if len(p.buf) < 5 {
			break
		}

		msgType := p.buf[0]
		msgLen := int(binary.BigEndian.Uint32(p.buf[1:5]))
		if msgLen < 4 || msgLen > maxPostgresMessageLen {
			p.buf = nil
			break
		}
		fullLen := 1 + msgLen
		if len(p.buf) < fullLen {
			break
		}

		payload := p.buf[5:fullLen]
		switch msgType {
		case pgMsgQuery:
			query := extractNullTermString(payload)
			if query != "" {
				effects = append(effects, model.NewSQLSideEffect(dbtype.Postgres, query, time.Now().UnixMilli()))
			}
		case pgMsgParse:
			idx := nullTermIndex(payload)
			if idx >= 0 && idx+1 < len(payload) {
				query := extractNullTermString(payload[idx+1:])
				if query != "" {
					effects = append(effects, model.NewSQLSideEffect(dbtype.Postgres, query, time.Now().UnixMilli()))
				}
			}
		}

		p.buf = p.buf[fullLen:]
	}

	if len(p.buf) > maxPostgresMessageLen {
		p.buf = nil
	}
	return effects
}

func extractNullTermString(data []byte) string {
	for i, b := range data {
		if b == 0 {
			return string(data[:i])
		}
	}
	return string(data)
}

func nullTermIndex(data []byte) int {
	for i, b := range data {
		if b == 0 {
			return i
		}
	}
	return -1
}
