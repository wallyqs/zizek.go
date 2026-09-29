package zizek

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"sync"
)

// Where a counterexample goes after the run that found it.
//
// A database is a cache that never needs invalidating. It stores raw
// serialized choice streams — the bytes under a reproduce blob, not the blob —
// keyed by a property's name, so the next run replays yesterday's failure
// instead of hoping to rediscover it.
//
// Both digests are the project's compiled keyed-draw hash (ir/keys.zir), so a
// directory written by this package is byte-for-byte the directory the Python
// and Ruby packages write, under the same reuse rules.

// DefaultDatabasePath is where a database goes when a caller asks for one
// without saying where.
const DefaultDatabasePath = ".zizek/examples"

// databaseNameFile names a key directory, for a human reading a checkout. The
// engine writes it and never reads it.
const databaseNameFile = ".name"

// Database stores counterexamples between runs.
//
// The key is a property's name, not its digest, so a backend over Redis or S3
// can key however its store prefers; the digest rule is DirectoryDatabase's
// business, and DatabaseDigest exposes it for a backend that wants the same
// layout. Every method is best-effort: a database that cannot be read or
// written must not fail a test that would otherwise pass, so none of them
// returns an error.
type Database interface {
	// Fetch returns every stored stream for this name, in any order. Sorting
	// is the engine's job.
	Fetch(key string) [][]byte
	// Save stores a stream. Storing one already present does nothing.
	Save(key string, value []byte)
	// Delete removes a stream. Removing one that is absent does nothing.
	Delete(key string, value []byte)
}

// DatabaseDigest is the sixteen hex digits a name or a stream is stored under.
//
// FNV-1a 64 over the tagged encoding compiled from ir/keys.zir, which is the
// same hash a keyed draw uses and the reason a database directory is portable
// across languages.
func DatabaseDigest(value []byte) string {
	return fmt.Sprintf("%016x", Key(value))
}

// sortDatabaseEntries orders stored entries the way they are replayed:
// shortest first, then bytes. The same order the shrinker prefers, so the
// simplest stored counterexample is the first one a run pays for.
func sortDatabaseEntries(entries [][]byte) {
	sort.SliceStable(entries, func(i, j int) bool {
		if len(entries[i]) != len(entries[j]) {
			return len(entries[i]) < len(entries[j])
		}
		return bytes.Compare(entries[i], entries[j]) < 0
	})
}

// InMemoryDatabase forgets when the process exits.
//
// For a suite that wants replay within a run — or a test about the database
// itself — without leaving anything behind. Safe for concurrent use, because a
// package-level test may run its properties in parallel.
type InMemoryDatabase struct {
	mu      sync.Mutex
	entries map[string][][]byte
}

// NewInMemoryDatabase returns an empty in-memory database.
func NewInMemoryDatabase() *InMemoryDatabase {
	return &InMemoryDatabase{entries: map[string][][]byte{}}
}

func (db *InMemoryDatabase) Fetch(key string) [][]byte {
	db.mu.Lock()
	defer db.mu.Unlock()
	stored := db.entries[key]
	out := make([][]byte, len(stored))
	for at, entry := range stored {
		out[at] = append([]byte(nil), entry...)
	}
	return out
}

func (db *InMemoryDatabase) Save(key string, value []byte) {
	db.mu.Lock()
	defer db.mu.Unlock()
	for _, entry := range db.entries[key] {
		if bytes.Equal(entry, value) {
			return
		}
	}
	db.entries[key] = append(db.entries[key], append([]byte(nil), value...))
}

func (db *InMemoryDatabase) Delete(key string, value []byte) {
	db.mu.Lock()
	defer db.mu.Unlock()
	stored := db.entries[key]
	for at, entry := range stored {
		if !bytes.Equal(entry, value) {
			continue
		}
		stored = append(stored[:at], stored[at+1:]...)
		if len(stored) == 0 {
			delete(db.entries, key)
		} else {
			db.entries[key] = stored
		}
		return
	}
}

// DirectoryDatabase is a directory of counterexamples, one subdirectory per
// property.
//
// The layout is shared with every other zizek package, so a directory checked
// into version control replays in any of them:
//
//	.zizek/examples/
//	  9dc0c2b1f8a17c4e/          # the key's digest
//	    .name                    # the property's name, for a human
//	    3a6e0f5c9b2d1877         # a stored stream
//
// Every operation is best-effort: a database that cannot be read or written
// must not fail a test that would otherwise pass.
type DirectoryDatabase struct {
	Path string
}

// NewDirectoryDatabase returns a database rooted at path. An empty path
// selects DefaultDatabasePath.
func NewDirectoryDatabase(path string) *DirectoryDatabase {
	if path == "" {
		path = DefaultDatabasePath
	}
	return &DirectoryDatabase{Path: path}
}

// KeyPath is the directory holding one property's stored counterexamples.
func (db *DirectoryDatabase) KeyPath(key string) string {
	return filepath.Join(db.Path, DatabaseDigest([]byte(key)))
}

// ValuePath is the file one stored stream lives in.
func (db *DirectoryDatabase) ValuePath(key string, value []byte) string {
	return filepath.Join(db.KeyPath(key), DatabaseDigest(value))
}

func (db *DirectoryDatabase) Fetch(key string) [][]byte {
	directory := db.KeyPath(key)
	names, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}
	var entries [][]byte
	for _, name := range names {
		if !isDatabaseEntry(name.Name()) {
			continue
		}
		payload, err := os.ReadFile(filepath.Join(directory, name.Name()))
		if err != nil {
			// Deleted between the listing and the read: a concurrent run
			// collected it, which is exactly what it is supposed to do.
			continue
		}
		entries = append(entries, payload)
	}
	return entries
}

func (db *DirectoryDatabase) Save(key string, value []byte) {
	directory := db.KeyPath(key)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return
	}
	nameFile := filepath.Join(directory, databaseNameFile)
	if _, err := os.Stat(nameFile); err != nil {
		writeDatabaseFile(nameFile, []byte(key))
	}
	entry := filepath.Join(directory, DatabaseDigest(value))
	if _, err := os.Stat(entry); err != nil {
		writeDatabaseFile(entry, value)
	}
}

func (db *DirectoryDatabase) Delete(key string, value []byte) {
	if err := os.Remove(db.ValuePath(key, value)); err != nil {
		return
	}
	// An emptied key directory takes its .name with it, so a database that
	// collected everything leaves no husk behind — but only once the last
	// entry is gone, or a directory that still holds counterexamples would
	// lose the label that says whose they are.
	directory := db.KeyPath(key)
	children, err := os.ReadDir(directory)
	if err != nil {
		return
	}
	for _, child := range children {
		if isDatabaseEntry(child.Name()) {
			return
		}
	}
	os.Remove(filepath.Join(directory, databaseNameFile))
	os.Remove(directory)
}

// isDatabaseEntry reports whether a filename is an entry: sixteen lowercase
// hex digits and nothing else. Anything else in a key directory belongs to
// something other than the engine — .name, a .gitignore, an editor's swap file
// — and is ignored rather than replayed.
func isDatabaseEntry(name string) bool {
	if len(name) != 16 {
		return false
	}
	for _, char := range name {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

// writeDatabaseFile writes through a temporary file, so a reader never sees a
// partial entry.
func writeDatabaseFile(path string, payload []byte) {
	handle, err := os.CreateTemp(filepath.Dir(path), ".zizek")
	if err != nil {
		return
	}
	temporary := handle.Name()
	_, writeErr := handle.Write(payload)
	closeErr := handle.Close()
	if writeErr != nil || closeErr != nil {
		os.Remove(temporary)
		return
	}
	if err := os.Rename(temporary, path); err != nil {
		os.Remove(temporary)
	}
}

// derivedDatabaseKey names a property that did not name itself.
//
// Deliberately local to Go: a package-qualified function name says nothing to
// the Python or Ruby suite testing the same behaviour, and a key that
// pretended otherwise would file a counterexample where the other language
// will never look for it. Set Config.DatabaseKey to share one.
func derivedDatabaseKey(property Property) string {
	pointer := reflect.ValueOf(property).Pointer()
	if fn := runtime.FuncForPC(pointer); fn != nil {
		return fn.Name()
	}
	return fmt.Sprintf("zizek.property@%d", pointer)
}
