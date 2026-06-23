// Package webserver exposes Weather CLI through a browser terminal.
package webserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

// Options configures the HTTP bridge.
type Options struct {
	Addr       string
	CLIPath    string
	StaticRoot string
	Version    string
}

// New builds the Weather CLI web bridge.
func New(opts Options, logger *slog.Logger) (http.Handler, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if opts.CLIPath == "" {
		opts.CLIPath = "weather-cli"
	}

	static, err := staticFS(opts.StaticRoot)
	if err != nil {
		return nil, err
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(static)))
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"ok","version":%q}`+"\n", opts.Version)
	})
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Warn("websocket upgrade failed", "error", err)
			return
		}
		session := newSession(conn, opts.CLIPath, launchArgs(r), logger)
		session.run(r.Context())
	})
	return mux, nil
}

func launchArgs(r *http.Request) []string {
	args := []string{"--render-mode", "inline"}
	q := r.URL.Query()
	lat, latErr := strconv.ParseFloat(q.Get("lat"), 64)
	lon, lonErr := strconv.ParseFloat(q.Get("lon"), 64)
	if latErr == nil && lonErr == nil && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180 {
		args = append(args,
			"--lat", strconv.FormatFloat(lat, 'f', -1, 64),
			"--lon", strconv.FormatFloat(lon, 'f', -1, 64),
		)
	}
	return args
}

func staticFS(root string) (fs.FS, error) {
	if root == "" {
		root = "web/static"
	}
	return os.DirFS(root), nil
}

type session struct {
	conn    *websocket.Conn
	cliPath string
	args    []string
	logger  *slog.Logger

	ptmx   *os.File
	ptmxMu sync.Mutex
}

type clientMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
	Data string `json:"data"`
}

func newSession(conn *websocket.Conn, cliPath string, args []string, logger *slog.Logger) *session {
	return &session{conn: conn, cliPath: cliPath, args: args, logger: logger}
}

func (s *session) run(ctx context.Context) {
	defer func() {
		_ = s.conn.Close()
	}()

	cols, rows := uint16(100), uint16(32)
	cmd := exec.CommandContext(ctx, s.cliPath, s.args...)
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"WEATHER_CLI_RENDER_MODE=inline",
		"XDG_CONFIG_HOME="+sessionConfigDir(),
	)

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		s.writeText("failed to start weather-cli: " + err.Error() + "\r\n")
		return
	}
	s.ptmx = ptmx
	defer func() {
		_ = ptmx.Close()
	}()
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 32*1024)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				if werr := s.conn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				if !errors.Is(err, io.EOF) {
					s.logger.Debug("pty read stopped", "error", err)
				}
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		default:
		}
		if err := s.readClientMessage(); err != nil {
			return
		}
	}
}

func (s *session) readClientMessage() error {
	messageType, payload, err := s.conn.ReadMessage()
	if err != nil {
		return err
	}
	switch messageType {
	case websocket.BinaryMessage:
		s.writePTY(payload)
	case websocket.TextMessage:
		var msg clientMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			s.writePTY(payload)
			return nil
		}
		switch msg.Type {
		case "resize":
			if msg.Cols > 0 && msg.Rows > 0 {
				s.resize(msg.Cols, msg.Rows)
			}
		case "input":
			s.writePTY([]byte(msg.Data))
		default:
			s.writePTY(payload)
		}
	}
	return nil
}

func (s *session) resize(cols, rows uint16) {
	s.ptmxMu.Lock()
	defer s.ptmxMu.Unlock()
	if s.ptmx == nil {
		return
	}
	_ = pty.Setsize(s.ptmx, &pty.Winsize{Cols: cols, Rows: rows})
}

func (s *session) writePTY(payload []byte) {
	s.ptmxMu.Lock()
	defer s.ptmxMu.Unlock()
	if s.ptmx != nil {
		_, _ = s.ptmx.Write(payload)
	}
}

func (s *session) writeText(text string) {
	_ = s.conn.WriteMessage(websocket.BinaryMessage, []byte(text))
}

func sessionConfigDir() string {
	base := os.Getenv("WEATHER_CLI_WEB_CONFIG_DIR")
	if base == "" {
		base = filepath.Join(os.TempDir(), "weather-cli-web")
	}
	_ = os.MkdirAll(base, 0o700)
	return filepath.Join(base, "session-"+strconv.FormatInt(time.Now().UnixNano(), 36)+"-"+strings.ReplaceAll(strconv.Itoa(os.Getpid()), "-", ""))
}
