package dbhook

import (
	"context"

	"shadiff/internal/model"
)

// DBHook 数据库操作捕获接口
type DBHook interface {
	// Start 启动数据库代理，监听 listenAddr，转发到 targetAddr
	Start(ctx context.Context) error
	// Stop 停止代理
	Stop() error
	// SideEffects 返回捕获的副作用通道
	SideEffects() <-chan model.SideEffect
	// Type 返回数据库类型
	Type() string
}

// Config DB 代理通用配置
type Config struct {
	DBType     string // mysql / postgres / mongo
	ListenAddr string // 代理监听地址
	TargetAddr string // 真实 DB 地址
}

// NewHook 根据类型创建对应的 DB hook
func NewHook(cfg Config) (DBHook, error) {
	switch cfg.DBType {
	case "mysql":
		return NewMySQLHook(cfg.ListenAddr, cfg.TargetAddr), nil
	case "postgres":
		return NewPostgresHook(cfg.ListenAddr, cfg.TargetAddr), nil
	case "mongo":
		return NewMongoHook(cfg.ListenAddr, cfg.TargetAddr), nil
	default:
		return nil, &UnsupportedDBError{DBType: cfg.DBType}
	}
}

// UnsupportedDBError 不支持的数据库类型
type UnsupportedDBError struct {
	DBType string
}

func (e *UnsupportedDBError) Error() string {
	return "不支持的数据库类型: " + e.DBType
}
