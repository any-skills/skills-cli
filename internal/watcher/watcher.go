package watcher

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
	"github.com/rushteam/skills-cli/internal/agent"
	"github.com/rushteam/skills-cli/internal/config"
	syncer "github.com/rushteam/skills-cli/internal/sync"
)

const debounceDelay = 300 * time.Millisecond

var (
	infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type Watcher struct {
	cfg       *config.Config
	direction string
	watcher   *fsnotify.Watcher
	stopCh    chan struct{}
	mu        sync.Mutex
	pending   map[string]time.Time
}

func New(cfg *config.Config, direction string) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		cfg:       cfg,
		direction: direction,
		watcher:   w,
		stopCh:    make(chan struct{}),
		pending:   make(map[string]time.Time),
	}, nil
}

func (w *Watcher) Start() error {
	dirs := w.collectWatchDirs()
	if len(dirs) == 0 {
		return fmt.Errorf("no directories to watch")
	}

	for _, dir := range dirs {
		if err := w.addRecursive(dir); err != nil {
			slog.Warn("failed to watch directory", "dir", dir, "error", err)
		}
	}

	fmt.Println(infoStyle.Render(fmt.Sprintf("Watching %d directories (direction: %s)", len(dirs), w.direction)))
	for _, d := range dirs {
		fmt.Println(dimStyle.Render(fmt.Sprintf("  %s", agent.ShortenPath(d))))
	}

	go w.eventLoop()
	return nil
}

func (w *Watcher) Stop() {
	close(w.stopCh)
	w.watcher.Close()
}

func (w *Watcher) Wait() {
	<-w.stopCh
}

func (w *Watcher) collectWatchDirs() []string {
	var dirs []string
	centralDir := config.SkillsHome()

	switch w.direction {
	case config.WatchCentralToAgents:
		dirs = append(dirs, centralDir)
	case config.WatchAgentsToCentral:
		dirs = append(dirs, w.agentDirs()...)
	case config.WatchBidirectional:
		dirs = append(dirs, centralDir)
		dirs = append(dirs, w.agentDirs()...)
	}

	var existing []string
	for _, d := range dirs {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			existing = append(existing, d)
		}
	}
	return existing
}

func (w *Watcher) agentDirs() []string {
	var dirs []string
	seen := make(map[string]bool)

	for _, ag := range w.cfg.Agents {
		dir := config.ResolveGlobalPath(ag)
		if !seen[dir] {
			dirs = append(dirs, dir)
			seen[dir] = true
		}
	}

	for _, proj := range w.cfg.Projects {
		agents := proj.Agents
		if len(agents) == 0 {
			agents = agent.DetectProjectAgents(proj.Path, w.cfg.Agents)
		}
		for _, agName := range agents {
			dir := agent.ResolveProjectSkillsDir(proj.Path, agName, w.cfg.Agents)
			if dir != "" && !seen[dir] {
				dirs = append(dirs, dir)
				seen[dir] = true
			}
		}
	}
	return dirs
}

func (w *Watcher) addRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return w.watcher.Add(path)
		}
		return nil
	})
}

func (w *Watcher) eventLoop() {
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}

	for {
		select {
		case <-w.stopCh:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				w.mu.Lock()
				w.pending[event.Name] = time.Now()
				w.mu.Unlock()
				timer.Reset(debounceDelay)
			}
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					w.watcher.Add(event.Name)
				}
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("watcher error", "error", err)
		case <-timer.C:
			w.processBatch()
		}
	}
}

func (w *Watcher) processBatch() {
	w.mu.Lock()
	changes := make(map[string]time.Time, len(w.pending))
	for k, v := range w.pending {
		changes[k] = v
	}
	w.pending = make(map[string]time.Time)
	w.mu.Unlock()

	if len(changes) == 0 {
		return
	}

	centralDir := config.SkillsHome()
	fmt.Println(infoStyle.Render(fmt.Sprintf("\n[%s] Detected %d change(s), syncing...", time.Now().Format("15:04:05"), len(changes))))

	isCentralChange := false
	for path := range changes {
		rel, err := filepath.Rel(centralDir, path)
		if err == nil && rel != "" && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, "..") {
			isCentralChange = true
			break
		}
	}

	switch w.direction {
	case config.WatchCentralToAgents:
		w.pushAll()
	case config.WatchAgentsToCentral:
		w.pullAll()
	case config.WatchBidirectional:
		if isCentralChange {
			w.pushAll()
		} else {
			w.pullAll()
		}
	}
}

// pushAll and pullAll delegate to the sync package so the watcher and the
// push/pull commands share one implementation. Force is set because watch
// auto-syncs without interactive conflict prompts.
func (w *Watcher) pushAll() {
	targets := syncer.ResolveTargets(w.cfg, nil, nil, true)
	syncer.Push(targets, syncer.SyncOptions{Force: true})
}

func (w *Watcher) pullAll() {
	targets := syncer.ResolveTargets(w.cfg, nil, nil, true)
	syncer.Pull(targets, syncer.SyncOptions{Force: true})
}
