// Package winui 提供 Windows 窗口的强制唤醒能力。
// 用于从托盘恢复长期隐藏/最小化的主窗口：
// 单纯 SW_SHOW 无法恢复最小化窗口，也不会抢占前台焦点，
// 因此这里组合 ShowWindowAsync(SW_RESTORE) + SetForegroundWindow。
//
// 全部使用 *Async / PostMessage 风格调用，可从任意线程安全调用，
// 不会阻塞调用方（托盘消息循环线程）。
package winui

import (
	"syscall"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procFindWindowW         = user32.NewProc("FindWindowW")
	procShowWindowAsync     = user32.NewProc("ShowWindowAsync")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procBringWindowToTop    = user32.NewProc("BringWindowToTop")
	procIsIconic            = user32.NewProc("IsIconic")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
	procPostMessageW        = user32.NewProc("PostMessageW")
)

const (
	swRestore = 9 // SW_RESTORE：还原最小化/最大化窗口到正常大小
	wmNull    = 0x0000
)

// FindWindowByTitle 按窗口标题精确查找顶层窗口句柄，找不到返回 0。
func FindWindowByTitle(title string) uintptr {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
	return hwnd
}

// WakeWindow 强制唤醒窗口：恢复、置顶并获取前台焦点。
// 所有调用都是异步/非阻塞的，适合从托盘回调线程使用。
func WakeWindow(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}

	// 先向窗口线程投递一个空消息，唤醒可能处于等待状态的消息循环
	procPostMessageW.Call(hwnd, wmNull, 0, 0)

	// 异步恢复窗口（不等待目标线程，避免跨线程阻塞）
	procShowWindowAsync.Call(hwnd, swRestore)

	// 置顶并抢占前台焦点
	procBringWindowToTop.Call(hwnd)
	procSetForegroundWindow.Call(hwnd)
	return true
}

// IsMinimized 判断窗口是否处于最小化状态。
func IsMinimized(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	ret, _, _ := procIsIconic.Call(hwnd)
	return ret != 0
}

// IsVisible 判断窗口当前是否可见。
func IsVisible(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	ret, _, _ := procIsWindowVisible.Call(hwnd)
	return ret != 0
}
