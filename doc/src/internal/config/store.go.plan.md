# store.go Technical Reference

## 1. File Location
- Project: shadiff
- Source file: internal/config/store.go
- Doc file: doc/src/internal/config/store.go.plan.md
- File type: Go source
- Module: shadiff

## 2. Core Responsibility
- Provides a thread-safe configuration store (`Store`) that loads, saves, and atomically updates the application configuration, persisted as `~/.shadiff/config.json`.
- Acts as the single source of truth for runtime configuration across all components.
- Changes to this file should be kept in sync with project-level documentation.

## 3. Inputs & Outputs
- Input sources: Reads configuration from `~/.shadiff/config.json` on disk; receives programmatic updates via the `Update` method.
- Output results: Provides `AppConfig` copies to consumers via `Get()`; writes updated configuration back to the JSON file via `Save()` and `Update()`.

## 4. Key Implementation Details
- Structs/interfaces:
  - `Store` -- Thread-safe config holder. Unexported fields: `path` (file path string), `config` (*AppConfig), `mu` (sync.RWMutex).
- Exported functions/methods:
  - `NewStore() (*Store, error)` -- Creates a store, ensures `~/.shadiff/` directory exists, loads config from file or falls back to `DefaultConfig()`.
  - `(s *Store) Load() error` -- Reads and unmarshals config from file. Uses `DefaultConfig()` as the base before unmarshaling to preserve defaults for missing fields.
  - `(s *Store) Save() error` -- Marshals and writes config to file with indented JSON formatting.
  - `(s *Store) Get() *AppConfig` -- Returns a shallow copy of the current config (read-lock protected).
  - `(s *Store) Update(fn func(*AppConfig)) error` -- Applies a mutation function to the config and persists the result atomically (write-lock protected).
  - `(s *Store) DataDir() string` -- Returns the configured data directory, defaulting to `~/.shadiff` if not explicitly set.
- Constants: None.

## 5. Dependencies
- Internal: `config.go` (same package) -- uses `AppConfig` and `DefaultConfig()`.
- External:
  - `encoding/json` -- JSON marshaling/unmarshaling.
  - `os` -- File I/O, user home directory lookup.
  - `path/filepath` -- Path construction.
  - `sync` -- RWMutex for thread safety.

## 6. Change Impact
- Changes to `Store` methods affect all components that read or write configuration at runtime.
- The file path `~/.shadiff/config.json` is hardcoded in `NewStore()`; changing it affects where users store their config.
- `Get()` returns a shallow copy; if `AppConfig` gains fields with reference types (slices, maps), deep-copy logic may be needed to prevent data races.
- `DataDir()` is used by the storage layer and logger to determine where data and logs are written.

## 7. Maintenance Notes
- `Load()` uses `DefaultConfig()` as the unmarshal target, so any fields missing from the JSON file will retain their default values. This provides forward-compatible config loading.
- `Get()` performs a struct value copy (`cfg := *s.config`), which is a shallow copy. Slice and map fields (e.g., `IgnoreHeaders`, `Rules`, `DBProxies`) share the underlying arrays with the Store's internal config. Callers should not mutate slices/maps returned by `Get()`.
- `Update()` holds a write lock during both the mutation and the file write; keep mutation functions fast to avoid blocking readers.
- `NewStore()` silently falls back to defaults on load failure (e.g., first run); this is intentional but means file permission errors are not surfaced to the user.
- Consider adding file-level atomic write (write to temp file, then rename) in `Save()`/`Update()` to prevent config corruption on crash.
