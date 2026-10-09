# SGREEN 项目测试计划

基于对 screen C 项目测试框架的分析，为 sgreen Go 项目制定以下测试计划：

## 1. 项目架构分析

**主要模块结构**：
- **cmd/sgreen** - 主程序入口（CLI界面）
- **internal/session** - 会话管理核心逻辑
- **internal/pty** - PTY（伪终端）处理
- **internal/ui** - 用户界面和ANSI处理
- **internal/config** - 配置解析

## 2. 测试框架设计

测试直接放在被测包旁边（Go 惯例），行为测试通过真实二进制验证 CLI：

```
internal/session/*_test.go   # 会话/窗口管理（HOME 隔离，真实进程）
internal/pty/*_test.go       # PTY 启动、DataConn 路由、daemon remote 端点
internal/ui/*_test.go        # ANSI、滚动缓冲、状态栏、消息、终端能力、编码
internal/config/*_test.go    # .screenrc 解析与查找
internal/daemon/*_test.go    # （预留）daemon 协议
cmd/sgreen/*_test.go         # CLI 决策逻辑（selectReattachSession 等）
test/behavior/               # 行为测试：构建真实二进制 + GNU screen 双跑对比
  cli_test.go                # 退出码/消息断言（与 GNU 4.09 实测对齐）
  compare_with_gnu_screen.sh # 49 个用例双跑对比，差异即失败
```

曾经存在的外部 `tests/` 目录（unit/mockhelpers/utils/fixtures）已移除：
unit 测试迁入对应 internal 包（覆盖率统计从此计入），
mockhelpers/utils/fixtures 从未被引用，属于死代码。
计划的 integration/performance/concurrency/platforms 目录从未落地；
需要时按 Go 惯例直接在目标包内加 `*_integration_test.go` / `*_bench_test.go`，
不再维护独立目录骨架。

## 3. 测试计划详情

### 3.1 单元测试（Unit Tests）

**session_test.go**
```go
// 参考 test-winmsgcond.c 的风格
func TestSessionCreate(t *testing.T) {
    // 测试会话创建
    s := session.NewSession("test", "/bin/bash", []string{})
    assert.NotNil(t, s)
    assert.Equal(t, "test", s.ID)
}

func TestSessionLifecycle(t *testing.T) {
    // 测试会话完整的生命周期
    s := session.NewSession("test", "/bin/bash", []string{})
    s.Start()
    defer s.Kill()

    // 测试会话状态
    assert.True(t, s.IsRunning())
    assert.NotNil(t, s.PTYProcess)
}

func TestWindowManagement(t *testing.T) {
    // 测试窗口创建、切换、删除
    s := session.NewSession("test", "/bin/bash", []string{})

    // 创建新窗口
    win, err := s.NewWindow("/bin/sh", []string{})
    assert.NoError(t, err)
    assert.Len(t, s.Windows, 2) // 默认窗口 + 新窗口

    // 切换窗口
    err = s.SwitchWindow(1)
    assert.NoError(t, err)
    assert.Equal(t, 1, s.CurrentWindow)
}
```

**pty_test.go**
```go
func TestPTYCreation(t *testing.T) {
    // 测试PTY进程创建
    ptyProc, err := pty.StartCmd(exec.Command("/bin/bash"))
    assert.NoError(t, err)
    defer ptyProc.Close()

    // 测试PTY基本功能
    assert.NotNil(t, ptyProc.PtsPath)
    assert.NotNil(t, ptyProc.Cmd)
}

func TestPTYReconnect(t *testing.T) {
    // 测试PTY重连功能
    ptyProc, err := pty.StartCmd(exec.Command("/bin/sh"))
    assert.NoError(t, err)

    // 模拟进程退出
    ptyProc.Cmd.Process.Kill()

    // 测试重连逻辑
    newProc, err := pty.ReconnectPTY(ptyProc.PtsPath)
    // ... 测试重连结果
}
```

### 3.2 集成测试（Integration Tests）

**session_lifecycle_test.go**
```go
func TestSessionFullLifecycle(t *testing.T) {
    // 创建临时目录
    tempDir := t.TempDir()

    // 创建会话
    s := session.NewSession("integration_test", "/bin/bash", []string{})
    s.SaveDir = tempDir

    // 保存会话
    err := s.Save()
    assert.NoError(t, err)

    // 从磁盘加载会话
    loadedS, err := session.LoadSession("integration_test", tempDir)
    assert.NoError(t, err)
    assert.Equal(t, s.ID, loadedS.ID)

    // 清理
    loadedS.Kill()
    os.RemoveAll(tempDir)
}
```

### 3.3 UI 模块详细测试计划

**ui_ansi_test.go** - ANSI 转义序列处理
```go
func TestANSIParser(t *testing.T) {
    // 测试各种 ANSI 转义序列解析
    tests := []struct {
        input    string
        expected []ANSISequence
    }{
        {"\x1b[31mHello", []ANSISequence{{Type: "color", Value: "red"}}},
        {"\x1b[2J", []ANSISequence{{Type: "clear"}}},
        {"\x1b[H", []ANSISequence{{Type: "home"}}},
    }
    // 实现测试逻辑
}

func TestANSIRendering(t *testing.T) {
    // 测试 ANSI 序列的渲染输出
    // 验证颜色、光标位置、清屏等操作
}

func TestANSICorruptedInput(t *testing.T) {
    // 测试损坏的 ANSI 序列处理
    // 确保不会崩溃，能优雅降级
}
```

**ui_scrollback_test.go** - 滚动缓冲测试
```go
func TestScrollbackBuffer(t *testing.T) {
    // 测试滚动缓冲的基本功能
    buffer := NewScrollbackBuffer(1000)
    
    // 添加数据
    buffer.AddLine("Line 1")
    buffer.AddLine("Line 2")
    
    // 验证数据正确性
    assert.Equal(t, 2, buffer.LineCount())
    assert.Equal(t, "Line 1", buffer.GetLine(0))
}

func TestScrollbackBufferOverflow(t *testing.T) {
    // 测试缓冲区溢出处理
    buffer := NewScrollbackBuffer(10)
    for i := 0; i < 20; i++ {
        buffer.AddLine(fmt.Sprintf("Line %d", i))
    }
    // 验证缓冲区保持固定大小
    assert.Equal(t, 10, buffer.LineCount())
}

func TestScrollbackSearch(t *testing.T) {
    // 测试滚动缓冲的搜索功能
    buffer := NewScrollbackBuffer(100)
    buffer.AddLine("error: something went wrong")
    buffer.AddLine("info: processing complete")
    
    results := buffer.Search("error")
    assert.Len(t, results, 1)
    assert.Equal(t, 0, results[0])
}
```

**ui_copymode_test.go** - 复制模式测试
```go
func TestCopyModeEnterExit(t *testing.T) {
    // 测试进入和退出复制模式
    ui := NewTestUI()
    ui.EnterCopyMode()
    assert.True(t, ui.InCopyMode())
    
    ui.ExitCopyMode()
    assert.False(t, ui.InCopyMode())
}

func TestCopyModeSelection(t *testing.T) {
    // 测试文本选择功能
    ui := NewTestUI()
    ui.EnterCopyMode()
    ui.SetSelectionStart(0, 0)
    ui.SetSelectionEnd(0, 10)
    
    selected := ui.GetSelectedText()
    assert.Contains(t, selected, "expected text")
}

func TestCopyModeNavigation(t *testing.T) {
    // 测试复制模式中的导航
    // 上下滚动、翻页等
}
```

**ui_terminal_caps_test.go** - 终端能力测试
```go
func TestTerminalCapabilitiesDetection(t *testing.T) {
    // 测试终端能力检测
    caps := DetectTerminalCapabilities()
    
    assert.NotEmpty(t, caps.Term)
    assert.True(t, caps.Colors > 0)
}

func TestTerminalCapsFallback(t *testing.T) {
    // 测试终端能力检测失败时的降级处理
    // 模拟无法获取 terminfo 数据的情况
}

func TestTerminalCapsCompatibility(t *testing.T) {
    // 测试不同终端的兼容性
    terminals := []string{"xterm-256color", "vt100", "screen"}
    for _, term := range terminals {
        caps := DetectTerminalCapabilities(term)
        assert.NotNil(t, caps)
    }
}
```

**ui_monitoring_test.go** - 监控功能测试
```go
func TestSessionMonitoring(t *testing.T) {
    // 测试会话监控功能
    ui := NewTestUI()
    ui.EnableMonitoring()
    
    // 模拟活动检测
    ui.RecordActivity("window1", time.Now())
    
    activity := ui.GetActivityLog()
    assert.Len(t, activity, 1)
}

func TestSilenceDetection(t *testing.T) {
    // 测试静默检测
    ui := NewTestUI()
    ui.EnableMonitoring()
    
    // 模拟长时间无活动
    // 验证静默通知
}

func TestMonitoringPerformance(t *testing.T) {
    // 测试监控功能的性能影响
    // 确保监控不会显著降低性能
}
```

**ui_status_test.go** - 状态栏测试
```go
func TestStatusBarDisplay(t *testing.T) {
    // 测试状态栏显示
    ui := NewTestUI()
    ui.SetStatusMessage("Session active")
    
    status := ui.GetStatusBar()
    assert.Contains(t, status, "Session active")
}

func TestStatusBarUpdate(t *testing.T) {
    // 测试状态栏实时更新
    ui := NewTestUI()
    ui.SetStatusMessage("Loading...")
    ui.SetStatusMessage("Done")
    
    assert.Equal(t, "Done", ui.GetStatusMessage())
}

func TestStatusBarWindowInfo(t *testing.T) {
    // 测试状态栏中的窗口信息显示
    ui := NewTestUI()
    ui.AddWindow("window1")
    ui.SetCurrentWindow("window1")
    
    status := ui.GetStatusBar()
    assert.Contains(t, status, "window1")
}
```

### 3.4 配置模块测试

**config_test.go** - 配置解析测试
```go
func TestConfigParsing(t *testing.T) {
    // 测试配置文件解析
    config := ParseConfig("testdata/valid_config.yaml")
    
    assert.Equal(t, "screen", config.DefaultShell)
    assert.True(t, config.EnableLogging)
}

func TestConfigValidation(t *testing.T) {
    // 测试配置验证
    tests := []struct {
        config  Config
        valid   bool
    }{
        {Config{Shell: "/bin/bash"}, true},
        {Config{Shell: ""}, false},
        {Config{WindowCount: -1}, false},
    }
    // 实现验证逻辑
}

func TestConfigDefaults(t *testing.T) {
    // 测试默认配置值
    config := NewDefaultConfig()
    
    assert.NotEmpty(t, config.Shell)
    assert.Greater(t, config.WindowCount, 0)
}

func TestConfigOverride(t *testing.T) {
    // 测试命令行参数覆盖配置文件
    config := ParseConfig("testdata/config.yaml")
    config.ApplyOverrides(map[string]string{
        "shell": "/bin/zsh",
    })
    
    assert.Equal(t, "/bin/zsh", config.Shell)
}
```

### 3.5 现有测试整合

**已存在的测试需要整合到新框架中**：

- [cmd/sgreen/main_test.go](cmd/sgreen/main_test.go) - CLI 会话选择和参数解析测试
- [internal/session/window_test.go](internal/session/window_test.go) - 窗口编号转换和编码检测测试
- [internal/ui/encoding_test.go](internal/ui/encoding_test.go) - 字符编码转换测试

这些测试应该保持不变，但需要在新的测试文档中明确记录其覆盖范围和作用。

### 3.6 Mock 策略（实际采用）

原计划编写 mockhelpers 包（PTY/会话/UI/syscall/fs/time 模拟器），该包从未被
任何测试引用，已删除。实际采用的隔离手段：

- **依赖注入**：`pty.RemoteControl`（Resize/Alive/Quit 函数字段）让 daemon
  控制路径可以在没有真实 daemon 的情况下测试（见 internal/pty/remote_test.go）。
- **进程注入**：cmd/sgreen 的 `selectReattachSession` 等决策函数接受
  `loadByName`/`isAttached` 函数参数，测试直接传入 stub。
- **内存管道**：`net.Pipe()` 充当 daemon socket 的替身。
- **HOME 隔离**：internal/session 的 TestMain 将 HOME 指向临时目录，
  会话目录惰性初始化（`dir()`），测试不触碰真实 ~/.sgreen。
- **真实 PTY**：pty/session 的行为测试直接启动短命令（/bin/sleep、/bin/echo），
  比模拟 PTY 更接近真实故障模式。

### 3.7 平台特定测试

**platforms/unix_test.go** - Unix 平台测试
```go
// +build darwin linux freebsd openbsd netbsd

package platforms

import (
    "os"
    "syscall"
    "testing"
)

func TestUnixPTYCreation(t *testing.T) {
    // 测试 Unix 平台特有的 PTY 创建
    // 验证权限、设备节点等
}

func TestUnixSignalHandling(t *testing.T) {
    // 测试 Unix 特有的信号处理
    // SIGHUP, SIGWINCH, SIGUSR1 等
}

func TestUnixSocketPath(t *testing.T) {
    // 测试 Unix socket 路径
    // /tmp/screens/S-username/ 格式
}

func TestUnixPermissions(t *testing.T) {
    // 测试文件和目录权限
    // 会话文件应该只有用户可读写
}
```

**platforms/windows_test.go** - Windows 平台测试
```go
// +build windows

package platforms

import (
    "testing"
)

func TestWindowsPTYCreation(t *testing.T) {
    // 测试 Windows 平台特有的 PTY 创建
    // 使用 ConPTY API
}

func TestWindowsSignalHandling(t *testing.T) {
    // 测试 Windows 特有的信号处理
    // Ctrl+C, Ctrl+Break 等
}

func TestWindowsSocketPath(t *testing.T) {
    // 测试 Windows socket 路径
    // \\.\pipe\ 格式
}

func TestWindowsPermissions(t *testing.T) {
    // 测试 Windows 文件和目录权限
    // ACL 权限控制
}
```

### 3.8 跨平台兼容性测试

**integration/cross_platform_test.go**
```go
func TestCrossPlatformSessionFormat(t *testing.T) {
    // 测试会话格式的跨平台兼容性
    // 确保在 Unix 和 Windows 上创建的会话可以互相加载
    
    sessionData := map[string]interface{}{
        "id":    "cross_platform_test",
        "shell": "/bin/bash",
        "windows": []map[string]interface{}{
            {"id": 0, "cmd": "/bin/sh"},
        },
    }
    
    // 序列化
    data, err := json.Marshal(sessionData)
    assert.NoError(t, err)
    
    // 反序列化
    var loaded map[string]interface{}
    err = json.Unmarshal(data, &loaded)
    assert.NoError(t, err)
    
    // 验证数据一致性
    assert.Equal(t, "cross_platform_test", loaded["id"])
}

func TestPathNormalization(t *testing.T) {
    // 测试路径的跨平台规范化
    paths := []struct {
        input    string
        expected string
    }{
        {"/tmp/test", normalizePath("/tmp/test")},
        {"C:\\temp\\test", normalizePath("C:\\temp\\test")},
    }
    
    for _, tc := range paths {
        normalized := normalizePath(tc.input)
        assert.NotEmpty(t, normalized)
    }
}
```

### 3.9 测试工具函数

**utils/assertions.go**
```go
func assertPTYRunning(t *testing.T, ptyProc *pty.PTYProcess) {
    t.Helper()
    if ptyProc == nil || ptyProc.Cmd == nil || ptyProc.Cmd.Process == nil {
        t.Errorf("Expected PTY process to be running")
    }
}

func assertSessionWindowCount(t *testing.T, session *session.Session, expected int) {
    t.Helper()
    if len(session.Windows) != expected {
        t.Errorf("Expected %d windows, got %d", expected, len(session.Windows))
    }
}
```

## 4. 测试执行策略

### 4.1 测试分层
1. **单元测试**：快速执行，覆盖核心逻辑
2. **集成测试**：较慢执行，测试模块间交互
3. **性能测试**：基准测试和内存泄漏检测
4. **并发测试**：竞态条件和并发安全验证
5. **行为测试**：最慢，与GNU screen对比验证

### 4.2 测试命令

```bash
# 运行所有测试（= make test）
go test -v ./...

# 单元测试（按包）
go test ./internal/session/...
go test ./internal/pty/...
go test ./internal/ui/...
go test ./internal/config/...
go test ./cmd/sgreen/...

# 行为测试（构建真实二进制）
go test -v ./test/behavior/...

# GNU screen 双跑对比（差异即失败）
./test/behavior/compare_with_gnu_screen.sh

# 覆盖率（-coverpkg 让外部测试包的贡献计入统计）
go test -covermode=atomic -coverprofile=coverage.out -coverpkg=./... ./...
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html

# 竞态检测
go test -race ./...
```


### 4.3 测试覆盖率目标

#### 4.3.1 覆盖率指标

> 现状（2026-08，合并口径 `-coverpkg=./...`）：总覆盖率约 18%，CI 门槛 15%。
> 目标分阶段推进：15%（现状）→ 40% → 70%（终态），每次上调前先补齐对应
> 模块测试，避免门槛长期红灯失效。零覆盖重灾区：internal/ui/attach.go、
> copymode.go、monitoring.go、cmd/sgreen/main.go 的 handler 函数。

- **总体覆盖率（终态目标）**：≥ 70%
- **核心模块覆盖率**：
  - `internal/session/`：≥ 85%
  - `internal/pty/`：≥ 80%
  - `internal/ui/`：≥ 75%
  - `internal/config/`：≥ 80%
  - `cmd/sgreen/`：≥ 70%

#### 4.3.2 覆盖率类型
- **语句覆盖率**：主要指标，如上所述
- **分支覆盖率**：核心模块 ≥ 70%
- **函数覆盖率**：所有模块 ≥ 80%

#### 4.3.3 覆盖率监控
```bash
# 检查覆盖率是否达标
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total

# CI 中设置覆盖率阈值
if [ $(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//') -lt 70 ]; then
    echo "Coverage below threshold"
    exit 1
fi
```

### 4.4 持续集成
- 每次提交都运行单元测试
- Pull Request 需要运行所有测试
- 定期运行与GNU screen的对比测试
- 每周运行完整的性能和并发测试
- 自动化覆盖率报告生成

### 4.5 CI/CD 配置

真实配置见仓库（本文不再内嵌 YAML 副本，避免与实际漂移）：

- [.github/workflows/test.yml](.github/workflows/test.yml)
  - `test` job：三平台矩阵，`go test -race -coverpkg=./... ./...`
    一次跑完测试+竞态+覆盖率，门槛 15%（当前 ~18%，目标分阶段提到 40%+）；
    Windows 跳过 -race（runner 工具链不稳）。
  - `weekly-full-test` job：每周一 schedule 触发，race 全量 + 安装
    GNU screen 跑双跑对比，报告有差异即失败。
- [.github/workflows/ci.yml](.github/workflows/ci.yml)
  - fmt（gofmt -l 只读检查）/ lint / 9 目标交叉编译构建 / gnu-screen-parity。


### 4.6 测试报告

#### 4.6.1 本地测试报告生成

```bash
# 生成详细的覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 生成覆盖率报告（含外部测试包贡献）
go test -covermode=atomic -coverprofile=coverage.out -coverpkg=./... ./...
go tool cover -html=coverage.out -o coverage.html

# 生成竞态条件检测报告
go test -race ./... 2> race_report.txt

# 生成内存分析报告（基准落地后按包运行）
go test -memprofile=mem.prof ./internal/session/...
go tool pprof -text mem.prof > memory_report.txt
```

#### 4.6.2 测试结果分析

```bash
# 查看测试覆盖率详情
go tool cover -func=coverage.out | grep -E "(total|session|pty|ui)"

# 查看未覆盖的代码
go tool cover -func=coverage.out | awk '$3 != "100.0%"'

# 查看性能基准对比
go install github.com/codahale/benchstat@latest
benchstat old_benchmark.txt new_benchmark.txt
```

## 5. 测试重点

### 5.1 核心功能优先级
1. **会话管理** - 创建、保存、加载、销毁
2. **窗口管理** - 创建、切换、关闭、标题设置
3. **PTY处理** - 启动、重连、信号处理
4. **CLI接口** - 所有命令行选项的正确性
5. **UI组件** - ANSI解析、滚动缓冲、复制模式

### 5.2 边界条件测试
- 内存不足时的表现
- 无效输入的处理
- 并发访问的安全性
- 系统调用失败的处理
- 异常 ANSI 序列的处理
- 大数据量的缓冲区操作

### 5.3 性能测试
- 大量窗口时的性能
- 长时间运行的稳定性
- 重连操作的效率
- 内存使用和泄漏检测
- 并发操作的性能表现

### 5.4 性能和并发测试（待落地）

性能与并发测试尚未编写。落地时的形态（遵循 Go 惯例，不建独立目录）：

- 基准：在目标包内添加 `BenchmarkXxx`（如 internal/session 的
  Save/Load 往返、internal/ui 的滚动缓冲写入），`go test -bench` 运行。
- 竞态：`go test -race ./...`（CI 常开，Windows 除外）。
- 压测/内存：`go test -tags=stress`、`-memprofile` 配合 pprof。


## 6. 实施步骤

### 阶段1：搭建测试框架（2周）
- 创建目录结构
- 实现基础模拟器
- 编写工具函数

### 阶段2：单元测试覆盖（4周）
- session模块测试
- window模块测试
- pty模块测试

### 阶段3：集成测试（2周）
- 会话生命周期测试
- 窗口切换测试
- 错误恢复测试

### 阶段4：优化和维护（持续）
- 提高测试覆盖率
- 优化测试性能
- 添加新功能测试

## 7.（已移除）测试目录创建脚本

原此节包含一段 mkdir + cat 的 shell 脚本，用于生成 tests/ 目录骨架和示例
测试。该骨架与实际结构不符（测试已位于各 internal 包内），脚本删除。


## 8. 测试依赖管理

确保在 `go.mod` 中添加测试依赖：

```go
require (
    github.com/stretchr/testify v1.8.4  // 单元测试断言库
    github.com/stretchr/testify v1.8.4  // Mock 支持
)
```

## 9. 测试文档和维护指南

### 9.1 测试编写规范

#### 9.1.1 命名约定
- **测试文件**：`xxx_test.go`，与被测试文件在同一目录
- **测试函数**：`TestFunctionName(t *testing.T)` 或 `TestFunctionName_Subcase(t *testing.T)`
- **基准测试**：`BenchmarkFunctionName(b *testing.B)`
- **示例测试**：`ExampleFunctionName()`

#### 9.1.2 测试结构
```go
func TestSessionCreate(t *testing.T) {
    // 准备测试数据
    tests := []struct {
        name    string
        input   string
        want    *Session
        wantErr bool
    }{
        {
            name:    "valid session",
            input:   "test_session",
            want:    &Session{ID: "test_session"},
            wantErr: false,
        },
        {
            name:    "empty name",
            input:   "",
            want:    nil,
            wantErr: true,
        },
    }
    
    // 执行测试
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := session.NewSession(tt.input, "/bin/bash", []string{})
            if (err != nil) != tt.wantErr {
                t.Errorf("NewSession() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got == nil && tt.want != nil {
                t.Errorf("NewSession() = nil, want %v", tt.want)
            }
        })
    }
}
```

#### 9.1.3 测试辅助函数
```go
// 创建测试会话的辅助函数
func createTestSession(t *testing.T) *session.Session {
    t.Helper()
    s, err := session.NewSession("test", "/bin/bash", []string{})
    if err != nil {
        t.Fatalf("Failed to create test session: %v", err)
    }
    t.Cleanup(func() {
        s.Kill()
    })
    return s
}

// 验证会话状态的辅助函数
func assertSessionRunning(t *testing.T, s *session.Session) {
    t.Helper()
    if !s.IsRunning() {
        t.Errorf("Expected session to be running")
    }
}
```

### 9.2 Mock 使用指南

#### 9.2.1 何时使用 Mock
- 测试代码依赖外部服务（网络、数据库）
- 测试需要模拟错误条件
- 测试需要隔离特定组件
- 测试执行速度很重要

#### 9.2.2 Mock 使用示例
```go
func TestSessionAttachWithMock(t *testing.T) {
    // 使用 Mock PTY
    mockPTY := mockhelpers.NewMockPTY()
    mockPTY.SetShouldFail(false)
    
    s := createTestSession(t)
    s.PTY = mockPTY
    
    // 测试附着逻辑
    err := s.Attach()
    assert.NoError(t, err)
    assert.True(t, s.IsAttached)
    
    // 验证 Mock 调用
    assert.Equal(t, 1, mockPTY.StartCallCount)
}
```

### 9.3 测试数据管理

#### 9.3.1 测试fixtures
```go
// tests/fixtures/testdata.go
package fixtures

import "github.com/inoki/sgreen/internal/session"

func ValidSession() *session.Session {
    return &session.Session{
        ID:    "valid_test_session",
        Pid:   12345,
        Shell: "/bin/bash",
    }
}

func InvalidSession() *session.Session {
    return &session.Session{
        ID:    "",
        Pid:   0,
        Shell: "",
    }
}
```

#### 9.3.2 测试配置文件
```yaml
# tests/fixtures/test_configs/valid_config.yaml
session:
  name: "test_session"
  shell: "/bin/bash"
  
windows:
  - id: 0
    cmd: "/bin/sh"
    
logging:
  enabled: true
  level: "debug"
```

### 9.4 性能测试指南

#### 9.4.1 编写基准测试
```go
func BenchmarkWindowSwitch(b *testing.B) {
    s := session.NewSession("bench", "/bin/bash", []string{})
    defer s.Kill()
    
    // 预热
    for i := 0; i < 10; i++ {
        s.NewWindow("/bin/sh", []string{})
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        s.SwitchWindow(i % 10)
    }
}
```

#### 9.4.2 内存分析
```go
func BenchmarkSessionMemoryUsage(b *testing.B) {
    var m1, m2 runtime.MemStats
    
    runtime.GC()
    runtime.ReadMemStats(&m1)
    
    for i := 0; i < b.N; i++ {
        s := session.NewSession(fmt.Sprintf("bench%d", i), "/bin/bash", []string{})
        s.Kill()
    }
    
    runtime.GC()
    runtime.ReadMemStats(&m2)
    
    b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "B/op")
}
```

### 9.5 并发测试指南

#### 9.5.1 竞态条件检测
```bash
# 运行带竞态检测的测试
go test -race ./...

# 如果发现竞态条件，查看详细报告
go test -race -v ./... 2> race_report.txt
```

#### 9.5.2 并发测试示例
```go
func TestConcurrentSessionAccess(t *testing.T) {
    s := createTestSession(t)
    
    var wg sync.WaitGroup
    errors := make(chan error, 10)
    
    // 并发访问会话
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            if err := s.Attach(); err != nil {
                errors <- err
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    for err := range errors {
        t.Errorf("Concurrent access error: %v", err)
    }
}
```

### 9.6 测试维护

#### 9.6.1 定期维护任务
- **每周**：检查测试覆盖率报告，识别未覆盖的关键代码
- **每月**：审查慢测试（>1秒），优化测试性能
- **每季度**：更新 Mock 数据，确保与实际 API 同步
- **每年**：重构重复测试代码，提高可维护性

#### 9.6.2 测试文档更新
当以下情况发生时，更新测试文档：
- 添加新的测试类型或测试工具
- 修改测试覆盖率目标
- 更改 CI/CD 配置
- 添加新的平台支持

#### 9.6.3 测试失败处理
1. **本地验证**：在本地复现失败的测试
2. **环境检查**：确认测试环境配置正确
3. **依赖更新**：检查是否有依赖更新导致的问题
4. **代码审查**：审查最近的代码变更
5. **修复或标记**：修复问题或标记为已知问题

### 9.7 测试最佳实践

#### 9.7.1 单元测试
- **快速**：每个测试应在毫秒级完成
- **独立**：测试之间不相互依赖
- **可重复**：多次运行结果一致
- **清晰**：测试名称和断言消息清晰易懂

#### 9.7.2 集成测试
- **真实环境**：使用接近生产的环境
- **必要依赖**：只测试必要的外部依赖
- **清理资源**：确保测试后清理所有资源
- **错误处理**：验证错误处理逻辑

#### 9.7.3 性能测试
- **基线建立**：建立性能基线，用于回归检测
- **环境一致**：在一致的环境中运行
- **多次运行**：多次运行取平均值
- **资源监控**：监控内存和CPU使用

### 9.8 测试工具和命令

```bash
make test                  # go test -v ./...
make build                 # 构建 build/sgreen
make test-behavior         # 仅行为测试
gofmt -l .                 # 格式检查（CI 强制）
go vet ./...               # 静态检查
go test -race ./...        # 竞态
go test -covermode=atomic -coverpkg=./... -coverprofile=c.out ./...
go tool cover -func=c.out  # 覆盖率摘要
```


## 10. 注意事项

1. **Go 测试约定**：遵循 Go 语言的测试命名约定
2. **并发安全**：确保所有测试用例是并发安全的
3. **资源清理**：每个测试用例都要正确清理资源
4. **Mock 策略**：合理使用 mock 避免测试依赖外部服务
5. **测试隔离**：确保测试之间相互独立
6. **平台兼容**：确保测试在所有支持的平台上通过
7. **性能监控**：定期检查测试性能，避免慢测试
8. **覆盖率目标**：确保核心模块达到覆盖率目标
9. **文档更新**：及时更新测试文档和注释
10. **代码审查**：所有测试代码也需要经过代码审查

这个完善的测试计划借鉴了 screen C 项目的结构化测试方法，同时针对 Go 语言的特性进行了全面优化，包含了性能测试、并发测试、跨平台测试、CI/CD 集成等现代测试实践，确保 sgreen 项目的可靠性和正确性。