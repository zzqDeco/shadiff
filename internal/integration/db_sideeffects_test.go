//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"shadiff/internal/capture/dbhook"
	diffengine "shadiff/internal/diff"
	"shadiff/internal/model"
	replayengine "shadiff/internal/replay"
	"shadiff/internal/storage"
)

const (
	mysqlPassword = "shadiff-pass"
	dbName        = "shadiff"
)

func TestMySQLDBProxyCapturesRealQuery(t *testing.T) {
	targetAddr := startMySQL(t)
	group, sink, proxyAddr := startDBProxy(t, "mysql", targetAddr)

	db := openSQL(t, "mysql", mysqlDSN(proxyAddr))
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const marker = "shadiff_mysql_integration"
	var got int
	if err := db.QueryRowContext(ctx, "SELECT 42 /* "+marker+" */").Scan(&got); err != nil {
		t.Fatalf("mysql query through proxy failed: %v", err)
	}
	if got != 42 {
		t.Fatalf("mysql query result = %d, want 42", got)
	}

	flushGroup(t, group)
	effect := waitForEffect(t, sink, func(effect model.SideEffect) bool {
		return effect.DBType == "mysql" && strings.Contains(effect.Query, marker)
	})
	if effect.Type != model.SideEffectDB {
		t.Fatalf("effect type = %q, want %q", effect.Type, model.SideEffectDB)
	}
}

func TestPostgresDBProxyCapturesRealQuery(t *testing.T) {
	targetAddr := startPostgres(t)
	group, sink, proxyAddr := startDBProxy(t, "postgres", targetAddr)

	db := openSQL(t, "postgres", postgresDSN(proxyAddr))
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const marker = "shadiff_postgres_integration"
	var got int
	if err := db.QueryRowContext(ctx, "SELECT 43 /* "+marker+" */").Scan(&got); err != nil {
		t.Fatalf("postgres query through proxy failed: %v", err)
	}
	if got != 43 {
		t.Fatalf("postgres query result = %d, want 43", got)
	}

	flushGroup(t, group)
	effect := waitForEffect(t, sink, func(effect model.SideEffect) bool {
		return effect.DBType == "postgres" && strings.Contains(effect.Query, marker)
	})
	if effect.Type != model.SideEffectDB {
		t.Fatalf("effect type = %q, want %q", effect.Type, model.SideEffectDB)
	}
}

func TestMongoDBProxyCapturesRealCommands(t *testing.T) {
	targetAddr := startMongo(t)
	group, sink, proxyAddr := startDBProxy(t, "mongo", targetAddr)

	client := openMongo(t, mongoURI(proxyAddr))
	defer disconnectMongo(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	coll := client.Database(dbName).Collection("users")
	if _, err := coll.InsertOne(ctx, bson.D{{Key: "name", Value: "ada"}}); err != nil {
		t.Fatalf("mongo insert through proxy failed: %v", err)
	}
	var found bson.M
	if err := coll.FindOne(ctx, bson.D{{Key: "name", Value: "ada"}}).Decode(&found); err != nil {
		t.Fatalf("mongo find through proxy failed: %v", err)
	}

	flushGroup(t, group)
	seen := collectMongoEffects(t, sink)
	for _, op := range []string{"insert", "find"} {
		effect, ok := seen[op]
		if !ok {
			t.Fatalf("missing mongo %s side effect; seen=%v", op, seen)
		}
		if effect.Collection != "users" || effect.Database != dbName {
			t.Fatalf("mongo %s effect = collection %q database %q, want users/%s", op, effect.Collection, effect.Database, dbName)
		}
	}
}

func TestRedisDBProxyCapturesRealCommands(t *testing.T) {
	targetAddr := startRedis(t)
	group, sink, proxyAddr := startDBProxy(t, "redis", targetAddr)

	client := openRedis(t, proxyAddr)
	defer closeRedis(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const key = "shadiff:redis:integration"
	if err := client.Set(ctx, key, "ada", 0).Err(); err != nil {
		t.Fatalf("redis set through proxy failed: %v", err)
	}
	got, err := client.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("redis get through proxy failed: %v", err)
	}
	if got != "ada" {
		t.Fatalf("redis get result = %q, want ada", got)
	}

	flushGroup(t, group)
	setEffect := waitForEffect(t, sink, func(effect model.SideEffect) bool {
		return effect.DBType == "redis" && effect.RedisCommand == "SET" && effect.RedisKey == key
	})
	if setEffect.Type != model.SideEffectDB {
		t.Fatalf("effect type = %q, want %q", setEffect.Type, model.SideEffectDB)
	}
	getEffect := waitForEffect(t, sink, func(effect model.SideEffect) bool {
		return effect.DBType == "redis" && effect.RedisCommand == "GET" && effect.RedisKey == key
	})
	if getEffect.Type != model.SideEffectDB {
		t.Fatalf("effect type = %q, want %q", getEffect.Type, model.SideEffectDB)
	}
}

func TestReplayDiffDetectsDBSideEffectDifference(t *testing.T) {
	targetAddr := startMySQL(t)
	group, sink, proxyAddr := startDBProxy(t, "mysql", targetAddr)

	db := openSQL(t, "mysql", mysqlDSN(proxyAddr))
	defer db.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got int
		if err := db.QueryRowContext(r.Context(), "SELECT 2 /* shadiff_e2e_replay */").Scan(&got); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	store, err := storage.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error: %v", err)
	}

	session := model.Session{
		ID:     "integration",
		Name:   "integration",
		Status: model.SessionCompleted,
	}
	if err := store.Create(&session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := []byte(`{"ok":true}`)
	original := model.Record{
		ID:        "original",
		SessionID: session.ID,
		Sequence:  1,
		Request: model.HTTPRequest{
			Method:  http.MethodGet,
			Path:    "/query",
			Headers: map[string][]string{},
		},
		Response: model.HTTPResponse{
			StatusCode: http.StatusOK,
			Headers:    map[string][]string{},
			Body:       body,
			BodyLen:    int64(len(body)),
		},
		SideEffects: []model.SideEffect{{
			Type:      model.SideEffectDB,
			DBType:    "mysql",
			Query:     "SELECT 1 /* shadiff_e2e_original */",
			Timestamp: time.Now().UnixMilli(),
		}},
		RecordedAt: time.Now().UnixMilli(),
	}
	if err := store.AppendRecord(session.ID, &original); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}

	replay := replayengine.NewEngine(store, replayengine.EngineConfig{
		SessionID:    session.ID,
		TargetURL:    server.URL,
		Concurrency:  1,
		Timeout:      10 * time.Second,
		SideEffectCh: sink,
		Flusher:      group,
		FlushTimeout: 5 * time.Second,
	})
	replayResults, err := replay.Run()
	if err != nil {
		t.Fatalf("replay Run() error: %v", err)
	}
	if len(replayResults) != 1 {
		t.Fatalf("replay result count = %d, want 1", len(replayResults))
	}
	if !recordHasQuery(replayResults[0].Replayed, "shadiff_e2e_replay") {
		t.Fatalf("replay record missing captured side effect: %+v", replayResults[0].Replayed.SideEffects)
	}

	diff := diffengine.NewEngine(store, diffengine.EngineConfig{SessionID: session.ID})
	diffResults, err := diff.Run()
	if err != nil {
		t.Fatalf("diff Run() error: %v", err)
	}
	if len(diffResults) != 1 {
		t.Fatalf("diff result count = %d, want 1", len(diffResults))
	}
	if diffResults[0].Match {
		t.Fatal("expected DB side-effect difference to break match")
	}
	if !hasDBQueryDifference(diffResults[0]) {
		t.Fatalf("expected db query difference, got %+v", diffResults[0].Differences)
	}
}

func TestReplayDiffDetectsRedisSideEffectDifference(t *testing.T) {
	targetAddr := startRedis(t)
	group, sink, proxyAddr := startDBProxy(t, "redis", targetAddr)

	client := openRedis(t, proxyAddr)
	defer closeRedis(t, client)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := client.Set(r.Context(), "shadiff:redis:replay", "new", 0).Err(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	store, err := storage.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error: %v", err)
	}

	session := model.Session{
		ID:     "redis-integration",
		Name:   "redis-integration",
		Status: model.SessionCompleted,
	}
	if err := store.Create(&session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	body := []byte(`{"ok":true}`)
	original := model.Record{
		ID:        "original",
		SessionID: session.ID,
		Sequence:  1,
		Request: model.HTTPRequest{
			Method:  http.MethodGet,
			Path:    "/redis",
			Headers: map[string][]string{},
		},
		Response: model.HTTPResponse{
			StatusCode: http.StatusOK,
			Headers:    map[string][]string{},
			Body:       body,
			BodyLen:    int64(len(body)),
		},
		SideEffects: []model.SideEffect{{
			Type:         model.SideEffectDB,
			DBType:       "redis",
			RedisCommand: "SET",
			RedisKey:     "shadiff:redis:record",
			RedisArgs:    []string{"shadiff:redis:record", "old"},
			Timestamp:    time.Now().UnixMilli(),
		}},
		RecordedAt: time.Now().UnixMilli(),
	}
	if err := store.AppendRecord(session.ID, &original); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}

	replay := replayengine.NewEngine(store, replayengine.EngineConfig{
		SessionID:    session.ID,
		TargetURL:    server.URL,
		Concurrency:  1,
		Timeout:      10 * time.Second,
		SideEffectCh: sink,
		Flusher:      group,
		FlushTimeout: 5 * time.Second,
	})
	replayResults, err := replay.Run()
	if err != nil {
		t.Fatalf("replay Run() error: %v", err)
	}
	if len(replayResults) != 1 {
		t.Fatalf("replay result count = %d, want 1", len(replayResults))
	}
	if !recordHasRedisCommand(replayResults[0].Replayed, "SET", "shadiff:redis:replay") {
		t.Fatalf("replay record missing Redis side effect: %+v", replayResults[0].Replayed.SideEffects)
	}

	diff := diffengine.NewEngine(store, diffengine.EngineConfig{SessionID: session.ID})
	diffResults, err := diff.Run()
	if err != nil {
		t.Fatalf("diff Run() error: %v", err)
	}
	if len(diffResults) != 1 {
		t.Fatalf("diff result count = %d, want 1", len(diffResults))
	}
	if diffResults[0].Match {
		t.Fatal("expected Redis side-effect difference to break match")
	}
	if !hasRedisDifference(diffResults[0]) {
		t.Fatalf("expected Redis difference, got %+v", diffResults[0].Differences)
	}
}

func requireDocker(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}").CombinedOutput()
	if err == nil {
		return
	}

	msg := strings.TrimSpace(string(out))
	if os.Getenv("CI") == "true" {
		t.Fatalf("docker is required for integration tests: %v %s", err, msg)
	}
	t.Skipf("docker is not available: %v %s", err, msg)
}

func startMySQL(t *testing.T) string {
	t.Helper()
	requireDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	container := startContainer(t, ctx, testcontainers.ContainerRequest{
		Image:        "mysql:8.0",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": mysqlPassword,
			"MYSQL_DATABASE":      dbName,
		},
		WaitingFor: wait.ForListeningPort("3306/tcp").WithStartupTimeout(2 * time.Minute),
	})
	addr := mappedAddress(t, ctx, container, "3306/tcp")
	waitForSQL(t, "mysql", mysqlDSN(addr))
	return addr
}

func startPostgres(t *testing.T) string {
	t.Helper()
	requireDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	container := startContainer(t, ctx, testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": mysqlPassword,
			"POSTGRES_DB":       dbName,
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(2 * time.Minute),
	})
	addr := mappedAddress(t, ctx, container, "5432/tcp")
	waitForSQL(t, "postgres", postgresDSN(addr))
	return addr
}

func startMongo(t *testing.T) string {
	t.Helper()
	requireDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	container := startContainer(t, ctx, testcontainers.ContainerRequest{
		Image:        "mongo:7",
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForListeningPort("27017/tcp").WithStartupTimeout(2 * time.Minute),
	})
	addr := mappedAddress(t, ctx, container, "27017/tcp")
	waitForMongo(t, mongoURI(addr))
	return addr
}

func startRedis(t *testing.T) string {
	t.Helper()
	requireDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	container := startContainer(t, ctx, testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(2 * time.Minute),
	})
	addr := mappedAddress(t, ctx, container, "6379/tcp")
	waitForRedis(t, addr)
	return addr
}

func startContainer(t *testing.T, ctx context.Context, req testcontainers.ContainerRequest) testcontainers.Container {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start container %s: %v", req.Image, err)
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := container.Terminate(cleanupCtx); err != nil {
			t.Logf("terminate container %s: %v", req.Image, err)
		}
	})

	return container
}

func mappedAddress(t *testing.T, ctx context.Context, container testcontainers.Container, port string) string {
	t.Helper()

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("container Host() error: %v", err)
	}
	mappedPort, err := container.MappedPort(ctx, port)
	if err != nil {
		t.Fatalf("container MappedPort(%s) error: %v", port, err)
	}
	return net.JoinHostPort(host, mappedPort.Port())
}

func startDBProxy(t *testing.T, dbType, targetAddr string) (*dbhook.Group, <-chan model.SideEffect, string) {
	t.Helper()

	listenAddr := freeTCPAddr(t)
	hook, err := dbhook.NewHook(dbhook.Config{
		DBType:     dbType,
		ListenAddr: listenAddr,
		TargetAddr: targetAddr,
	})
	if err != nil {
		t.Fatalf("NewHook(%s) error: %v", dbType, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := hook.Start(ctx); err != nil {
		cancel()
		t.Fatalf("Start(%s) error: %v", dbType, err)
	}

	sink := make(chan model.SideEffect, 100)
	group := dbhook.NewGroup(ctx, []dbhook.DBHook{hook}, sink)
	t.Cleanup(func() {
		cancel()
		done := make(chan error, 1)
		go func() {
			done <- group.Stop()
		}()
		select {
		case err := <-done:
			if err != nil {
				t.Logf("stop %s proxy: %v", dbType, err)
			}
		case <-time.After(5 * time.Second):
			t.Logf("timed out stopping %s proxy; continuing test cleanup", dbType)
		}
	})

	return group, sink, listenAddr
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve tcp address: %v", err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

func flushGroup(t *testing.T, group *dbhook.Group) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := group.Flush(ctx); err != nil {
		t.Fatalf("Flush() error: %v", err)
	}
}

func waitForEffect(t *testing.T, sink <-chan model.SideEffect, match func(model.SideEffect) bool) model.SideEffect {
	t.Helper()

	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	for {
		select {
		case effect := <-sink:
			if match(effect) {
				return effect
			}
		case <-timer.C:
			t.Fatal("timed out waiting for matching side effect")
		}
	}
}

func collectMongoEffects(t *testing.T, sink <-chan model.SideEffect) map[string]model.SideEffect {
	t.Helper()

	seen := make(map[string]model.SideEffect)
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	for len(seen) < 2 {
		select {
		case effect := <-sink:
			if effect.DBType == "mongo" && effect.Collection == "users" && effect.Database == dbName {
				switch effect.Operation {
				case "insert", "find":
					seen[effect.Operation] = effect
				}
			}
		case <-timer.C:
			return seen
		}
	}
	return seen
}

func openSQL(t *testing.T, driver, dsn string) *sql.DB {
	t.Helper()

	db, err := sql.Open(driver, dsn)
	if err != nil {
		t.Fatalf("sql.Open(%s) error: %v", driver, err)
	}
	return db
}

func waitForSQL(t *testing.T, driver, dsn string) {
	t.Helper()

	deadline := time.Now().Add(90 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		db, err := sql.Open(driver, dsn)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			lastErr = db.PingContext(ctx)
			cancel()
			_ = db.Close()
			if lastErr == nil {
				return
			}
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("%s database not ready: %v", driver, lastErr)
}

func waitForMongo(t *testing.T, uri string) {
	t.Helper()

	deadline := time.Now().Add(90 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		client, err := mongo.Connect(options.Client().ApplyURI(uri))
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			lastErr = client.Ping(ctx, readpref.Primary())
			cancel()
			_ = client.Disconnect(context.Background())
			if lastErr == nil {
				return
			}
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("mongo database not ready: %v", lastErr)
}

func waitForRedis(t *testing.T, addr string) {
	t.Helper()

	deadline := time.Now().Add(90 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		client := redis.NewClient(&redis.Options{Addr: addr, Protocol: 2})
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		lastErr = client.Ping(ctx).Err()
		cancel()
		_ = client.Close()
		if lastErr == nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("redis database not ready: %v", lastErr)
}

func openMongo(t *testing.T, uri string) *mongo.Client {
	t.Helper()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("mongo.Connect() error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		t.Fatalf("mongo Ping() error: %v", err)
	}
	return client
}

func openRedis(t *testing.T, addr string) *redis.Client {
	t.Helper()

	client := redis.NewClient(&redis.Options{Addr: addr, Protocol: 2})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("redis Ping() error: %v", err)
	}
	return client
}

func closeRedis(t *testing.T, client *redis.Client) {
	t.Helper()

	if err := client.Close(); err != nil {
		t.Logf("redis close: %v", err)
	}
}

func disconnectMongo(t *testing.T, client *mongo.Client) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Disconnect(ctx); err != nil {
		t.Logf("mongo disconnect: %v", err)
	}
}

func mysqlDSN(addr string) string {
	return fmt.Sprintf("root:%s@tcp(%s)/%s?parseTime=true", mysqlPassword, addr, dbName)
}

func postgresDSN(addr string) string {
	return fmt.Sprintf("postgres://postgres:%s@%s/%s?sslmode=disable", mysqlPassword, addr, dbName)
}

func mongoURI(addr string) string {
	return fmt.Sprintf("mongodb://%s/?directConnection=true", addr)
}

func recordHasQuery(record model.Record, marker string) bool {
	for _, effect := range record.SideEffects {
		if effect.DBType == "mysql" && strings.Contains(effect.Query, marker) {
			return true
		}
	}
	return false
}

func recordHasRedisCommand(record model.Record, command, key string) bool {
	for _, effect := range record.SideEffects {
		if effect.DBType == "redis" && effect.RedisCommand == command && effect.RedisKey == key {
			return true
		}
	}
	return false
}

func hasDBQueryDifference(result model.DiffResult) bool {
	for _, difference := range result.Differences {
		if difference.Kind == model.DiffDBQuery && !difference.Ignored {
			return true
		}
	}
	return false
}

func hasRedisDifference(result model.DiffResult) bool {
	for _, difference := range result.Differences {
		if difference.Kind == model.DiffRedisCommand && !difference.Ignored {
			return true
		}
	}
	return false
}
