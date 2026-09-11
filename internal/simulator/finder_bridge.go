package simulator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/webdav"
)

// FinderBridge exposes callback-backed Android container contents through a
// read-only WebDAV server bound to the local machine.
type FinderBridge struct {
	ctx         context.Context
	cancel      context.CancelFunc
	list        func(context.Context, string) ([]FileInfo, error)
	materialize func(context.Context, string) (string, error)
	listener    net.Listener
	server      *http.Server

	mu       sync.RWMutex
	roots    map[string]string
	closed   bool
	once     sync.Once
	err      error
	serveErr error
}

// NewFinderBridge starts a loopback-only read-only WebDAV server. A bridge URL
// is created separately for each Android container with URL.
func NewFinderBridge(ctx context.Context, list func(context.Context, string) ([]FileInfo, error), materialize func(context.Context, string) (string, error)) (*FinderBridge, error) {
	if list == nil || materialize == nil {
		return nil, errors.New("finder bridge requires list and materialize callbacks")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for finder bridge: %w", err)
	}
	bridgeCtx, cancel := context.WithCancel(ctx)
	bridge := &FinderBridge{
		ctx:         bridgeCtx,
		cancel:      cancel,
		list:        list,
		materialize: materialize,
		listener:    listener,
		roots:       make(map[string]string),
	}
	bridge.server = &http.Server{
		Handler:           http.HandlerFunc(bridge.serveHTTP),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	go func() {
		err := bridge.server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			bridge.mu.Lock()
			bridge.serveErr = err
			bridge.mu.Unlock()
		}
	}()
	return bridge, nil
}

// URL creates a capability URL for root. The opaque capability is scoped to
// this bridge and is intentionally not logged.
func (b *FinderBridge) URL(ctx context.Context, root string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := b.ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(root) == "" {
		return "", errors.New("finder bridge root is empty")
	}

	token, err := finderBridgeToken()
	if err != nil {
		return "", err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return "", net.ErrClosed
	}
	b.roots[token] = root
	return "http://" + b.listener.Addr().String() + "/" + token + "/", nil
}

// Close stops accepting requests and cancels outstanding callback contexts.
func (b *FinderBridge) Close() error {
	b.once.Do(func() {
		b.mu.Lock()
		b.closed = true
		b.roots = nil
		b.mu.Unlock()
		b.cancel()
		b.err = b.server.Close()
		if errors.Is(b.err, http.ErrServerClosed) || errors.Is(b.err, net.ErrClosed) {
			b.err = nil
		}
		b.mu.RLock()
		serveErr := b.serveErr
		b.mu.RUnlock()
		if serveErr != nil {
			b.err = errors.Join(b.err, serveErr)
		}
	})
	return b.err
}

func (b *FinderBridge) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if !b.isLocalRequest(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodOptions && r.Method != "PROPFIND" && r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "OPTIONS, PROPFIND, GET, HEAD")
		http.Error(w, "read-only finder bridge", http.StatusMethodNotAllowed)
		return
	}

	root, token, ok := b.requestRoot(r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	requestCtx, cancel := context.WithCancel(r.Context())
	stop := context.AfterFunc(b.ctx, cancel)
	defer stop()
	defer cancel()
	request := r.Clone(requestCtx)
	handler := webdav.Handler{
		Prefix:     "/" + token,
		FileSystem: &finderBridgeFS{bridge: b, root: root},
		LockSystem: webdav.NewMemLS(),
	}
	handler.ServeHTTP(w, request)
}

func (b *FinderBridge) isLocalRequest(r *http.Request) bool {
	if r.Host != b.listener.Addr().String() {
		return false
	}
	origin := r.Header.Get("Origin")
	return origin == "" || origin == "http://"+b.listener.Addr().String()
}

func (b *FinderBridge) requestRoot(r *http.Request) (string, string, bool) {
	rawPath := r.URL.EscapedPath()
	if !strings.HasPrefix(rawPath, "/") {
		return "", "", false
	}
	parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(rawPath, "/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", "", false
	}
	decoded := make([]string, len(parts))
	for index, part := range parts {
		if !validFinderPathPart(part) {
			return "", "", false
		}
		decoded[index], _ = pathUnescape(part)
	}

	b.mu.RLock()
	root, ok := b.roots[decoded[0]]
	b.mu.RUnlock()
	if !ok {
		return "", "", false
	}
	return root, decoded[0], true
}

func validFinderPathPart(raw string) bool {
	if raw == "" {
		return false
	}
	decoded, err := pathUnescape(raw)
	if err != nil || decoded == "" || decoded == "." || decoded == ".." {
		return false
	}
	return !strings.ContainsAny(decoded, "/\\\x00")
}

func finderBridgeToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate finder bridge capability: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

type finderBridgeFS struct {
	bridge *FinderBridge
	root   string
}

func (f *finderBridgeFS) Mkdir(context.Context, string, os.FileMode) error { return fs.ErrPermission }

func (f *finderBridgeFS) RemoveAll(context.Context, string) error { return fs.ErrPermission }

func (f *finderBridgeFS) Rename(context.Context, string, string) error { return fs.ErrPermission }

func (f *finderBridgeFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	entry, err := f.resolve(ctx, name)
	if err != nil {
		return nil, err
	}
	return finderBridgeInfo{entry: entry}, nil
}

func (f *finderBridgeFS) OpenFile(ctx context.Context, name string, flag int, _ os.FileMode) (webdav.File, error) {
	if flag&(os.O_WRONLY|os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_EXCL|os.O_TRUNC) != 0 {
		return nil, fs.ErrPermission
	}
	entry, err := f.resolve(ctx, name)
	if err != nil {
		return nil, err
	}
	if entry.IsDirectory {
		entries, err := f.list(ctx, entry.Path)
		if err != nil {
			return nil, err
		}
		return &finderBridgeDir{info: finderBridgeInfo{entry: entry}, entries: entries}, nil
	}
	return &finderBridgeFile{
		ctx:         ctx,
		entry:       entry,
		info:        finderBridgeInfo{entry: entry},
		materialize: f.bridge.materialize,
	}, nil
}

func (f *finderBridgeFS) resolve(ctx context.Context, name string) (FileInfo, error) {
	parts, err := finderBridgePathParts(name)
	if err != nil {
		return FileInfo{}, err
	}
	current := FileInfo{Name: "/", Path: f.root, IsDirectory: true}
	for _, segment := range parts {
		entries, err := f.list(ctx, current.Path)
		if err != nil {
			return FileInfo{}, err
		}
		found := false
		for _, entry := range entries {
			if entry.Name == segment {
				current = entry
				found = true
				break
			}
		}
		if !found {
			return FileInfo{}, fs.ErrNotExist
		}
	}
	return current, nil
}

func (f *finderBridgeFS) list(ctx context.Context, directory string) ([]FileInfo, error) {
	entries, err := f.bridge.list(ctx, directory)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !validFinderEntryName(entry.Name) || entry.Path != path.Join(directory, entry.Name) {
			return nil, errors.New("finder bridge callback returned an invalid entry")
		}
	}
	return entries, nil
}

func finderBridgePathParts(name string) ([]string, error) {
	if name == "/" {
		return nil, nil
	}
	if !strings.HasPrefix(name, "/") || strings.Contains(name, "\\") {
		return nil, fs.ErrInvalid
	}
	parts := strings.Split(strings.TrimPrefix(name, "/"), "/")
	for _, part := range parts {
		if !validFinderEntryName(part) {
			return nil, fs.ErrInvalid
		}
	}
	return parts, nil
}

func validFinderEntryName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\\x00")
}

type finderBridgeInfo struct{ entry FileInfo }

func (i finderBridgeInfo) Name() string { return i.entry.Name }
func (i finderBridgeInfo) Size() int64  { return i.entry.Size }
func (i finderBridgeInfo) Mode() os.FileMode {
	if i.entry.IsDirectory {
		return os.ModeDir | 0555
	}
	return 0444
}
func (i finderBridgeInfo) ModTime() time.Time { return i.entry.ModifiedAt }
func (i finderBridgeInfo) IsDir() bool        { return i.entry.IsDirectory }
func (i finderBridgeInfo) Sys() any           { return nil }

type finderBridgeFile struct {
	ctx         context.Context
	entry       FileInfo
	info        finderBridgeInfo
	materialize func(context.Context, string) (string, error)

	mu   sync.Mutex
	file *os.File
}

func (f *finderBridgeFile) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.file == nil {
		return nil
	}
	return f.file.Close()
}

func (f *finderBridgeFile) Read(buffer []byte) (int, error) {
	file, err := f.open()
	if err != nil {
		return 0, err
	}
	return file.Read(buffer)
}

func (f *finderBridgeFile) Seek(offset int64, whence int) (int64, error) {
	file, err := f.open()
	if err != nil {
		return 0, err
	}
	return file.Seek(offset, whence)
}

func (f *finderBridgeFile) Stat() (os.FileInfo, error) { return f.info, nil }

func (f *finderBridgeFile) Readdir(int) ([]os.FileInfo, error) {
	return nil, &os.PathError{Op: "readdir", Path: f.info.Name(), Err: syscall.ENOTDIR}
}
func (f *finderBridgeFile) Write([]byte) (int, error) { return 0, fs.ErrPermission }

func (f *finderBridgeFile) open() (*os.File, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.file != nil {
		return f.file, nil
	}
	localPath, err := f.materialize(f.ctx, f.entry.Path)
	if err != nil {
		return nil, fmt.Errorf("materialize %q: %w", f.entry.Name, err)
	}
	localInfo, err := os.Lstat(localPath)
	if err != nil {
		return nil, fmt.Errorf("inspect materialized %q: %w", f.entry.Name, err)
	}
	if !localInfo.Mode().IsRegular() {
		return nil, errors.New("materialized finder file is not a regular file")
	}
	f.file, err = os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("open materialized %q: %w", f.entry.Name, err)
	}
	return f.file, nil
}

type finderBridgeDir struct {
	info     finderBridgeInfo
	entries  []FileInfo
	position int
}

func (d *finderBridgeDir) Close() error             { return nil }
func (d *finderBridgeDir) Read([]byte) (int, error) { return 0, io.EOF }
func (d *finderBridgeDir) Seek(offset int64, whence int) (int64, error) {
	if offset != 0 || whence != io.SeekStart {
		return 0, fs.ErrInvalid
	}
	d.position = 0
	return 0, nil
}
func (d *finderBridgeDir) Stat() (os.FileInfo, error) { return d.info, nil }
func (d *finderBridgeDir) Write([]byte) (int, error)  { return 0, fs.ErrPermission }
func (d *finderBridgeDir) Readdir(count int) ([]os.FileInfo, error) {
	if d.position >= len(d.entries) {
		if count > 0 {
			return nil, io.EOF
		}
		return []os.FileInfo{}, nil
	}
	end := len(d.entries)
	if count > 0 && d.position+count < end {
		end = d.position + count
	}
	results := make([]os.FileInfo, 0, end-d.position)
	for _, entry := range d.entries[d.position:end] {
		results = append(results, finderBridgeInfo{entry: entry})
	}
	d.position = end
	return results, nil
}

// url.PathUnescape is assigned to make malformed escaped paths testable without
// exposing URL parsing details throughout the request validator.
var pathUnescape = url.PathUnescape
