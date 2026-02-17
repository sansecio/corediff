package manifest

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"
	"syscall"
)

// Manifest tracks which package@version pairs have been indexed, which
// packages are replaced by a monorepo, and which packages are tracked for
// automatic updates. Backed by an append-only text file.
// Indexed entries use "package@version" format; replace entries use "replace:package" format;
// tracked entries use "track:package" format.
type Manifest struct {
	mu       sync.Mutex
	path     string
	indexed  map[string]struct{} // "package@version"
	replaced map[string]struct{} // "package-name" (no version)
	tracked  map[string]struct{} // bare package name or git URL
	file     *os.File            // kept open for flock
}

// PathFromDB derives the manifest path from a database path.
// Replaces .db suffix with .manifest, or appends .manifest if no .db suffix.
func PathFromDB(dbPath string) string {
	if base, ok := strings.CutSuffix(dbPath, ".db"); ok {
		return base + ".manifest"
	}
	return dbPath + ".manifest"
}

// Load opens or creates a manifest file at path. The file is locked (flock)
// for cross-process safety. Call Close when done.
func Load(path string) (*Manifest, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening manifest %s: %w", path, err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("locking manifest %s: %w", path, err)
	}

	indexed := make(map[string]struct{})
	replaced := make(map[string]struct{})
	tracked := make(map[string]struct{})
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if pkg, ok := strings.CutPrefix(line, "replace:"); ok {
			replaced[pkg] = struct{}{}
		} else if pkg, ok := strings.CutPrefix(line, "track:"); ok {
			tracked[pkg] = struct{}{}
		} else if strings.Contains(line, "@") {
			indexed[line] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		f.Close()
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}

	return &Manifest{
		path:     path,
		indexed:  indexed,
		replaced: replaced,
		tracked:  tracked,
		file:     f,
	}, nil
}

// IsIndexed reports whether the given package@version has been indexed.
func (m *Manifest) IsIndexed(pkg, version string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.indexed[pkg+"@"+version]
	return ok
}

// MarkIndexed records that package@version has been indexed.
// The entry is appended to the file immediately.
func (m *Manifest) MarkIndexed(pkg, version string) error {
	key := pkg + "@" + version

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.indexed[key]; ok {
		return nil // already recorded
	}

	if _, err := fmt.Fprintln(m.file, key); err != nil {
		return fmt.Errorf("writing to manifest: %w", err)
	}
	m.indexed[key] = struct{}{}
	return nil
}

// IsReplaced reports whether the given package is replaced by a monorepo.
func (m *Manifest) IsReplaced(pkg string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.replaced[pkg]
	return ok
}

// MarkReplaced records that a package is replaced by a monorepo.
// The entry is appended to the file immediately.
func (m *Manifest) MarkReplaced(pkg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.replaced[pkg]; ok {
		return nil // already recorded
	}

	if _, err := fmt.Fprintln(m.file, "replace:"+pkg); err != nil {
		return fmt.Errorf("writing to manifest: %w", err)
	}
	m.replaced[pkg] = struct{}{}
	return nil
}

// MarkTracked records that a package should be tracked for automatic updates.
// The entry is appended to the file immediately.
func (m *Manifest) MarkTracked(pkg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tracked[pkg]; ok {
		return nil // already recorded
	}

	if _, err := fmt.Fprintln(m.file, "track:"+pkg); err != nil {
		return fmt.Errorf("writing to manifest: %w", err)
	}
	m.tracked[pkg] = struct{}{}
	return nil
}

// TrackedPackages returns all packages marked for tracking.
func (m *Manifest) TrackedPackages() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Collect(maps.Keys(m.tracked))
}

// Packages returns the set of unique package names in the manifest.
func (m *Manifest) Packages() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	pkgs := make(map[string]struct{})
	for key := range m.indexed {
		if idx := strings.LastIndex(key, "@"); idx > 0 {
			pkgs[key[:idx]] = struct{}{}
		}
	}

	return slices.Collect(maps.Keys(pkgs))
}

// Close releases the file lock and closes the underlying file.
func (m *Manifest) Close() error {
	if m.file == nil {
		return nil
	}
	if err := syscall.Flock(int(m.file.Fd()), syscall.LOCK_UN); err != nil {
		m.file.Close()
		return fmt.Errorf("unlocking manifest: %w", err)
	}
	return m.file.Close()
}
