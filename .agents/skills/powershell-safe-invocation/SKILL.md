---
name: powershell-safe-invocation
description: Use when writing or running PowerShell on Windows, especially native programs, quoted paths, escaping, pwsh vs powershell.exe, Start-Process, ProcessStartInfo, file operations, recursive delete or move, SSH/WSL/Bash calls, encoding, BOM, Chinese text, running exe locks, or shell troubleshooting.
---

# PowerShell Safe Invocation

当 PowerShell 是 Codex 或自动化的执行层时使用本 skill。核心风险是把原本结构化的参数压成一整段字符串，再交给多层 shell 重新解析，导致引号、空格、`$()`、JSON、中文或路径边界被破坏。

## 核心规则

不要把复杂命令再塞进一层引号里。

按这个顺序选择最简单且安全的方式：

1. PowerShell cmdlet。
2. 原生命令使用 `& $exe @args`。
3. 复杂脚本写入临时 `.ps1`，用 `pwsh.exe -NoLogo -NoProfile -NonInteractive -File script.ps1` 执行。
4. 需要精确参数边界或分离 stdout/stderr 时使用 `ProcessStartInfo.ArgumentList`。
5. 只有需要提权、新/隐藏窗口、脱离当前终端或 shell 关联行为时才用 `Start-Process`。
6. 只有需要 cmd 语义、`.cmd` / `.bat` 行为或 cmd 特有展开时才用 `cmd.exe /c`。
7. `Invoke-Expression` 只能作为可信 PowerShell 源码的最后选择。

任务涉及 SSH、WSL、Bash、JSON、正则、递归文件修改、编码、stdout/stderr 捕获或 shell 调用失败时，读取 `reference.md` 的对应章节。

## Shell

优先使用 PowerShell 7 的 `pwsh.exe`，除非明确需要 Windows PowerShell 5.1。

不确定当前 shell 时先验证：

```powershell
$PSVersionTable.PSVersion
$PSNativeCommandArgumentPassing
Get-Command pwsh -ErrorAction SilentlyContinue
Get-Command powershell -ErrorAction SilentlyContinue
```

不要假设安装 PowerShell 7 后 `powershell.exe` 会变成 PowerShell 7：

- `pwsh.exe` = PowerShell 7
- `powershell.exe` = Windows PowerShell 5.1

非平凡自动化脚本优先使用：

```text
pwsh.exe -NoLogo -NoProfile -NonInteractive -File script.ps1
```

不要习惯性添加 `-ExecutionPolicy Bypass`。只有可信脚本确实被策略阻止，且策略允许覆盖时才添加。

## 原生命令

能分开传参时，不要拼一个大字符串。

```powershell
$exe = 'C:\Path With Spaces\tool.exe'
$args = @(
    '--input'
    'C:\Data Folder\input.json'
    '--empty'
    ''
)

& $exe @args

$exitCode = $LASTEXITCODE
if ($exitCode -ne 0) {
    throw "$exe failed with exit code $exitCode"
}
```

规则：

- 每个原生参数都是数组中的一个独立元素。
- 可执行文件路径存进变量后，用 `&` 调用。
- 原生命令结束后立刻保存 `$LASTEXITCODE`。
- 不要用 `Invoke-Expression`。
- 不要为了启动可执行文件额外套一层 `cmd.exe /c`。
- 不要在 PowerShell 里使用 Bash 风格的 `\"` 转义。
- 不要随手过滤空参数；省略参数、`''` 和 `$null` 含义不同。

## Cmdlet 与文件

cmdlet 使用 hashtable splatting：

```powershell
$params = @{
    LiteralPath = 'C:\Data[1]\input.txt'
    Destination = 'C:\Output'
    Force       = $true
    ErrorAction = 'Stop'
}

Copy-Item @params
```

规则：

- 真实路径优先使用 `-LiteralPath`，除非明确需要通配符展开。
- cmdlet 失败需要停止时使用 `$ErrorActionPreference = 'Stop'` 或 `-ErrorAction Stop`。
- 不要用 `$LASTEXITCODE` 判断普通 cmdlet 是否成功。
- 递归删除、移动或覆盖前，解析绝对 root 和 target，并验证 target 在预期 root 内。
- 文件枚举和修改尽量留在同一个 PowerShell 流程里完成，不要枚举路径后拼字符串交给另一个 shell 删除或移动。

## 复杂命令

命令包含这些内容时，改用 `.ps1` 文件或 here-string：

- 多行代码
- 嵌套引号
- JSON、XML、正则
- 管道或重定向
- SSH 远端脚本
- WSL/Bash 脚本
- `$()`、`$VAR`、`%VAR%`
- 非 ASCII 路径或中文输出

JSON 优先构造对象后序列化：

```powershell
$data = [ordered]@{
    name = $name
    path = $path
}

$json = $data | ConvertTo-Json -Depth 10
```

字面量字符串和路径使用单引号。只有需要 PowerShell 展开时才使用双引号。避免反引号续行；改用数组、hashtable、splatting、括号或 script block。

## 远端 Shell

PowerShell 会先于 SSH、WSL、Bash、Python 或 Git 解析命令。

避免：

```powershell
ssh root@example.com "backup=/opt/app/app.backup.$(date +%Y%m%d%H%M%S)"
```

简单远端 Bash 可以用 PowerShell 单引号；复杂脚本用 here-string 通过 stdin 传给 `bash -s`：

```powershell
$script = @'
set -e
backup=/opt/app/app.backup.$(date +%Y%m%d%H%M%S)
echo "$backup"
'@

$script | ssh root@example.com 'bash -s'
```

不要在 PowerShell 里直接写 Bash heredoc：

```powershell
python - <<'PY'
print("hello")
PY
```

改用 PowerShell here-string：

```powershell
@'
print("hello")
'@ | python -
```

## 编码与锁

跨平台文本优先使用 UTF-8 without BOM。终端乱码不代表文件损坏；先检查字节或用明确编码读取。

不要按任意 byte index 截断 UTF-8 字符串，除非确认截断点是合法字符边界。

Windows 不能覆盖正在运行的 `.exe`。如果构建报 `target\debug\*.exe` access denied，先找到并停止旧进程再重建。

## 排障顺序

当参数被吞、引号错乱、中文异常或命令行为不符合预期时，按顺序剥离复杂度：

1. 去掉 `cmd.exe /c`。
2. 去掉 `-Command`。
3. 去掉 `Invoke-Expression`。
4. 去掉手写嵌套引号。
5. 把代码放进最小 `.ps1` 文件。
6. 用 `& $exe @args` 直接调用原生命令。
7. 打印每个参数和长度。
8. 原生命令结束后立刻保存 `$LASTEXITCODE`。
