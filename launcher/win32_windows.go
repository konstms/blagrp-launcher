package main

// Минимальные привязки к Win32 через стандартный пакет syscall (без внешних библиотек).

import (
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")

	pRegisterClassExW     = user32.NewProc("RegisterClassExW")
	pCreateWindowExW      = user32.NewProc("CreateWindowExW")
	pDefWindowProcW       = user32.NewProc("DefWindowProcW")
	pGetMessageW          = user32.NewProc("GetMessageW")
	pTranslateMessage     = user32.NewProc("TranslateMessage")
	pDispatchMessageW     = user32.NewProc("DispatchMessageW")
	pIsDialogMessageW     = user32.NewProc("IsDialogMessageW")
	pPostQuitMessage      = user32.NewProc("PostQuitMessage")
	pPostMessageW         = user32.NewProc("PostMessageW")
	pSendMessageW         = user32.NewProc("SendMessageW")
	pShowWindow           = user32.NewProc("ShowWindow")
	pUpdateWindow         = user32.NewProc("UpdateWindow")
	pBeginPaint           = user32.NewProc("BeginPaint")
	pEndPaint             = user32.NewProc("EndPaint")
	pInvalidateRect       = user32.NewProc("InvalidateRect")
	pGetClientRect        = user32.NewProc("GetClientRect")
	pLoadCursorW          = user32.NewProc("LoadCursorW")
	pSetCursor            = user32.NewProc("SetCursor")
	pMessageBoxW          = user32.NewProc("MessageBoxW")
	pGetWindowTextW       = user32.NewProc("GetWindowTextW")
	pGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	pSetWindowTextW       = user32.NewProc("SetWindowTextW")
	pFillRect             = user32.NewProc("FillRect")
	pDrawTextW            = user32.NewProc("DrawTextW")
	pGetSystemMetrics     = user32.NewProc("GetSystemMetrics")
	pAdjustWindowRectEx   = user32.NewProc("AdjustWindowRectEx")
	pSetProcessDPIAware   = user32.NewProc("SetProcessDPIAware")
	pGetDC                = user32.NewProc("GetDC")
	pReleaseDC            = user32.NewProc("ReleaseDC")
	pMoveWindow           = user32.NewProc("MoveWindow")
	pEnableWindow         = user32.NewProc("EnableWindow")
	pSetFocus             = user32.NewProc("SetFocus")
	pLoadImageW           = user32.NewProc("LoadImageW")
	pSetTimer             = user32.NewProc("SetTimer")

	pCreateSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	pCreatePen              = gdi32.NewProc("CreatePen")
	pDeleteObject           = gdi32.NewProc("DeleteObject")
	pSelectObject           = gdi32.NewProc("SelectObject")
	pGetStockObject         = gdi32.NewProc("GetStockObject")
	pSetBkMode              = gdi32.NewProc("SetBkMode")
	pSetBkColor             = gdi32.NewProc("SetBkColor")
	pSetTextColor           = gdi32.NewProc("SetTextColor")
	pCreateFontW            = gdi32.NewProc("CreateFontW")
	pRoundRect              = gdi32.NewProc("RoundRect")
	pEllipse                = gdi32.NewProc("Ellipse")
	pCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	pBitBlt                 = gdi32.NewProc("BitBlt")
	pDeleteDC               = gdi32.NewProc("DeleteDC")
	pGetDeviceCaps          = gdi32.NewProc("GetDeviceCaps")
	pMoveToEx               = gdi32.NewProc("MoveToEx")
	pLineTo                 = gdi32.NewProc("LineTo")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	pShellExecuteW         = shell32.NewProc("ShellExecuteW")
	pSHBrowseForFolderW    = shell32.NewProc("SHBrowseForFolderW")
	pSHGetPathFromIDListW  = shell32.NewProc("SHGetPathFromIDListW")
	pCoInitializeEx        = ole32.NewProc("CoInitializeEx")
	pCoTaskMemFree         = ole32.NewProc("CoTaskMemFree")
	pRegCreateKeyExW       = advapi32.NewProc("RegCreateKeyExW")
	pRegSetValueExW        = advapi32.NewProc("RegSetValueExW")
	pDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	WS_OVERLAPPED   = 0x00000000
	WS_CAPTION      = 0x00C00000
	WS_SYSMENU      = 0x00080000
	WS_MINIMIZEBOX  = 0x00020000
	WS_VISIBLE      = 0x10000000
	WS_CHILD        = 0x40000000
	WS_CLIPCHILDREN = 0x02000000
	WS_TABSTOP      = 0x00010000
	ES_AUTOHSCROLL  = 0x0080
	ES_CENTER       = 0x0001

	WM_DESTROY       = 0x0002
	WM_PAINT         = 0x000F
	WM_CLOSE         = 0x0010
	WM_ERASEBKGND    = 0x0014
	WM_SETCURSOR     = 0x0020
	WM_SETFONT       = 0x0030
	WM_COMMAND       = 0x0111
	WM_TIMER         = 0x0113
	WM_CTLCOLOREDIT  = 0x0133
	WM_MOUSEMOVE     = 0x0200
	WM_LBUTTONDOWN   = 0x0201
	WM_LBUTTONUP     = 0x0202
	WM_APP           = 0x8000
	EM_LIMITTEXT     = 0x00C5
	EM_SETMARGINS    = 0x00D3
	EN_CHANGE        = 0x0300

	SW_SHOW         = 5
	SW_SHOWNORMAL   = 1
	IDC_ARROW       = 32512
	IDC_HAND        = 32649
	TRANSPARENT     = 1
	NULL_PEN        = 8
	DT_CENTER       = 0x1
	DT_VCENTER      = 0x4
	DT_SINGLELINE   = 0x20
	DT_WORDBREAK    = 0x10
	DT_END_ELLIPSIS = 0x8000
	DT_PATH_ELLIPSIS = 0x4000
	DT_NOPREFIX     = 0x800
	DT_RIGHT        = 0x2
	SRCCOPY         = 0x00CC0020

	MB_OK          = 0x0
	MB_ICONERROR   = 0x10
	MB_ICONWARNING = 0x30
	MB_ICONINFO    = 0x40
	MB_YESNO       = 0x4
	IDYES          = 6
)

type RECT struct{ Left, Top, Right, Bottom int32 }
type POINT struct{ X, Y int32 }

type WNDCLASSEXW struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
	Private uint32
}

type PAINTSTRUCT struct {
	Hdc         uintptr
	Erase       int32
	RcPaint     RECT
	Restore     int32
	IncUpdate   int32
	RgbReserved [32]byte
}

type BROWSEINFOW struct {
	Owner       uintptr
	Root        uintptr
	DisplayName *uint16
	Title       *uint16
	Flags       uint32
	Fn          uintptr
	LParam      uintptr
	Image       int32
}

func u16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func rgb(r, g, b uint8) uintptr { return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16 }

func msgBox(hwnd uintptr, text, title string, flags uintptr) int {
	r, _, _ := pMessageBoxW.Call(hwnd, uintptr(unsafe.Pointer(u16(text))), uintptr(unsafe.Pointer(u16(title))), flags)
	return int(r)
}

func getText(hwnd uintptr) string {
	n, _, _ := pGetWindowTextLengthW.Call(hwnd)
	buf := make([]uint16, n+1)
	pGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), n+1)
	return syscall.UTF16ToString(buf)
}

func setText(hwnd uintptr, s string) {
	pSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(u16(s))))
}

func shellOpen(target, args, dir string) error {
	var a, d uintptr
	if args != "" {
		a = uintptr(unsafe.Pointer(u16(args)))
	}
	if dir != "" {
		d = uintptr(unsafe.Pointer(u16(dir)))
	}
	r, _, _ := pShellExecuteW.Call(0, uintptr(unsafe.Pointer(u16("open"))), uintptr(unsafe.Pointer(u16(target))), a, d, SW_SHOWNORMAL)
	if r <= 32 {
		return syscall.Errno(r)
	}
	return nil
}

func browseFolder(owner uintptr, title string) string {
	pCoInitializeEx.Call(0, 2) // COINIT_APARTMENTTHREADED
	disp := make([]uint16, 260)
	bi := BROWSEINFOW{
		Owner:       owner,
		DisplayName: &disp[0],
		Title:       u16(title),
		Flags:       0x0001 | 0x0040 | 0x0200, // RETURNONLYFSDIRS | NEWDIALOGSTYLE | NONEWFOLDERBUTTON
	}
	pidl, _, _ := pSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return ""
	}
	defer pCoTaskMemFree.Call(pidl)
	path := make([]uint16, 1024)
	r, _, _ := pSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(path)
}

// Запись строкового значения в HKEY_CURRENT_USER.
func regSetString(subkey, name, value string) error {
	const HKCU = 0x80000001
	const KEY_SET_VALUE = 0x0002
	var key syscall.Handle
	r, _, _ := pRegCreateKeyExW.Call(HKCU, uintptr(unsafe.Pointer(u16(subkey))), 0, 0, 0, KEY_SET_VALUE, 0, uintptr(unsafe.Pointer(&key)), 0)
	if r != 0 {
		return syscall.Errno(r)
	}
	defer syscall.RegCloseKey(key)
	data, _ := syscall.UTF16FromString(value)
	r, _, _ = pRegSetValueExW.Call(uintptr(key), uintptr(unsafe.Pointer(u16(name))), 0, syscall.REG_SZ, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)*2))
	if r != 0 {
		return syscall.Errno(r)
	}
	return nil
}

func regGetString(subkey, name string) string {
	var key syscall.Handle
	if err := syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, u16(subkey), 0, syscall.KEY_READ, &key); err != nil {
		return ""
	}
	defer syscall.RegCloseKey(key)
	var typ, n uint32
	if err := syscall.RegQueryValueEx(key, u16(name), nil, &typ, nil, &n); err != nil || n == 0 {
		return ""
	}
	buf := make([]uint16, n/2+1)
	if err := syscall.RegQueryValueEx(key, u16(name), nil, &typ, (*byte)(unsafe.Pointer(&buf[0])), &n); err != nil {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
