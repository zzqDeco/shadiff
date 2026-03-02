package dbhook

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"

	"shadiff/internal/logger"
	"shadiff/internal/model"
)

// MySQLHook MySQL 协议代理，解析 COM_QUERY 包捕获 SQL 语句
type MySQLHook struct {
	listenAddr  string
	targetAddr  string
	listener    net.Listener
	sideEffects chan model.SideEffect
	done        chan struct{}
	wg          sync.WaitGroup
}

// MySQL 协议常量
const (
	mysqlComQuery        = 0x03
	mysqlComStmtPrepare  = 0x16
	mysqlComStmtExecute  = 0x17
)

func NewMySQLHook(listenAddr, targetAddr string) *MySQLHook {
	return &MySQLHook{
		listenAddr:  listenAddr,
		targetAddr:  targetAddr,
		sideEffects: make(chan model.SideEffect, 1000),
		done:        make(chan struct{}),
	}
}

func (h *MySQLHook) Type() string { return "mysql" }

func (h *MySQLHook) SideEffects() <-chan model.SideEffect {
	return h.sideEffects
}

func (h *MySQLHook) Start(ctx context.Context) error {
	var err error
	h.listener, err = net.Listen("tcp", h.listenAddr)
	if err != nil {
		return err
	}

	logger.DBHookEvent("started", "mysql", "listen", h.listenAddr, "target", h.targetAddr)

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
					logger.Error("mysql accept error", err)
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

func (h *MySQLHook) Stop() error {
	close(h.done)
	if h.listener != nil {
		h.listener.Close()
	}
	h.wg.Wait()
	close(h.sideEffects)
	return nil
}

func (h *MySQLHook) handleConn(clientConn net.Conn) {
	defer clientConn.Close()

	// 连接到真实 MySQL
	serverConn, err := net.DialTimeout("tcp", h.targetAddr, 10*time.Second)
	if err != nil {
		logger.Error("mysql connect target failed", err)
		return
	}
	defer serverConn.Close()

	// 双向转发，同时嗅探客户端发往服务端的数据
	var wg sync.WaitGroup

	// 服务端 -> 客户端（透传）
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(clientConn, serverConn)
	}()

	// 客户端 -> 服务端（嗅探 MySQL 包）
	wg.Add(1)
	go func() {
		defer wg.Done()
		h.sniffClientToServer(clientConn, serverConn)
	}()

	wg.Wait()
}

// sniffClientToServer 嗅探客户端发送的数据，解析 MySQL 协议包
func (h *MySQLHook) sniffClientToServer(client, server net.Conn) {
	buf := make([]byte, 64*1024)
	for {
		n, err := client.Read(buf)
		if err != nil {
			return
		}

		// 转发到服务端
		if _, err := server.Write(buf[:n]); err != nil {
			return
		}

		// 尝试解析 MySQL 包
		h.parseMySQLPacket(buf[:n])
	}
}

// parseMySQLPacket 解析 MySQL 协议包，提取 SQL 语句
func (h *MySQLHook) parseMySQLPacket(data []byte) {
	// MySQL 包格式: 3字节长度 + 1字节序号 + payload
	if len(data) < 5 {
		return
	}

	// 读取包长度 (3 bytes little-endian)
	payloadLen := int(uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16)
	_ = data[3] // sequence number

	if len(data) < 4+payloadLen || payloadLen < 1 {
		return
	}

	commandByte := data[4]
	payload := data[5 : 4+payloadLen]

	switch commandByte {
	case mysqlComQuery:
		query := string(payload)
		h.emitSideEffect(query)
	case mysqlComStmtPrepare:
		query := string(payload)
		h.emitSideEffect(query)
	}
}

func (h *MySQLHook) emitSideEffect(query string) {
	effect := model.SideEffect{
		Type:      model.SideEffectDB,
		DBType:    "mysql",
		Query:     query,
		Timestamp: time.Now().UnixMilli(),
	}

	select {
	case h.sideEffects <- effect:
	default:
		logger.Warn("mysql side effect channel full, dropping")
	}
}

// readMySQLPacketLength 读取 MySQL 包长度（辅助函数）
func readMySQLPacketLength(data []byte) int {
	if len(data) < 3 {
		return 0
	}
	return int(binary.LittleEndian.Uint32(append(data[:3], 0)))
}
