# PowerShell Safe Invocation Reference

只读取当前任务需要的章节。

## 1. 解析层

一条生成出来的命令可能经过多层解析：

```text
Codex / tool call
  -> PowerShell
  -> native program argument parser
  -> ssh / wsl / python / git / docker
  -> remote shell or target program
```

每一层都可能重新解释引号、反斜杠、美元符号、反引号、管道、重定向、括号、JSON、正则和 Unicode 文本。

避免这种深度嵌套命令：

```text
cmd.exe /c pwsh.exe -Command "$json = '{\"name\":\"test\"}'; ..."
```

优先写 `.ps1`：

```powershell
$data = [ordered]@{
    name = 'test'
}

$data |
    ConvertTo-Json -Depth 10 |
    Set-Content -LiteralPath $outputPath -Encoding utf8
```

执行方式：

```text
pwsh.exe -NoLogo -NoProfile -NonInteractive -File script.ps1
```

## 2. PowerShell 版本

PowerShell 7 和 Windows PowerShell 5.1 可以同时存在：

```text
pwsh.exe        PowerShell 7
powershell.exe  Windows PowerShell 5.1
```

验证当前进程：

```powershell
$PSVersionTable
$PSNativeCommandArgumentPassing
```

PowerShell 7.3 及以后改进了原生命令参数传递。Windows 上常见默认值是 `Windows`。不要在没有明确兼容性证据时改成 `Legacy`。

一次命令显示 PowerShell 7，不代表后续所有 agent 调用都使用 `pwsh.exe`。包装器可能调用不同 shell。

## 3. 原生参数数组

正确写法：

```powershell
$exe = 'C:\Program Files\App\tool.exe'

$args = @(
    '--input'
    'C:\Data Folder\input.json'
    '--name'
    'value with spaces'
    '--empty'
    ''
)

& $exe @args
$exitCode = $LASTEXITCODE
```

下面三种含义不同：

- 省略参数
- 空字符串 `''`
- `$null`

不要随手删除空参数。

调试参数边界：

```powershell
$args | ForEach-Object {
    '[{0}] Length={1}' -f $_, $_.Length
}
```

在下一个原生命令覆盖退出码前保存 `$LASTEXITCODE`：

```powershell
& $exe @args
$exitCode = $LASTEXITCODE

if ($exitCode -ne 0) {
    throw "$exe failed with exit code $exitCode"
}
```

有些工具定义了非零成功码。确认目标工具的退出码契约前，不要盲目套用 `-ne 0`。

## 4. Cmdlet Splatting 与错误处理

cmdlet 参数使用 hashtable：

```powershell
$params = @{
    LiteralPath = $source
    Destination = $destination
    Force       = $true
    ErrorAction = 'Stop'
}

Copy-Item @params
```

需要失败即停时使用 terminating error：

```powershell
$ErrorActionPreference = 'Stop'

try {
    Copy-Item -LiteralPath $source -Destination $destination -ErrorAction Stop
}
catch {
    throw "Copy failed: $($_.Exception.Message)"
}
```

`$LASTEXITCODE` 用于原生命令和脚本退出码，不用于普通 cmdlet 成功判断。

## 5. 字符串与转义

字面量路径：

```powershell
$path = 'C:\Program Files\App\data.json'
```

需要展开：

```powershell
$message = "Output path: $path"
```

不要使用 Bash 风格转义：

```powershell
# Wrong
"\"quoted\""
```

使用：

```powershell
'"quoted"'
```

必要时才用：

```powershell
"`"quoted`""
```

变量名后紧跟文本时用花括号：

```powershell
"${name}_suffix"
```

属性访问用子表达式：

```powershell
"Exit code: $($process.ExitCode)"
```

## 6. 避免反引号续行

避免：

```powershell
& $exe `
    '--input' `
    $inputPath `
    '--output' `
    $outputPath
```

反引号后有尾随空格时，续行会静默失效。

优先使用：

```powershell
$args = @(
    '--input'
    $inputPath
    '--output'
    $outputPath
)

& $exe @args
```

管道、逗号、操作符和未闭合分隔符后面的自然换行也是安全的。

## 7. Here-String 与 JSON

字面量多行文本：

```powershell
$text = @'
{
  "name": "$literal"
}
'@
```

需要展开的多行文本：

```powershell
$text = @"
Name: $name
"@
```

结束标记必须单独位于行首。

JSON 优先序列化对象：

```powershell
$data = [ordered]@{
    name = $name
    path = $path
    flags = @('a', 'b')
}

$data |
    ConvertTo-Json -Depth 10 |
    Set-Content -LiteralPath $jsonPath -Encoding utf8
```

除非无法避免，不要手写 JSON 转义。

## 8. Start-Process

只有需要这些行为时才用 `Start-Process`：

- 提权
- 新窗口或隐藏窗口
- 脱离当前终端的后台进程
- shell 文件关联
- 明确依赖它的进程对象语义

简单示例：

```powershell
$process = Start-Process `
    -FilePath $exe `
    -ArgumentList '--mode test' `
    -Wait `
    -PassThru

if ($process.ExitCode -ne 0) {
    throw "Process failed with exit code $($process.ExitCode)"
}
```

注意：`-ArgumentList` 会拼成一条命令行字符串。

需要精确参数边界时使用 `ProcessStartInfo.ArgumentList`：

```powershell
$psi = [System.Diagnostics.ProcessStartInfo]::new()
$psi.FileName = $exe
$psi.UseShellExecute = $false

$psi.ArgumentList.Add('--input')
$psi.ArgumentList.Add('C:\Path With Spaces\input.json')
$psi.ArgumentList.Add('--empty')
$psi.ArgumentList.Add('')

$process = [System.Diagnostics.Process]::Start($psi)
$process.WaitForExit()

if ($process.ExitCode -ne 0) {
    throw "Process failed with exit code $($process.ExitCode)"
}
```

## 9. 捕获 stdout 与 stderr

需要分别捕获 stdout 和 stderr 时使用 `ProcessStartInfo`：

```powershell
$psi = [System.Diagnostics.ProcessStartInfo]::new()
$psi.FileName = $exe
$psi.UseShellExecute = $false
$psi.RedirectStandardOutput = $true
$psi.RedirectStandardError = $true

foreach ($arg in $args) {
    $psi.ArgumentList.Add($arg)
}

$process = [System.Diagnostics.Process]::Start($psi)
$stdout = $process.StandardOutput.ReadToEnd()
$stderr = $process.StandardError.ReadToEnd()
$process.WaitForExit()

if ($process.ExitCode -ne 0) {
    throw "Command failed with exit code $($process.ExitCode): $stderr"
}
```

不要把二进制输出通过文本 cmdlet 管道处理。

## 10. 文件与路径安全

真实路径使用 `-LiteralPath`：

```powershell
Get-Item -LiteralPath $path
Copy-Item -LiteralPath $source -Destination $destination
Remove-Item -LiteralPath $path
```

只有明确需要通配符展开时才用 `-Path`。

用 `Join-Path` 或 `[System.IO.Path]::Combine()` 构造路径：

```powershell
$path = Join-Path -Path $root -ChildPath 'subdir\file.txt'
```

已存在路径使用 `Resolve-Path -LiteralPath`。目标可能尚不存在时，使用 `[System.IO.Path]::GetFullPath()` 归一化。

### 递归修改边界校验

```powershell
$root = (Resolve-Path -LiteralPath 'C:\ExpectedRoot').Path
$target = (Resolve-Path -LiteralPath $candidate).Path

$rootPrefix = $root.TrimEnd(
    [System.IO.Path]::DirectorySeparatorChar,
    [System.IO.Path]::AltDirectorySeparatorChar
) + [System.IO.Path]::DirectorySeparatorChar

if (-not $target.StartsWith(
    $rootPrefix,
    [System.StringComparison]::OrdinalIgnoreCase
)) {
    throw "Refusing to modify path outside expected root: $target"
}

Remove-Item -LiteralPath $target -Recurse -Force
```

只写 `$target.StartsWith($root)` 不够，因为 `C:\WorkBackup` 不是 `C:\Work` 的子目录。

还要拒绝：

- 空路径
- 文件系统根目录
- 预期根目录本身，除非明确允许
- 无法解析或不符合预期的目标

删除和移动尽量在同一个 shell 中完成，不要在 PowerShell 枚举路径后交给另一个 shell。

## 11. SSH、WSL 与 Bash

错误示例：

```powershell
ssh root@example.com "backup=/opt/app/app.backup.$(date +%Y%m%d%H%M%S)"
```

这里 `$()` 会被本地 PowerShell 展开，远端 Bash 收不到原始脚本。

简单远端 Bash：

```powershell
ssh root@example.com 'backup=/opt/app/app.backup.$(date +%Y%m%d%H%M%S); echo "$backup"'
```

复杂远端 Bash：

```powershell
$script = @'
set -e
backup=/opt/app/app.backup.$(date +%Y%m%d%H%M%S)
systemctl stop app.service
cp -a /opt/app/app "$backup"
install -o root -g root -m 755 /tmp/app.new /opt/app/app
systemctl start app.service
'@

$script | ssh root@example.com 'bash -s'
```

不要在 PowerShell 中直接写 Bash heredoc：

```powershell
python - <<'PY'
print("hello")
PY
```

改用：

```powershell
@'
print("hello")
'@ | python -
```

复杂 WSL 命令：

```powershell
$script = @'
set -e
cd /mnt/c/project
cargo build --release
'@

$script | wsl -d Ubuntu -- bash -s
```

## 12. cmd.exe、批处理与 Stop-Parsing

只有需要 cmd 特有语义时才用 `cmd.exe /c`，例如：

- cmd built-in
- 必须依赖 `.cmd` 或 `.bat` 行为
- cmd 特有展开或重定向

PowerShell 7 已支持：

```powershell
command1 && command2
command1 || command2
```

`.cmd`、`.bat` 和 `cmd.exe` 会增加一层解析器，也可能触发旧式参数行为。

默认避免 `--%`。它是 Windows 特有行为，会关闭后续 PowerShell 解析。只有固定字面量原生命令无法用参数数组可靠表达时才考虑。

## 13. 环境变量

读取：

```powershell
$env:NAME
```

设置给当前进程及其子进程：

```powershell
$env:NAME = 'value'
```

不要在 PowerShell 中使用 `%NAME%`。

子进程中的环境变量变更不会传播回父 PowerShell 进程。

## 14. 编码、BOM 与中文

跨平台源代码、Markdown、YAML 和 JSON 优先使用：

```text
UTF-8 no BOM
```

Windows PowerShell 5.1 的 `-Encoding utf8` 通常会写入 UTF-8 BOM。PowerShell 7 支持 `-Encoding utf8NoBOM`。

检查 BOM：

```powershell
$bytes = [System.IO.File]::ReadAllBytes('README.md')
($bytes[0..2] | ForEach-Object { $_.ToString('X2') }) -join ' '
```

写入 UTF-8 no BOM：

```powershell
$utf8NoBom = [System.Text.UTF8Encoding]::new($false)
[System.IO.File]::WriteAllText('README.md', $text, $utf8NoBom)
```

终端乱码不证明文件损坏。验证真实字节或用明确编码读取：

```powershell
[System.IO.File]::ReadAllText(
    'README.md',
    [System.Text.UTF8Encoding]::new($false)
)
```

截断 UTF-8 文本时，不要按任意 byte index 截断，除非截断点是合法字符边界。日志摘要、AI 回复、CLI 输出长度限制尤其要注意。

二进制数据使用 byte API：

```powershell
[System.IO.File]::WriteAllBytes($path, $bytes)
```

不要用文本 cmdlet 处理二进制内容。

## 15. Windows 运行中 exe 锁

Windows 不能覆盖正在运行的可执行文件。常见错误：

```text
failed to remove file target\debug\app.exe
Access is denied. (os error 5)
```

查找并停止旧进程：

```powershell
Get-Process | Where-Object { $_.ProcessName -like '*app*' }
Stop-Process -Id <pid>
```

自动化构建前，确认目标 exe 没有被上一轮 `cargo run`、测试进程或后台服务占用。

## 16. 诊断清单

参数损坏时先收集：

```powershell
$PSVersionTable
$PSNativeCommandArgumentPassing
Get-Command pwsh
Get-Command powershell
Get-Command $exe -ErrorAction SilentlyContinue
```

然后按顺序简化：

1. 去掉 `cmd.exe`。
2. 去掉 `-Command`。
3. 去掉 `Invoke-Expression`。
4. 去掉手写嵌套引号。
5. 把代码放进最小 `.ps1` 文件。
6. 用 `& $exe @args` 直接调用原生命令。
7. 打印每个参数及其长度。
8. 原生命令结束后立刻保存 `$LASTEXITCODE`。

## 17. 相关但非核心

Linux binary ABI、glibc mismatch 和 musl static build 是部署兼容性问题。它们可能出现在 PowerShell 驱动的部署流程中，但不是 PowerShell/Codex 执行层问题。除非当前任务是部署命令调用本身，否则不要把它们扩进本 skill。
