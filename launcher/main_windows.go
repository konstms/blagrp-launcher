package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type Phase int

const (
	phCheck Phase = iota
	phNeedGame
	phBadVersion
	phNeedSamp
	phNeedUpdate
	phUpdating
	phReady
	phError
	phSelfUpdate
	phLaunching
)

type UI struct {
	mu       sync.Mutex
	phase    Phase
	status   string
	progress float64 // 0..1, <0 — скрыть
	online   string
	onlineOK bool
	plan     *Plan
	manifest *Manifest
	prog     *Progress
	started  time.Time

	hwnd, edit    uintptr
	cfg           *Config
	scale         float64
	W, H          int32
	hover         int // 0 нет, 1 кнопка, 2 ссылка «изменить»
	fonts         map[string]uintptr
	editBrush     uintptr
	cursorArrow   uintptr
	cursorHand    uintptr
	busy          bool
}

var ui = &UI{progress: -1, online: "Проверяю сервер…"}

func (u *UI) set(f func()) {
	u.mu.Lock()
	f()
	u.mu.Unlock()
	if u.hwnd != 0 {
		pPostMessageW.Call(u.hwnd, WM_APP, 0, 0)
	}
}

func (u *UI) sc(v int) int32 { return int32(float64(v)*u.scale + 0.5) }

func main() {
	runtime.LockOSThread()
	pSetProcessDPIAware.Call()
	cleanupOldExe()

	ui.cfg = loadConfig()
	if ui.cfg.Nick == "" {
		ui.cfg.Nick = regGetString(`Software\SAMP`, "PlayerName")
	}
	if checkGame(ui.cfg.Game) == GameMissing {
		ui.cfg.Game = detectGame()
	}

	hdc, _, _ := pGetDC.Call(0)
	dpi, _, _ := pGetDeviceCaps.Call(hdc, 88) // LOGPIXELSX
	pReleaseDC.Call(0, hdc)
	ui.scale = float64(dpi) / 96
	if ui.scale < 1 {
		ui.scale = 1
	}
	ui.W, ui.H = ui.sc(880), ui.sc(540)
	ui.createFonts()
	ui.editBrush, _, _ = pCreateSolidBrush.Call(rgb(20, 31, 51))
	ui.cursorArrow, _, _ = pLoadCursorW.Call(0, IDC_ARROW)
	ui.cursorHand, _, _ = pLoadCursorW.Call(0, IDC_HAND)

	inst, _, _ := pGetModuleHandleW.Call(0)
	icon, _, _ := pLoadImageW.Call(inst, 1, 1, 0, 0, 0x40) // ресурс иконки №1, если есть
	cls := u16("BlagRPLauncher")
	wc := WNDCLASSEXW{
		WndProc:   syscall.NewCallback(wndProc),
		Instance:  inst,
		Cursor:    ui.cursorArrow,
		ClassName: cls,
		Icon:      icon,
		IconSm:    icon,
	}
	wc.Size = uint32(unsafe.Sizeof(wc))
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	style := uintptr(WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX | WS_CLIPCHILDREN)
	r := RECT{0, 0, ui.W, ui.H}
	pAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&r)), style, 0, 0)
	ww, wh := r.Right-r.Left, r.Bottom-r.Top
	sw, _, _ := pGetSystemMetrics.Call(0)
	sh, _, _ := pGetSystemMetrics.Call(1)
	x, y := (int32(sw)-ww)/2, (int32(sh)-wh)/2
	hwnd, _, _ := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(u16(AppTitle+" — лаунчер"))),
		style, uintptr(x), uintptr(y), uintptr(ww), uintptr(wh), 0, 0, inst, 0)
	ui.hwnd = hwnd
	dark := int32(1)
	pDwmSetWindowAttribute.Call(hwnd, 20, uintptr(unsafe.Pointer(&dark)), 4)

	ui.edit, _, _ = pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(u16("EDIT"))), uintptr(unsafe.Pointer(u16(ui.cfg.Nick))),
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, 0, 0, 10, 10, hwnd, 100, inst, 0)
	pSendMessageW.Call(ui.edit, WM_SETFONT, ui.fonts["edit"], 1)
	pSendMessageW.Call(ui.edit, EM_LIMITTEXT, 20, 0)
	ex, ey, _, _ := ui.editRect()
	pMoveWindow.Call(ui.edit, uintptr(ex+ui.sc(12)), uintptr(ey+ui.sc(9)), uintptr(ui.sc(276)), uintptr(ui.sc(24)), 1)

	pShowWindow.Call(hwnd, SW_SHOW)
	pUpdateWindow.Call(hwnd)
	pSetTimer.Call(hwnd, 1, 150, 0)

	go onlineLoop()
	go runCheck(false)

	var msg MSG
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (u *UI) createFonts() {
	mk := func(px int, weight int, face string) uintptr {
		f, _, _ := pCreateFontW.Call(uintptr(uint32(-u.sc(px))), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(u16(face))))
		return f
	}
	u.fonts = map[string]uintptr{
		"kicker": mk(13, 700, "Segoe UI"),
		"title":  mk(58, 900, "Segoe UI Black"),
		"sub":    mk(22, 600, "Segoe UI Semibold"),
		"body":   mk(15, 400, "Segoe UI"),
		"small":  mk(12, 700, "Segoe UI"),
		"edit":   mk(17, 600, "Segoe UI Semibold"),
		"path":   mk(14, 400, "Segoe UI"),
		"button": mk(21, 800, "Segoe UI"),
		"big":    mk(300, 900, "Segoe UI Black"),
		"pill":   mk(14, 600, "Segoe UI Semibold"),
	}
}

// Геометрия элементов (в пикселях клиентской области).
func (u *UI) panelRect() (int32, int32, int32, int32) {
	return u.sc(28), u.H - u.sc(206), u.W - u.sc(28), u.H - u.sc(28)
}
func (u *UI) editRect() (int32, int32, int32, int32) {
	_, t, _, _ := u.panelRect()
	return u.sc(56), t + u.sc(44), u.sc(56 + 300), t + u.sc(44+42)
}
func (u *UI) pathRect() (int32, int32, int32, int32) {
	_, t, _, _ := u.panelRect()
	return u.sc(384), t + u.sc(44), u.W - u.sc(300), t + u.sc(44+42)
}
func (u *UI) linkRect() (int32, int32, int32, int32) {
	_, t, _, _ := u.panelRect()
	return u.sc(384), t + u.sc(90), u.sc(384 + 130), t + u.sc(90+20)
}
func (u *UI) buttonRect() (int32, int32, int32, int32) {
	_, t, _, _ := u.panelRect()
	return u.W - u.sc(56+220), t + u.sc(36), u.W - u.sc(56), t + u.sc(36+66)
}

func in(x, y int32, l, t, r, b int32) bool { return x >= l && x < r && y >= t && y < b }

func wndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	switch msg {
	case WM_ERASEBKGND:
		return 1
	case WM_PAINT:
		ui.paint(hwnd)
		return 0
	case WM_APP:
		pInvalidateRect.Call(hwnd, 0, 0)
		return 0
	case WM_TIMER:
		ui.mu.Lock()
		ph := ui.phase
		ui.mu.Unlock()
		if ph == phUpdating || ph == phCheck || ph == phSelfUpdate {
			pInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0
	case WM_CTLCOLOREDIT:
		pSetTextColor.Call(wparam, rgb(255, 255, 255))
		pSetBkColor.Call(wparam, rgb(20, 31, 51))
		return ui.editBrush
	case WM_SETCURSOR:
		if uint16(lparam) == 1 && ui.hover != 0 { // HTCLIENT
			pSetCursor.Call(ui.cursorHand)
			return 1
		}
	case WM_MOUSEMOVE:
		x, y := int32(int16(lparam)), int32(int16(lparam>>16))
		h := 0
		if l, t, r, b := ui.buttonRect(); in(x, y, l, t, r, b) && ui.buttonEnabled() {
			h = 1
		} else if l, t, r, b := ui.linkRect(); in(x, y, l, t, r, b) {
			h = 2
		}
		if h != ui.hover {
			ui.hover = h
			pInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0
	case WM_LBUTTONUP:
		x, y := int32(int16(lparam)), int32(int16(lparam>>16))
		if l, t, r, b := ui.buttonRect(); in(x, y, l, t, r, b) && ui.buttonEnabled() {
			ui.onButton()
		} else if l, t, r, b := ui.linkRect(); in(x, y, l, t, r, b) {
			ui.chooseFolder()
		}
		return 0
	case WM_CLOSE:
		ui.mu.Lock()
		ph := ui.phase
		ui.mu.Unlock()
		if ph == phUpdating {
			if msgBox(hwnd, "Идёт загрузка сборки. Прервать? Скачанное сохранится, продолжить можно позже.", AppTitle, MB_YESNO|MB_ICONWARNING) != IDYES {
				return 0
			}
		}
		ui.cfg.Nick = strings.TrimSpace(getText(ui.edit))
		saveConfig(ui.cfg)
	case WM_DESTROY:
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return r
}

func (u *UI) buttonEnabled() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	switch u.phase {
	case phCheck, phUpdating, phSelfUpdate, phLaunching:
		return false
	}
	return true
}

func (u *UI) buttonLabel() string {
	switch u.phase {
	case phCheck:
		return "ПРОВЕРКА…"
	case phNeedGame:
		return "ВЫБРАТЬ ПАПКУ"
	case phBadVersion:
		return "ВЫБРАТЬ ДРУГУЮ"
	case phNeedSamp:
		return "СКАЧАТЬ SA-MP"
	case phNeedUpdate:
		return "УСТАНОВИТЬ"
	case phUpdating:
		return fmt.Sprintf("ЗАГРУЗКА %d%%", int(u.progress*100))
	case phReady:
		return "ИГРАТЬ"
	case phError:
		return "ПОВТОРИТЬ"
	case phSelfUpdate:
		return "ОБНОВЛЕНИЕ…"
	case phLaunching:
		return "ЗАПУСК…"
	}
	return ""
}

func (u *UI) onButton() {
	u.mu.Lock()
	ph := u.phase
	u.mu.Unlock()
	switch ph {
	case phNeedGame, phBadVersion:
		u.chooseFolder()
	case phNeedSamp:
		_ = shellOpen(SampURL, "", "")
		u.set(func() { u.status = "Установи SA-MP 0.3.7 в папку игры, затем вернись — лаунчер проверит сам." })
		go func() {
			for i := 0; i < 600; i++ {
				time.Sleep(3 * time.Second)
				if checkGame(u.cfg.Game) != GameNoSamp {
					runCheck(false)
					return
				}
			}
		}()
	case phNeedUpdate:
		go runUpdate()
	case phError:
		go runCheck(false)
	case phReady:
		u.launch()
	}
}

func (u *UI) chooseFolder() {
	if u.busyPhase() {
		return
	}
	dir := browseFolder(u.hwnd, "Выбери папку, где лежит gta_sa.exe (GTA San Andreas с SA-MP)")
	if dir == "" {
		return
	}
	switch checkGame(dir) {
	case GameMissing:
		msgBox(u.hwnd, "В этой папке нет gta_sa.exe. Выбери папку, где установлена GTA San Andreas.", AppTitle, MB_ICONWARNING)
		return
	}
	u.cfg.Game = dir
	saveConfig(u.cfg)
	go runCheck(false)
}

func (u *UI) busyPhase() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.phase == phUpdating || u.phase == phSelfUpdate || u.phase == phLaunching
}

func (u *UI) launch() {
	nick := strings.TrimSpace(getText(u.edit))
	if !nickRe.MatchString(nick) {
		msgBox(u.hwnd, "Ник должен быть от 3 до 20 символов: латинские буквы, цифры и знак _.\nНапример: Ivan_Petrov", AppTitle, MB_ICONWARNING)
		pSetFocus.Call(u.edit)
		return
	}
	if !rpNickRe.MatchString(nick) {
		if msgBox(u.hwnd, "На РП-сервере ник пишется как Имя_Фамилия, например Ivan_Petrov.\nВсё равно играть с ником «"+nick+"»?", AppTitle, MB_YESNO|MB_ICONWARNING) != IDYES {
			pSetFocus.Call(u.edit)
			return
		}
	}
	u.cfg.Nick = nick
	saveConfig(u.cfg)
	game := u.cfg.Game
	_ = regSetString(`Software\SAMP`, "PlayerName", nick)
	_ = regSetString(`Software\SAMP`, "gta_sa_exe", filepath.Join(game, "gta_sa.exe"))
	server := ServerAddr
	u.mu.Lock()
	if u.manifest != nil && u.manifest.Server != "" {
		server = u.manifest.Server
	}
	u.mu.Unlock()
	if err := shellOpen(filepath.Join(game, "samp.exe"), server, game); err != nil {
		msgBox(u.hwnd, "Не удалось запустить samp.exe: "+err.Error(), AppTitle, MB_ICONERROR)
		return
	}
	u.set(func() { u.phase = phLaunching; u.status = "Запускаю игру и подключаюсь к " + server + "…" })
	go func() {
		time.Sleep(2500 * time.Millisecond)
		pPostMessageW.Call(u.hwnd, WM_CLOSE, 0, 0)
	}()
}

func detectGame() string {
	var cands []string
	if p := regGetString(`Software\SAMP`, "gta_sa_exe"); p != "" {
		cands = append(cands, filepath.Dir(p))
	}
	for _, d := range []string{"C", "D", "E"} {
		cands = append(cands,
			d+`:\Games\GTA San Andreas`,
			d+`:\GTA San Andreas`,
			d+`:\Program Files (x86)\Rockstar Games\GTA San Andreas`,
			d+`:\Program Files\Rockstar Games\GTA San Andreas`,
		)
	}
	for _, c := range cands {
		if checkGame(c) == GameOK {
			return c
		}
	}
	for _, c := range cands {
		if checkGame(c) != GameMissing {
			return c
		}
	}
	return ""
}

func onlineLoop() {
	for {
		info, err := querySamp(ServerAddr)
		ui.set(func() {
			if err != nil {
				ui.online, ui.onlineOK = "Сервер недоступен", false
			} else {
				ui.online, ui.onlineOK = fmt.Sprintf("Онлайн: %d / %d", info.Players, info.Max), true
			}
		})
		time.Sleep(15 * time.Second)
	}
}

func mb(b int64) string {
	if b < 1024*1024 {
		return fmt.Sprintf("%.1f МБ", float64(b)/1024/1024)
	}
	return fmt.Sprintf("%d МБ", (b+512*1024)/1024/1024)
}

func runCheck(auto bool) {
	ui.set(func() { ui.phase = phCheck; ui.status = "Проверяю игру…"; ui.progress = -1 })
	game := ui.cfg.Game
	switch checkGame(game) {
	case GameMissing:
		ui.set(func() {
			ui.phase = phNeedGame
			ui.status = "Не нашёл GTA San Andreas. Укажи папку, где лежит gta_sa.exe."
		})
		return
	case GameBadVersion:
		ui.set(func() {
			ui.phase = phBadVersion
			ui.status = "Эта версия GTA не подходит для SA-MP. Нужна GTA San Andreas версии 1.0 US."
		})
		return
	case GameIncomplete:
		ui.set(func() {
			ui.phase = phBadVersion
			ui.status = "В этой папке неполная GTA (нет звука или моделей). Выбери папку с полностью установленной игрой."
		})
		return
	case GameNoSamp:
		ui.set(func() {
			ui.phase = phNeedSamp
			ui.status = "В папке игры нет SA-MP. Установи SA-MP 0.3.7 с официального сайта в эту папку."
		})
		return
	}
	ui.set(func() { ui.status = "Получаю список файлов сборки…" })
	m, err := fetchManifest(context.Background(), PackBaseURL)
	if err != nil {
		ui.set(func() {
			ui.phase = phError
			ui.status = "Нет связи с сервером обновлений: " + shortErr(err)
		})
		return
	}
	ui.set(func() { ui.manifest = m })
	if m.Launcher.Version != "" && m.Launcher.Version != Version && m.Launcher.Hash != "" {
		if selfUpdate(m) {
			return
		}
	}
	st := loadState(statePath(game))
	plan, err := makePlan(game, m, st, func(done, total int) {
		ui.mu.Lock()
		ui.progress = float64(done) / float64(max(total, 1))
		ui.status = fmt.Sprintf("Проверяю файлы сборки: %d из %d", done, total)
		ui.mu.Unlock()
	})
	_ = saveState(statePath(game), st)
	if err != nil {
		ui.set(func() { ui.phase = phError; ui.status = shortErr(err) })
		return
	}
	if plan.Empty() {
		ui.set(func() {
			ui.phase = phReady
			ui.plan = plan
			ui.progress = -1
			ui.status = "Сборка актуальна. Можно играть."
			if m.Build > 0 {
				ui.status = fmt.Sprintf("Установлена актуальная сборка №%d (от %s). Можно играть.", m.Build, m.Built)
			}
		})
		return
	}
	ui.set(func() {
		ui.phase = phNeedUpdate
		ui.plan = plan
		ui.progress = -1
		if len(st.Files) == 0 {
			ui.status = fmt.Sprintf("Нужно скачать сборку сервера: %s. Оригинальные файлы игры сохранятся в _blagrp_backup.", mb(plan.Bytes))
		} else {
			ui.status = fmt.Sprintf("Вышло обновление сборки: %s.", mb(plan.Bytes))
			if m.Build > 0 {
				ui.status = fmt.Sprintf("Вышла сборка №%d (от %s): скачать %s.", m.Build, m.Built, mb(plan.Bytes))
			}
		}
	})
	// Небольшие обновления ставим сразу.
	if len(st.Files) > 0 && plan.Bytes < 60*1024*1024 {
		runUpdate()
	}
}

func runUpdate() {
	ui.mu.Lock()
	plan, m := ui.plan, ui.manifest
	if ui.phase == phUpdating || plan == nil || m == nil {
		ui.mu.Unlock()
		return
	}
	prog := &Progress{Total: plan.Bytes, Count: len(plan.Download)}
	ui.phase = phUpdating
	ui.prog = prog
	ui.progress = 0
	ui.started = time.Now()
	ui.mu.Unlock()
	game := ui.cfg.Game
	st := loadState(statePath(game))

	done := make(chan error, 1)
	go func() { done <- applyPlan(context.Background(), game, PackBaseURL, m, st, statePath(game), plan, prog) }()
	tick := time.NewTicker(300 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case err := <-done:
			if err != nil {
				ui.set(func() {
					ui.phase = phError
					if isFsErr(err) && (os.IsPermission(err) || strings.Contains(strings.ToLower(err.Error()), "access")) {
						ui.status = "Нет доступа к папке игры. Запусти лаунчер от имени администратора или перенеси игру, например, в C:\\Games."
					} else {
						ui.status = "Ошибка загрузки: " + shortErr(err)
					}
				})
				return
			}
			runCheck(true)
			return
		case <-tick.C:
			d := prog.Done.Load()
			el := time.Since(ui.started).Seconds()
			ui.set(func() {
				if prog.Total > 0 {
					ui.progress = float64(d) / float64(prog.Total)
				}
				speed := ""
				if el > 1 {
					speed = fmt.Sprintf(" · %.1f МБ/с", float64(d)/el/1024/1024)
				}
				ui.status = fmt.Sprintf("Загрузка сборки: %s из %s%s · файлов %d из %d", mb(d), mb(prog.Total), speed, prog.Files.Load(), prog.Count)
			})
		}
	}
}

func shortErr(err error) string {
	s := err.Error()
	if len(s) > 140 {
		s = s[:140] + "…"
	}
	return s
}

// ---------- самообновление лаунчера ----------

func cleanupOldExe() {
	if exe, err := os.Executable(); err == nil {
		_ = os.Remove(exe + ".old")
	}
}

func selfUpdate(m *Manifest) bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	ui.set(func() { ui.phase = phSelfUpdate; ui.status = "Обновляю лаунчер до версии " + m.Launcher.Version + "…" })
	prog := &Progress{Total: m.Launcher.ZSize}
	f := ManifestFile{Path: "launcher.exe", Size: m.Launcher.Size, Hash: m.Launcher.Hash, ZSize: m.Launcher.ZSize}
	newPath := exe + ".new"
	if _, err := downloadTo(context.Background(), objectURL(PackBaseURL, f.Hash), newPath, f, prog); err != nil {
		return false
	}
	if err := os.Rename(exe, exe+".old"); err != nil {
		os.Remove(newPath)
		return false
	}
	if err := os.Rename(newPath, exe); err != nil {
		_ = os.Rename(exe+".old", exe)
		return false
	}
	_ = shellOpen(exe, "", filepath.Dir(exe))
	pPostMessageW.Call(ui.hwnd, WM_CLOSE, 0, 0)
	return true
}
