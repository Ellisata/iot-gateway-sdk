// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Edwin and iot-gateway contributors

package serialbus

import (
	"errors"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeRes 记录关闭次数的句柄替身。
type fakeRes struct {
	closed atomic.Int32
}

func (f *fakeRes) Close() error {
	f.closed.Add(1)
	return nil
}

// factory 交出可观察的句柄替身，并可切换成打开失败。
type factory struct {
	mu    sync.Mutex
	res   []*fakeRes
	fail  bool
	opens int
}

func (f *factory) open() (*fakeRes, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opens++
	if f.fail {
		return nil, errors.New("no such port")
	}
	r := &fakeRes{}
	f.res = append(f.res, r)
	return r, nil
}

func (f *factory) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.res)
}

func (f *factory) opened(i int) *fakeRes {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.res[i]
}

func testKey(port string) Key {
	return Key{Port: port, Baud: 2400, DataBits: 8, StopBits: 1, Parity: "E"}
}

// 同一端口、参数相同的两个会话必须共享同一个句柄与总线锁。
func TestSharedAcrossAcquires(t *testing.T) {
	bus := New[*fakeRes]()
	f := &factory{}

	a, err := bus.Acquire(testKey("COM7"), f.open)
	if err != nil {
		t.Fatalf("首个会话 Acquire 失败: %v", err)
	}
	b, err := bus.Acquire(testKey("COM7"), f.open)
	if err != nil {
		t.Fatalf("第二个会话 Acquire 失败: %v", err)
	}

	if a != b {
		t.Fatal("同一端口、相同参数的两个会话没有共享同一个句柄")
	}
	if got := f.count(); got != 1 {
		t.Errorf("端口被打开 %d 次，want 1（共享句柄）", got)
	}
	if got := a.Refs(); got != 2 {
		t.Errorf("Refs = %d, want 2", got)
	}

	// 最后一个引用关闭前，端口不得关闭。
	if err := a.Release(); err != nil {
		t.Fatalf("Release(1) 失败: %v", err)
	}
	if got := f.opened(0).closed.Load(); got != 0 {
		t.Errorf("关掉一个会话就把端口关了（closed=%d），另一个会话的句柄会失效", got)
	}
	if !b.Alive() {
		t.Error("关闭一个会话把共享句柄标记成断开了")
	}
	if err := b.Release(); err != nil {
		t.Fatalf("Release(2) 失败: %v", err)
	}
	if got := f.opened(0).closed.Load(); got != 1 {
		t.Errorf("最后一个引用释放后端口关闭次数 = %d, want 1", got)
	}
}

// 端口名与校验位的写法差异必须归一到同一个 Key。
func TestKeyNormalization(t *testing.T) {
	bus := New[*fakeRes]()
	f := &factory{}

	k1 := testKey("COM3")
	k2 := Key{Port: "  COM3  ", Baud: 2400, DataBits: 8, StopBits: 1, Parity: "e"}

	a, err := bus.Acquire(k1, f.open)
	if err != nil {
		t.Fatalf("首个会话 Acquire 失败: %v", err)
	}
	b, err := bus.Acquire(k2, f.open)
	if err != nil {
		t.Fatalf("等价写法的第二个会话被拒: %v", err)
	}
	if a != b || f.count() != 1 {
		t.Errorf("等价写法（空白/校验位大小写）没有共享句柄：same=%v opens=%d", a == b, f.count())
	}

	if runtime.GOOS == "windows" {
		c, err := bus.Acquire(Key{Port: "com3", Baud: 2400, DataBits: 8, StopBits: 1, Parity: "E"}, f.open)
		if err != nil {
			t.Fatalf("Windows 大小写不同的端口名被拒: %v", err)
		}
		if c != a || f.count() != 1 {
			t.Errorf("Windows 下端口名大小写未归一：opens=%d", f.count())
		}
	}
}

// 同端口不同参数必须是明确的配置错误，而不是悄悄再开一个句柄。
func TestParamsConflict(t *testing.T) {
	bus := New[*fakeRes]()
	f := &factory{}

	if _, err := bus.Acquire(testKey("COM9"), f.open); err != nil {
		t.Fatalf("首个会话 Acquire 失败: %v", err)
	}

	k := testKey("COM9")
	k.Baud = 9600
	_, err := bus.Acquire(k, f.open)
	if err == nil {
		t.Fatal("同端口不同波特率应当报配置错误，实际打开了第二个句柄")
	}
	for _, want := range []string{"COM9", "baud=9600", "baud=2400", "相同的串口参数"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("错误信息 %q 未包含 %q", err.Error(), want)
		}
	}
	if got := f.count(); got != 1 {
		t.Errorf("端口被打开 %d 次，want 1（参数冲突时不得再开句柄）", got)
	}
}

// 句柄判死后，下一个会话必须换一个新句柄，并先把旧 fd 关掉（termios 恢复
// 必须发生在新 open 之前，否则新句柄备份到的是一份将被覆盖的状态）。
func TestReopensDeadHandle(t *testing.T) {
	bus := New[*fakeRes]()
	f := &factory{}

	first, err := bus.Acquire(testKey("COM5"), f.open)
	if err != nil {
		t.Fatalf("首个会话 Acquire 失败: %v", err)
	}
	first.MarkBroken()
	if first.Alive() {
		t.Fatal("MarkBroken 之后 Alive 仍为 true")
	}

	second, err := bus.Acquire(testKey("COM5"), f.open)
	if err != nil {
		t.Fatalf("判死后重连失败: %v", err)
	}
	if first == second {
		t.Fatal("判死的句柄被复用了，应换新句柄")
	}
	if got := f.count(); got != 2 {
		t.Errorf("打开次数 = %d, want 2", got)
	}
	if got := f.opened(0).closed.Load(); got != 1 {
		t.Errorf("旧句柄关闭次数 = %d, want 1", got)
	}
	if !second.Alive() {
		t.Error("新句柄应当是可用状态")
	}
}

// 并发建连（不同会话同时 Connect）只能打开一个句柄。
func TestConcurrentAcquire(t *testing.T) {
	bus := New[*fakeRes]()
	f := &factory{}

	const n = 8
	results := make([]*Handle[*fakeRes], n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = bus.Acquire(testKey("COM11"), f.open)
		}(i)
	}
	close(start)
	wg.Wait()

	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("第 %d 个会话 Acquire 失败: %v", i, errs[i])
		}
		if results[i] != results[0] {
			t.Fatalf("第 %d 个会话拿到了不同的句柄", i)
		}
	}
	if got := f.count(); got != 1 {
		t.Errorf("并发建连打开了 %d 个句柄，want 1", got)
	}

	for i := 0; i < n; i++ {
		_ = results[i].Release()
	}
	if got := f.opened(0).closed.Load(); got != 1 {
		t.Errorf("全部会话关闭后端口关闭次数 = %d, want 1", got)
	}
}

// 打开失败时注册表不得留下半初始化条目，否则后续连接会命中一个死句柄。
func TestOpenFailureLeavesNoEntry(t *testing.T) {
	bus := New[*fakeRes]()
	f := &factory{fail: true}

	if _, err := bus.Acquire(testKey("COM21"), f.open); err == nil {
		t.Fatal("打开失败时应当返回错误")
	}

	bus.mu.Lock()
	left := len(bus.byKey) + len(bus.byPort)
	bus.mu.Unlock()
	if left != 0 {
		t.Errorf("打开失败后注册表残留 %d 个条目", left)
	}
}

// Release 必须幂等：驱动重连路径可能在 Close 之后再次调用。
func TestReleaseIdempotent(t *testing.T) {
	bus := New[*fakeRes]()
	f := &factory{}

	h, err := bus.Acquire(testKey("COM13"), f.open)
	if err != nil {
		t.Fatalf("Acquire 失败: %v", err)
	}
	if err := h.Release(); err != nil {
		t.Fatalf("Release 失败: %v", err)
	}
	if err := h.Release(); err != nil {
		t.Fatalf("重复 Release 失败: %v", err)
	}
	if err := h.Release(); err != nil {
		t.Fatalf("再次 Release 失败: %v", err)
	}
	if got := f.opened(0).closed.Load(); got != 1 {
		t.Errorf("重复 Release 导致端口关闭 %d 次, want 1", got)
	}
}

// 总线锁要真的把两个会话的交互串行化。
func TestLockSerializes(t *testing.T) {
	bus := New[*fakeRes]()
	f := &factory{}
	h, err := bus.Acquire(testKey("COM15"), f.open)
	if err != nil {
		t.Fatalf("Acquire 失败: %v", err)
	}
	defer h.Release()

	h.Lock()
	acquired := make(chan struct{})
	go func() {
		h.Lock()
		close(acquired)
		h.Unlock()
	}()

	select {
	case <-acquired:
		t.Fatal("总线被占用时另一个会话仍拿到了锁（帧会交错）")
	default:
	}
	h.Unlock()

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("总线释放后另一个会话仍拿不到锁")
	}
}

// 空端口名直接报错，不能登记成一个空键的句柄。
func TestEmptyPortRejected(t *testing.T) {
	bus := New[*fakeRes]()
	f := &factory{}
	if _, err := bus.Acquire(Key{}, f.open); err == nil {
		t.Fatal("空端口名应当报错")
	}
	if f.count() != 0 {
		t.Error("空端口名仍然打开了句柄")
	}
}
