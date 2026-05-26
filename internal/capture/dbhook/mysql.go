package dbhook

import (
	"encoding/binary"
	"time"

	"shadiff/internal/dbtype"
	"shadiff/internal/model"
)

// MySQLHook is a MySQL protocol proxy that parses COM_QUERY packets to capture SQL statements.
type MySQLHook struct {
	*tcpProxy
}

// MySQL protocol constants.
const (
	mysqlComQuery       = 0x03
	mysqlComStmtPrepare = 0x16
	mysqlComStmtExecute = 0x17
	maxMySQLPacketSize  = 16 * 1024 * 1024
)

type mysqlParser struct {
	buf []byte
}

func NewMySQLHook(listenAddr, targetAddr string) *MySQLHook {
	return &MySQLHook{
		tcpProxy: newTCPProxy(dbtype.MySQL, listenAddr, targetAddr, func() protocolParser {
			return &mysqlParser{}
		}),
	}
}

func (p *mysqlParser) Feed(data []byte) []model.SideEffect {
	p.buf = append(p.buf, data...)

	var effects []model.SideEffect
	for len(p.buf) >= 4 {
		payloadLen := int(uint32(p.buf[0]) | uint32(p.buf[1])<<8 | uint32(p.buf[2])<<16)
		if payloadLen < 1 {
			p.buf = p.buf[4:]
			continue
		}
		if payloadLen > maxMySQLPacketSize {
			p.buf = nil
			return effects
		}
		packetLen := 4 + payloadLen
		if len(p.buf) < packetLen {
			break
		}

		commandByte := p.buf[4]
		payload := p.buf[5:packetLen]
		switch commandByte {
		case mysqlComQuery, mysqlComStmtPrepare:
			effects = append(effects, model.NewSQLSideEffect(dbtype.MySQL, string(payload), time.Now().UnixMilli()))
		}

		p.buf = p.buf[packetLen:]
	}

	if len(p.buf) > maxMySQLPacketSize {
		p.buf = nil
	}
	return effects
}

// readMySQLPacketLength reads the MySQL packet length (helper function).
func readMySQLPacketLength(data []byte) int {
	if len(data) < 3 {
		return 0
	}
	return int(binary.LittleEndian.Uint32(append(data[:3], 0)))
}
