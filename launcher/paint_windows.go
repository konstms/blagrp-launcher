package main

import (
	"syscall"
	"unsafe"
)

func fillRect(dc uintptr, l, t, r, b int32, color uintptr) {
	br, _, _ := pCreateSolidBrush.Call(color)
	rc := RECT{l, t, r, b}
	pFillRect.Call(dc, uintptr(unsafe.Pointer(&rc)), br)
	pDeleteObject.Call(br)
}

func roundRect(dc uintptr, l, t, r, b, rad int32, fill uintptr, border uintptr, hasBorder bool) {
	br, _, _ := pCreateSolidBrush.Call(fill)
	var pen uintptr
	if hasBorder {
		pen, _, _ = pCreatePen.Call(0, 1, border)
	} else {
		pen, _, _ = pGetStockObject.Call(NULL_PEN)
	}
	ob, _, _ := pSelectObject.Call(dc, br)
	op, _, _ := pSelectObject.Call(dc, pen)
	pRoundRect.Call(dc, uintptr(l), uintptr(t), uintptr(r), uintptr(b), uintptr(rad), uintptr(rad))
	pSelectObject.Call(dc, ob)
	pSelectObject.Call(dc, op)
	pDeleteObject.Call(br)
	if hasBorder {
		pDeleteObject.Call(pen)
	}
}

func ellipse(dc uintptr, l, t, r, b int32, color uintptr) {
	br, _, _ := pCreateSolidBrush.Call(color)
	pen, _, _ := pGetStockObject.Call(NULL_PEN)
	ob, _, _ := pSelectObject.Call(dc, br)
	op, _, _ := pSelectObject.Call(dc, pen)
	pEllipse.Call(dc, uintptr(l), uintptr(t), uintptr(r), uintptr(b))
	pSelectObject.Call(dc, ob)
	pSelectObject.Call(dc, op)
	pDeleteObject.Call(br)
}

func drawText(dc uintptr, s string, font uintptr, color uintptr, l, t, r, b int32, flags uintptr) {
	pSelectObject.Call(dc, font)
	pSetTextColor.Call(dc, color)
	rc := RECT{l, t, r, b}
	p, _ := syscall.UTF16PtrFromString(s)
	pDrawTextW.Call(dc, uintptr(unsafe.Pointer(p)), ^uintptr(0), uintptr(unsafe.Pointer(&rc)), flags|DT_NOPREFIX)
}

func lerp(a, b uint8, t float64) uint8 { return uint8(float64(a) + (float64(b)-float64(a))*t) }

// Цвета
var (
	cAmber     = rgb(245, 165, 36)
	cAmberHi   = rgb(255, 189, 74)
	cWhite     = rgb(255, 255, 255)
	cMuted     = rgb(150, 168, 196)
	cDim       = rgb(98, 116, 146)
	cPanel     = rgb(13, 21, 36)
	cPanelEdge = rgb(36, 51, 79)
	cField     = rgb(20, 31, 51)
	cFieldEdge = rgb(48, 66, 99)
	cTrack     = rgb(30, 42, 64)
	cGreen     = rgb(63, 207, 122)
	cRed       = rgb(235, 87, 87)
)

func (u *UI) paint(hwnd uintptr) {
	var ps PAINTSTRUCT
	hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	defer pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	W, H := u.W, u.H
	mem, _, _ := pCreateCompatibleDC.Call(hdc)
	bmp, _, _ := pCreateCompatibleBitmap.Call(hdc, uintptr(W), uintptr(H))
	old, _, _ := pSelectObject.Call(mem, bmp)
	pSetBkMode.Call(mem, TRANSPARENT)

	u.mu.Lock()
	phase, status, progress := u.phase, u.status, u.progress
	online, onlineOK := u.online, u.onlineOK
	label := u.buttonLabel()
	game := u.cfg.Game
	u.mu.Unlock()

	// Фон: вертикальный градиент ночного неба.
	for y := int32(0); y < H; y += 2 {
		t := float64(y) / float64(H)
		fillRect(mem, 0, y, W, y+2, rgb(lerp(10, 22, t), lerp(17, 34, t), lerp(31, 58, t)))
	}
	// Тёплое «зарево» снизу справа.
	for i := int32(0); i < 18; i++ {
		t := float64(i) / 18
		c := rgb(lerp(22, 60, t), lerp(34, 44, t), lerp(58, 52, t))
		ellipse(mem, W-u.sc(520)+u.sc(int(i)*12), H-u.sc(420)+u.sc(int(i)*12), W+u.sc(320)-u.sc(int(i)*12), H+u.sc(260)-u.sc(int(i)*12), c)
	}
	// Большой код региона «28» на заднем плане.
	drawText(mem, "28", u.fonts["big"], rgb(30, 46, 74), W-u.sc(470), u.sc(-30), W-u.sc(10), u.sc(330), DT_RIGHT|DT_SINGLELINE)

	// Заголовок.
	drawText(mem, "КРИМИНАЛЬНАЯ РОССИЯ  ·  АМУРСКАЯ ОБЛАСТЬ", u.fonts["kicker"], cAmber, u.sc(52), u.sc(46), W, u.sc(66), DT_SINGLELINE)
	drawText(mem, "БЛАГОВЕЩЕНСК", u.fonts["title"], cWhite, u.sc(48), u.sc(62), W, u.sc(140), DT_SINGLELINE)
	drawText(mem, "ROLE PLAY", u.fonts["sub"], cMuted, u.sc(52), u.sc(134), W, u.sc(164), DT_SINGLELINE)
	fillRect(mem, u.sc(52), u.sc(176), u.sc(52+56), u.sc(179), cAmber)
	drawText(mem, "Лаунчер сам поставит сборку сервера поверх твоей GTA San Andreas, будет держать её в актуальном состоянии и подключит тебя к игре одной кнопкой.",
		u.fonts["body"], cMuted, u.sc(52), u.sc(194), u.sc(52+430), u.sc(194+80), DT_WORDBREAK)

	// Онлайн.
	pl, pt, pr, pb := W-u.sc(48+210), u.sc(44), W-u.sc(48), u.sc(44+36)
	roundRect(mem, pl, pt, pr, pb, u.sc(36), cPanel, cPanelEdge, true)
	dot := cRed
	if onlineOK {
		dot = cGreen
	}
	ellipse(mem, pl+u.sc(16), pt+u.sc(14), pl+u.sc(24), pt+u.sc(22), dot)
	drawText(mem, online, u.fonts["pill"], cWhite, pl+u.sc(32), pt, pr-u.sc(12), pb, DT_SINGLELINE|DT_VCENTER)

	// Нижняя панель.
	l, t, r, b := u.panelRect()
	roundRect(mem, l, t, r, b, u.sc(22), cPanel, cPanelEdge, true)

	el, et, er, eb := u.editRect()
	drawText(mem, "НИК В ИГРЕ", u.fonts["small"], cDim, el, t+u.sc(20), er, t+u.sc(40), DT_SINGLELINE)
	roundRect(mem, el, et, er, eb, u.sc(12), cField, cFieldEdge, true)
	drawText(mem, "Формат: Имя_Фамилия", u.fonts["small"], cDim, el, eb+u.sc(6), er, eb+u.sc(24), DT_SINGLELINE)

	ql, qt, qr, qb := u.pathRect()
	drawText(mem, "ПАПКА ИГРЫ", u.fonts["small"], cDim, ql, t+u.sc(20), qr, t+u.sc(40), DT_SINGLELINE)
	gtxt := game
	if gtxt == "" {
		gtxt = "не выбрана"
	}
	drawText(mem, gtxt, u.fonts["path"], cWhite, ql, qt, qr, qb, DT_SINGLELINE|DT_VCENTER|DT_PATH_ELLIPSIS)
	ll, lt, lr, lb := u.linkRect()
	linkColor := cAmber
	if u.hover == 2 {
		linkColor = cAmberHi
	}
	drawText(mem, "Изменить папку →", u.fonts["small"], linkColor, ll, lt, lr, lb, DT_SINGLELINE)

	// Кнопка.
	bl, bt, br, bb := u.buttonRect()
	enabled := phase != phCheck && phase != phUpdating && phase != phSelfUpdate && phase != phLaunching
	fill, txt := cAmber, rgb(18, 18, 18)
	if !enabled {
		fill, txt = rgb(44, 58, 84), cMuted
	} else if u.hover == 1 {
		fill = cAmberHi
	}
	if phase == phUpdating && progress > 0 {
		roundRect(mem, bl, bt, br, bb, u.sc(16), rgb(44, 58, 84), 0, false)
	} else {
		roundRect(mem, bl, bt, br, bb, u.sc(16), fill, 0, false)
	}
	drawText(mem, label, u.fonts["button"], txt, bl, bt, br, bb, DT_SINGLELINE|DT_VCENTER|DT_CENTER)

	// Статус и прогресс.
	sl, st, sr := u.sc(56), b-u.sc(62), r-u.sc(28)
	drawText(mem, status, u.fonts["body"], cMuted, sl, st, sr, st+u.sc(22), DT_SINGLELINE|DT_END_ELLIPSIS)
	if progress >= 0 {
		pt2 := b - u.sc(30)
		roundRect(mem, sl, pt2, sr, pt2+u.sc(8), u.sc(8), cTrack, 0, false)
		p := progress
		if p > 1 {
			p = 1
		}
		w := int32(float64(sr-sl) * p)
		if w > u.sc(8) {
			roundRect(mem, sl, pt2, sl+w, pt2+u.sc(8), u.sc(8), cAmber, 0, false)
		}
	}
	drawText(mem, "v"+Version, u.fonts["small"], cDim, W-u.sc(120), H-u.sc(22), W-u.sc(32), H-u.sc(4), DT_SINGLELINE|DT_RIGHT)

	pBitBlt.Call(hdc, 0, 0, uintptr(W), uintptr(H), mem, 0, 0, SRCCOPY)
	pSelectObject.Call(mem, old)
	pDeleteObject.Call(bmp)
	pDeleteDC.Call(mem)
}
