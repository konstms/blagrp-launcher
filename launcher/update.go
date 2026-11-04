package main

// Обновление файлов сборки. Код не зависит от Windows, поэтому его можно
// проверять тестами на любой системе.

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Манифест сборки, лежит в ветке pack: manifest.json.
type Manifest struct {
	Version  int            `json:"version"`
	Server   string         `json:"server"`
	Launcher LauncherInfo   `json:"launcher"`
	Files    []ManifestFile `json:"files"`
	// Файлы, которые нужно переименовать, если их ещё нет (например,
	// оригинальный vorbisFile.dll -> vorbisHooked.dll перед установкой ASI-загрузчика).
	Renames []RenameRule `json:"renames,omitempty"`
}

type LauncherInfo struct {
	Version string `json:"v"`
	Hash    string `json:"h"`
	Size    int64  `json:"s"`
	ZSize   int64  `json:"z"`
}

type ManifestFile struct {
	Path  string `json:"p"`
	Size  int64  `json:"s"`
	Hash  string `json:"h"`
	ZSize int64  `json:"z"`
}

type RenameRule struct {
	// Если файла To нет, а From есть и его хеш НЕ равен IfNotHash, From переименовывается в To.
	From      string `json:"from"`
	To        string `json:"to"`
	IfNotHash string `json:"if_not"`
	// Если задан — переименовывать только файл ровно такого размера.
	IfSize int64 `json:"if_size,omitempty"`
}

// Состояние: какие файлы поставил лаунчер (чтобы не пересчитывать хеши и
// чтобы удалять только своё).
type State struct {
	Files map[string]StateFile `json:"files"`
}

type StateFile struct {
	Size  int64  `json:"s"`
	Hash  string `json:"h"`
	MTime int64  `json:"m"`
}

const backupDir = "_blagrp_backup"
const tmpDir = "_blagrp_tmp"

var httpClient = &http.Client{Timeout: 0}

func fetchManifest(ctx context.Context, base string) (*Manifest, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", base+"manifest.json?t="+fmt.Sprint(time.Now().Unix()), nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("сервер обновлений ответил %d", resp.StatusCode)
	}
	var m Manifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, fmt.Errorf("повреждён список файлов: %v", err)
	}
	for _, f := range m.Files {
		if !safeRel(f.Path) {
			return nil, fmt.Errorf("недопустимый путь в списке файлов: %q", f.Path)
		}
	}
	for _, r := range m.Renames {
		if !safeRel(r.From) || !safeRel(r.To) {
			return nil, fmt.Errorf("недопустимый путь в списке файлов")
		}
	}
	return &m, nil
}

// safeRel: только относительные пути внутри папки игры.
func safeRel(p string) bool {
	if p == "" || strings.Contains(p, "\\") || strings.HasPrefix(p, "/") || strings.Contains(p, ":") {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func objectURL(base, hash string) string {
	return base + "o/" + hash[:2] + "/" + hash + ".gz"
}

func loadState(path string) *State {
	s := &State{Files: map[string]StateFile{}}
	b, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(b, s)
		if s.Files == nil {
			s.Files = map[string]StateFile{}
		}
	}
	return s
}

func saveState(path string, s *State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(s, "", " ")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Plan — что нужно сделать, чтобы папка игры совпала с манифестом.
type Plan struct {
	Download []ManifestFile
	Remove   []string // файлы, поставленные лаунчером ранее и больше не нужные
	Bytes    int64    // сколько скачать (сжатые байты)
}

func (p *Plan) Empty() bool { return len(p.Download) == 0 && len(p.Remove) == 0 }

// checkProgress вызывается при проверке файлов: done из total.
func makePlan(game string, m *Manifest, st *State, checkProgress func(done, total int)) (*Plan, error) {
	p := &Plan{}
	want := map[string]bool{}
	for i, f := range m.Files {
		if checkProgress != nil {
			checkProgress(i, len(m.Files))
		}
		want[strings.ToLower(f.Path)] = true
		full := filepath.Join(game, filepath.FromSlash(f.Path))
		fi, err := os.Stat(full)
		if err == nil && !fi.IsDir() && fi.Size() == f.Size {
			if s, ok := st.Files[f.Path]; ok && s.Hash == f.Hash && s.Size == f.Size && s.MTime == fi.ModTime().UnixNano() {
				continue
			}
			h, err := hashFile(full)
			if err == nil && h == f.Hash {
				st.Files[f.Path] = StateFile{Size: f.Size, Hash: f.Hash, MTime: fi.ModTime().UnixNano()}
				continue
			}
		}
		p.Download = append(p.Download, f)
		p.Bytes += f.ZSize
	}
	for path := range st.Files {
		if !want[strings.ToLower(path)] {
			p.Remove = append(p.Remove, path)
		}
	}
	sort.Strings(p.Remove)
	// Самые большие файлы качаем первыми: так прогресс честнее.
	sort.Slice(p.Download, func(i, j int) bool { return p.Download[i].ZSize > p.Download[j].ZSize })
	return p, nil
}

// applyRenames выполняет правила переименования до установки файлов.
func applyRenames(game string, m *Manifest) error {
	for _, r := range m.Renames {
		from := filepath.Join(game, filepath.FromSlash(r.From))
		to := filepath.Join(game, filepath.FromSlash(r.To))
		if _, err := os.Stat(to); err == nil {
			continue
		}
		fi, err := os.Stat(from)
		if err != nil || (r.IfSize > 0 && fi.Size() != r.IfSize) {
			continue
		}
		if r.IfNotHash != "" {
			if h, _ := hashFile(from); h == r.IfNotHash {
				continue
			}
		}
		if err := copyFile(from, to); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst + ".tmp")
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(dst+".tmp", dst)
}

type Progress struct {
	Done  atomic.Int64 // скачано сжатых байт
	Total int64
	Files atomic.Int32
	Count int
}

// applyPlan качает и ставит файлы. Чужие (не наши) файлы перед заменой
// копируются в _blagrp_backup, чтобы их можно было вернуть.
func applyPlan(ctx context.Context, game, base string, m *Manifest, st *State, statePath string, p *Plan, prog *Progress) error {
	if err := applyRenames(game, m); err != nil {
		return wrapFsErr(err)
	}
	tmp := filepath.Join(game, tmpDir)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return wrapFsErr(err)
	}
	defer os.RemoveAll(tmp)

	var mu sync.Mutex
	jobs := make(chan ManifestFile)
	errc := make(chan error, 1)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				err := installFile(ctx, game, base, f, prog)
				if err != nil {
					select {
					case errc <- err:
					default:
					}
					cancel()
					continue
				}
				full := filepath.Join(game, filepath.FromSlash(f.Path))
				if fi, err := os.Stat(full); err == nil {
					mu.Lock()
					st.Files[f.Path] = StateFile{Size: f.Size, Hash: f.Hash, MTime: fi.ModTime().UnixNano()}
					mu.Unlock()
				}
				prog.Files.Add(1)
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, f := range p.Download {
			select {
			case jobs <- f:
			case <-ctx.Done():
				return
			}
		}
	}()
	wg.Wait()
	mu.Lock()
	_ = saveState(statePath, st)
	mu.Unlock()
	select {
	case err := <-errc:
		return err
	default:
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// Удаляем ставшие ненужными свои файлы и возвращаем оригиналы из бэкапа.
	for _, rel := range p.Remove {
		full := filepath.Join(game, filepath.FromSlash(rel))
		_ = os.Remove(full)
		bak := filepath.Join(game, backupDir, filepath.FromSlash(rel))
		if _, err := os.Stat(bak); err == nil {
			_ = copyFile(bak, full)
		}
		delete(st.Files, rel)
	}
	return wrapFsErr(saveState(statePath, st))
}

func installFile(ctx context.Context, game, base string, f ManifestFile, prog *Progress) error {
	full := filepath.Join(game, filepath.FromSlash(f.Path))
	// Бэкап чужого файла (один раз).
	if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
		bak := filepath.Join(game, backupDir, filepath.FromSlash(f.Path))
		if _, err := os.Stat(bak); errors.Is(err, os.ErrNotExist) {
			if err := copyFile(full, bak); err != nil {
				return wrapFsErr(err)
			}
		}
	}
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
		n, err := downloadTo(ctx, objectURL(base, f.Hash), full, f, prog)
		if err == nil {
			return nil
		}
		prog.Done.Add(-n)
		lastErr = err
		if isFsErr(err) {
			return err
		}
	}
	return fmt.Errorf("не удалось скачать %s: %v", f.Path, lastErr)
}

type countingReader struct {
	r    io.Reader
	n    int64
	prog *Progress
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	c.prog.Done.Add(int64(n))
	return n, err
}

type fsError struct{ error }

func wrapFsErr(err error) error {
	if err == nil {
		return nil
	}
	return fsError{err}
}

func isFsErr(err error) bool {
	var fe fsError
	return errors.As(err, &fe)
}

func downloadTo(ctx context.Context, url, full string, f ManifestFile, prog *Progress) (int64, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("ответ %d", resp.StatusCode)
	}
	cr := &countingReader{r: resp.Body, prog: prog}
	zr, err := gzip.NewReader(cr)
	if err != nil {
		return cr.n, err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return cr.n, wrapFsErr(err)
	}
	tmp := full + ".blagrp-tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return cr.n, wrapFsErr(err)
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(out, h), zr)
	cerr := out.Close()
	if err != nil {
		os.Remove(tmp)
		return cr.n, err
	}
	if cerr != nil {
		os.Remove(tmp)
		return cr.n, wrapFsErr(cerr)
	}
	if n != f.Size || hex.EncodeToString(h.Sum(nil)) != f.Hash {
		os.Remove(tmp)
		return cr.n, fmt.Errorf("файл повреждён при скачивании")
	}
	if err := replaceFile(tmp, full); err != nil {
		os.Remove(tmp)
		return cr.n, wrapFsErr(err)
	}
	return cr.n, nil
}

// replaceFile заменяет файл, даже если старый занят (переименовывает его в сторону).
func replaceFile(tmp, full string) error {
	if err := os.Rename(tmp, full); err == nil {
		return nil
	}
	old := full + ".blagrp-old"
	_ = os.Remove(old)
	if err := os.Rename(full, old); err != nil {
		return err
	}
	if err := os.Rename(tmp, full); err != nil {
		_ = os.Rename(old, full)
		return err
	}
	_ = os.Remove(old)
	return nil
}
