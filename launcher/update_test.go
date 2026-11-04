package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakePack struct {
	objs map[string][]byte
	m    Manifest
}

func (p *fakePack) add(path string, data []byte) {
	h := sha256.Sum256(data)
	hs := hex.EncodeToString(h[:])
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write(data)
	zw.Close()
	p.objs[hs] = buf.Bytes()
	p.m.Files = append(p.m.Files, ManifestFile{Path: path, Size: int64(len(data)), Hash: hs, ZSize: int64(buf.Len())})
}

func (p *fakePack) server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/manifest.json") {
			json.NewEncoder(w).Encode(p.m)
			return
		}
		name := filepath.Base(r.URL.Path)
		if b, ok := p.objs[strings.TrimSuffix(name, ".gz")]; ok {
			w.Write(b)
			return
		}
		http.NotFound(w, r)
	}))
}

func run(t *testing.T, game, base string) *Plan {
	m, err := fetchManifest(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	st := loadState(statePath(game))
	plan, err := makePlan(game, m, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	prog := &Progress{Total: plan.Bytes}
	if err := applyPlan(context.Background(), game, base, m, st, statePath(game), plan, prog); err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestInstallUpdateRemove(t *testing.T) {
	game := t.TempDir()
	os.WriteFile(filepath.Join(game, "stream.ini"), []byte("original"), 0o644)
	os.WriteFile(filepath.Join(game, "vorbisFile.dll"), []byte("rockstar-vorbis"), 0o644)

	p := &fakePack{objs: map[string][]byte{}}
	p.add("stream.ini", []byte("memory 262144"))
	p.add("modloader/gai/a.dff", bytes.Repeat([]byte("x"), 100000))
	p.add("vorbisFile.dll", []byte("asi-loader"))
	h := sha256.Sum256([]byte("asi-loader"))
	p.m.Renames = []RenameRule{{From: "vorbisFile.dll", To: "vorbisHooked.dll", IfNotHash: hex.EncodeToString(h[:])}}
	srv := p.server()
	defer srv.Close()
	base := srv.URL + "/"

	plan := run(t, game, base)
	if len(plan.Download) != 3 {
		t.Fatalf("want 3 downloads, got %d", len(plan.Download))
	}
	if b, _ := os.ReadFile(filepath.Join(game, "stream.ini")); string(b) != "memory 262144" {
		t.Fatal("stream.ini not replaced")
	}
	if b, _ := os.ReadFile(filepath.Join(game, backupDir, "stream.ini")); string(b) != "original" {
		t.Fatal("no backup")
	}
	if b, _ := os.ReadFile(filepath.Join(game, "vorbisHooked.dll")); string(b) != "rockstar-vorbis" {
		t.Fatal("rename rule failed")
	}
	// Второй запуск — ничего не качаем.
	if plan := run(t, game, base); !plan.Empty() {
		t.Fatalf("second run not empty: %+v", plan)
	}
	// Игрок испортил файл — перекачиваем только его.
	os.WriteFile(filepath.Join(game, "modloader/gai/a.dff"), bytes.Repeat([]byte("y"), 100000), 0o644)
	if plan := run(t, game, base); len(plan.Download) != 1 {
		t.Fatalf("want 1 redownload, got %d", len(plan.Download))
	}
	// Убираем stream.ini из сборки — должен вернуться оригинал.
	p.m.Files = p.m.Files[1:]
	plan = run(t, game, base)
	if len(plan.Remove) != 1 {
		t.Fatalf("want 1 removal, got %v", plan.Remove)
	}
	if b, _ := os.ReadFile(filepath.Join(game, "stream.ini")); string(b) != "original" {
		t.Fatal("original not restored")
	}
}

func TestCorruptObjectRejected(t *testing.T) {
	game := t.TempDir()
	p := &fakePack{objs: map[string][]byte{}}
	p.add("a.txt", []byte("hello"))
	// подменяем содержимое объекта
	for k := range p.objs {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		zw.Write([]byte("evil!"))
		zw.Close()
		p.objs[k] = buf.Bytes()
	}
	srv := p.server()
	defer srv.Close()
	m, _ := fetchManifest(context.Background(), srv.URL+"/")
	st := loadState(statePath(game))
	plan, _ := makePlan(game, m, st, nil)
	err := applyPlan(context.Background(), game, srv.URL+"/", m, st, statePath(game), plan, &Progress{})
	if err == nil {
		t.Fatal("corrupt file accepted")
	}
	if _, err := os.Stat(filepath.Join(game, "a.txt")); err == nil {
		t.Fatal("corrupt file written")
	}
}

func TestSafeRel(t *testing.T) {
	for _, bad := range []string{"../x", "/etc/x", "a/../../b", `a\b`, "C:/x", "", "a//b"} {
		if safeRel(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
	if !safeRel("modloader/gai/a.dff") {
		t.Error("good path rejected")
	}
}
