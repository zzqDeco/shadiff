package storage

import "shadiff/internal/model"

// SessionStore 定义会话存储的接口
type SessionStore interface {
	Create(session *model.Session) error
	Get(id string) (*model.Session, error)
	List(filter *model.SessionFilter) ([]model.Session, error)
	Update(session *model.Session) error
	Delete(id string) error
}

// RecordStore 定义行为记录存储的接口
type RecordStore interface {
	// AppendRecord 追加一条记录到会话 (JSONL 流式写入)
	AppendRecord(sessionID string, record *model.Record) error
	// ListRecords 读取会话的所有记录
	ListRecords(sessionID string) ([]model.Record, error)
	// GetRecord 获取单条记录
	GetRecord(sessionID string, recordID string) (*model.Record, error)
	// CountRecords 返回会话的记录数
	CountRecords(sessionID string) (int, error)
}

// DiffStore 定义对拍结果存储的接口
type DiffStore interface {
	// SaveResults 保存对拍结果
	SaveResults(sessionID string, results []model.DiffResult) error
	// LoadResults 加载对拍结果
	LoadResults(sessionID string) ([]model.DiffResult, error)
}
