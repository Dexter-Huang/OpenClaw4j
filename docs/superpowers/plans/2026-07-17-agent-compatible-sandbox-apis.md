# Agent-Compatible Sandbox APIs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `OpenClaw4j-Sandbox` 中实现 Agent 可用的 AIO 风格 file、bash 和 MCP 兼容接口。

**Architecture:** 在现有 Rust Axum 服务中新增 `path_guard`、`file_api`、`bash_api`、`mcp_api` 四个模块，`main.rs` 只装配路由和共享状态。`file_api` 与 `bash_api` 共用 workspace 路径约束，`mcp_api` 通过内部 service 调用 file/bash 能力并提供 JSON-RPC 与 REST 包装。

**Tech Stack:** Rust 2021、Axum 0.7、Tokio、Serde、Tempfile、Uuid；新增 `regex`、`walkdir`、`globset`、`base64`。

## Global Constraints

- 文档、设计说明和新增代码注释默认使用中文。
- Git 命令必须从 `D:\IDEA_project\OpenClaw4j` 根仓库运行。
- `OpenClaw4j-Bankend/`、`OpenClaw4j-Frontend/`、`OpenClaw4j-Sandbox/` 都是根仓库普通子目录。
- 本计划不实现完整 AIO Sandbox。
- 本计划不实现浏览器、Jupyter、Code Server、Node.js 会话、display、proxy、skills 等 AIO 子系统。
- 本计划不实现 `/v1/file/upload`、`/v1/file/download`、file watch、SSE 事件流。
- 本计划不实现 `sudo` 参数和受保护系统路径写入。
- 本计划不实现外部 MCP Hub 聚合，只提供当前 Rust sandbox 内置工具。
- 所有 file/bash 路径默认限制在 `OPENCLAW_SANDBOX_WORKSPACE_DIR` 内。
- `/health` 和 `/v1/execute` 行为必须保持兼容。
- Rust 改动完成前的最小验证以 `cargo test` 为准。
- 不主动运行后端 Java `spotless:check`、`checkstyle:check`、`spotbugs:check`，除非用户明确要求。

---

## File Structure

- Modify: `OpenClaw4j-Sandbox/Cargo.toml`
  - 增加 `regex`、`walkdir`、`globset`、`base64` 依赖。
- Modify: `OpenClaw4j-Sandbox/src/config.rs`
  - 增加 workspace、bash、file 限制配置字段和环境变量读取。
- Create: `OpenClaw4j-Sandbox/src/path_guard.rs`
  - 负责 workspace 路径归一化、目录创建和逃逸检查。
- Create: `OpenClaw4j-Sandbox/src/file_api.rs`
  - 负责 `/v1/file/*` 请求/响应模型、文件 service、路由 handler。
- Create: `OpenClaw4j-Sandbox/src/bash_api.rs`
  - 负责 bash session/command 状态、进程管理、输出缓存和路由 handler。
- Create: `OpenClaw4j-Sandbox/src/mcp_api.rs`
  - 负责 MCP tool registry、`/mcp` JSON-RPC 和 `/v1/mcp/*` REST 包装。
- Modify: `OpenClaw4j-Sandbox/src/main.rs`
  - 注册新增模块、共享状态和路由。
- Modify: `OpenClaw4j-Sandbox/README.md`
  - 补充 workspace、file、bash、MCP 调用示例和不兼容项。

## Task 1: Workspace Config And Path Guard

**Files:**
- Modify: `OpenClaw4j-Sandbox/Cargo.toml`
- Modify: `OpenClaw4j-Sandbox/src/config.rs`
- Create: `OpenClaw4j-Sandbox/src/path_guard.rs`

**Interfaces:**
- Produces: `SandboxConfig.workspace_dir: PathBuf`
- Produces: `SandboxConfig.bash_default_timeout_ms: u64`
- Produces: `SandboxConfig.bash_hard_timeout_ms: u64`
- Produces: `SandboxConfig.bash_output_limit_bytes: usize`
- Produces: `SandboxConfig.file_read_limit_bytes: usize`
- Produces: `PathGuard::new(workspace_root: PathBuf) -> Self`
- Produces: `PathGuard::ensure_workspace(&self) -> Result<(), String>`
- Produces: `PathGuard::resolve(&self, input: impl AsRef<Path>) -> Result<PathBuf, PathGuardError>`
- Produces: `PathGuardError { input: String, message: String, error_type: &'static str }`

- [ ] **Step 1: Add failing path guard tests**

Add this test module to new file `OpenClaw4j-Sandbox/src/path_guard.rs`:

```rust
#[cfg(test)]
mod tests {
    use super::{PathGuard, PathGuardError};
    use std::path::PathBuf;

    fn guard() -> PathGuard {
        PathGuard::new(PathBuf::from("/tmp/openclaw4j-workspace"))
    }

    #[test]
    fn resolves_relative_paths_inside_workspace() {
        let resolved = guard().resolve("reports/out.txt").unwrap();
        assert_eq!(resolved, PathBuf::from("/tmp/openclaw4j-workspace/reports/out.txt"));
    }

    #[test]
    fn allows_absolute_paths_inside_workspace() {
        let resolved = guard()
            .resolve("/tmp/openclaw4j-workspace/reports/out.txt")
            .unwrap();
        assert_eq!(resolved, PathBuf::from("/tmp/openclaw4j-workspace/reports/out.txt"));
    }

    #[test]
    fn rejects_parent_directory_escape() {
        let error = guard().resolve("../secret.txt").unwrap_err();
        assert_eq!(error.error_type, "invalid_path");
        assert!(error.message.contains("workspace"));
    }

    #[test]
    fn rejects_absolute_paths_outside_workspace() {
        let error = guard().resolve("/etc/passwd").unwrap_err();
        assert_eq!(error.error_type, "invalid_path");
    }
}
```

- [ ] **Step 2: Run the failing test**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
cargo test path_guard -- --nocapture
```

Expected: FAIL because `path_guard` module is not registered or `PathGuard` is not implemented.

- [ ] **Step 3: Register dependencies and module**

In `OpenClaw4j-Sandbox/Cargo.toml`, add:

```toml
base64 = "0.22"
globset = "0.4"
regex = "1"
walkdir = "2"
```

In `OpenClaw4j-Sandbox/src/main.rs`, add:

```rust
mod path_guard;
```

- [ ] **Step 4: Implement path guard**

Create `OpenClaw4j-Sandbox/src/path_guard.rs` with:

```rust
use std::fs;
use std::path::{Component, Path, PathBuf};

#[derive(Clone, Debug)]
pub struct PathGuard {
    workspace_root: PathBuf,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct PathGuardError {
    pub input: String,
    pub message: String,
    pub error_type: &'static str,
}

impl PathGuard {
    pub fn new(workspace_root: PathBuf) -> Self {
        Self {
            workspace_root: normalize_path(&workspace_root),
        }
    }

    pub fn ensure_workspace(&self) -> Result<(), String> {
        fs::create_dir_all(&self.workspace_root).map_err(|err| err.to_string())
    }

    pub fn workspace_root(&self) -> &Path {
        &self.workspace_root
    }

    pub fn resolve(&self, input: impl AsRef<Path>) -> Result<PathBuf, PathGuardError> {
        let input_path = input.as_ref();
        let candidate = if input_path.is_absolute() {
            normalize_path(input_path)
        } else {
            normalize_path(&self.workspace_root.join(input_path))
        };

        if candidate.starts_with(&self.workspace_root) {
            return Ok(candidate);
        }

        Err(PathGuardError {
            input: input_path.to_string_lossy().to_string(),
            message: format!(
                "path must stay inside workspace {}",
                self.workspace_root.display()
            ),
            error_type: "invalid_path",
        })
    }
}

fn normalize_path(path: &Path) -> PathBuf {
    let mut normalized = PathBuf::new();
    for component in path.components() {
        match component {
            Component::CurDir => {}
            Component::ParentDir => {
                normalized.pop();
            }
            Component::Prefix(prefix) => normalized.push(prefix.as_os_str()),
            Component::RootDir => normalized.push(component.as_os_str()),
            Component::Normal(value) => normalized.push(value),
        }
    }
    normalized
}
```

- [ ] **Step 5: Extend sandbox config**

Modify `OpenClaw4j-Sandbox/src/config.rs`:

```rust
#[derive(Clone, Debug)]
pub struct SandboxConfig {
    pub bind: String,
    pub work_dir: PathBuf,
    pub workspace_dir: PathBuf,
    pub default_timeout_ms: u64,
    pub max_timeout_ms: u64,
    pub memory_limit: String,
    pub process_limit: u32,
    pub stdout_limit_bytes: usize,
    pub stderr_limit_bytes: usize,
    pub deps_dir: PathBuf,
    pub python_runtime: String,
    pub node_path: String,
    pub bash_default_timeout_ms: u64,
    pub bash_hard_timeout_ms: u64,
    pub bash_output_limit_bytes: usize,
    pub file_read_limit_bytes: usize,
}
```

In `from_env()`, add:

```rust
workspace_dir: PathBuf::from(env_value(
    "OPENCLAW_SANDBOX_WORKSPACE_DIR",
    "/tmp/openclaw4j-workspace",
)),
bash_default_timeout_ms: env_u64("OPENCLAW_SANDBOX_BASH_DEFAULT_TIMEOUT_MS", 30_000),
bash_hard_timeout_ms: env_u64("OPENCLAW_SANDBOX_BASH_HARD_TIMEOUT_MS", 120_000),
bash_output_limit_bytes: env_usize("OPENCLAW_SANDBOX_BASH_OUTPUT_LIMIT_BYTES", 65_536),
file_read_limit_bytes: env_usize("OPENCLAW_SANDBOX_FILE_READ_LIMIT_BYTES", 1_048_576),
```

Update existing test config literals with:

```rust
workspace_dir: PathBuf::from("/tmp/openclaw4j-workspace"),
bash_default_timeout_ms: 30_000,
bash_hard_timeout_ms: 120_000,
bash_output_limit_bytes: 65_536,
file_read_limit_bytes: 1_048_576,
```

- [ ] **Step 6: Verify Task 1 passes**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
cargo test path_guard config -- --nocapture
```

Expected: PASS.

- [ ] **Step 7: Commit Task 1**

Run from repo root:

```powershell
cd D:\IDEA_project\OpenClaw4j
git add OpenClaw4j-Sandbox/Cargo.toml OpenClaw4j-Sandbox/Cargo.lock OpenClaw4j-Sandbox/src/config.rs OpenClaw4j-Sandbox/src/main.rs OpenClaw4j-Sandbox/src/path_guard.rs
git commit -m "feat: add sandbox workspace path guard"
```

## Task 2: Common File Response And Basic File API

**Files:**
- Create: `OpenClaw4j-Sandbox/src/file_api.rs`
- Modify: `OpenClaw4j-Sandbox/src/main.rs`

**Interfaces:**
- Consumes: `PathGuard::resolve`
- Produces: `file_api::router() -> Router<AppState>`
- Produces: `FileService::read_file`, `write_file`, `replace_file`, `list_dir`
- Produces routes: `POST /v1/file/read`, `POST /v1/file/write`, `POST /v1/file/replace`, `POST /v1/file/list`

- [ ] **Step 1: Add failing file API service tests**

Create `OpenClaw4j-Sandbox/src/file_api.rs` with only the test module first:

```rust
#[cfg(test)]
mod tests {
    use super::{FileService, ReadFileRequest, ReplaceFileRequest, WriteFileRequest};
    use crate::path_guard::PathGuard;
    use serde_json::Value;
    use std::fs;
    use tempfile::tempdir;

    fn service() -> (tempfile::TempDir, FileService) {
        let dir = tempdir().unwrap();
        let service = FileService::new(PathGuard::new(dir.path().to_path_buf()), 1024 * 1024);
        (dir, service)
    }

    #[tokio::test]
    async fn writes_and_reads_utf8_file() {
        let (_dir, service) = service();

        let write = service
            .write_file(WriteFileRequest {
                file: "note.txt".to_string(),
                content: "hello".to_string(),
                encoding: None,
                append: None,
                leading_newline: None,
                trailing_newline: Some(true),
                sudo: None,
            })
            .await;
        assert!(write.success);
        assert_eq!(write.data["bytes_written"], Value::from(6));

        let read = service
            .read_file(ReadFileRequest {
                file: "note.txt".to_string(),
                start_line: None,
                end_line: None,
                sudo: None,
            })
            .await;
        assert!(read.success);
        assert_eq!(read.data["content"], Value::from("hello\n"));
    }

    #[tokio::test]
    async fn rejects_workspace_escape_as_business_error() {
        let (_dir, service) = service();

        let read = service
            .read_file(ReadFileRequest {
                file: "../secret.txt".to_string(),
                start_line: None,
                end_line: None,
                sudo: None,
            })
            .await;

        assert!(!read.success);
        assert_eq!(read.data["error_type"], Value::from("invalid_path"));
    }

    #[tokio::test]
    async fn replaces_text_in_file() {
        let (dir, service) = service();
        fs::write(dir.path().join("app.txt"), "hello world").unwrap();

        let result = service
            .replace_file(ReplaceFileRequest {
                file: "app.txt".to_string(),
                old_str: "world".to_string(),
                new_str: "sandbox".to_string(),
                replace_all: None,
                sudo: None,
            })
            .await;

        assert!(result.success);
        assert_eq!(fs::read_to_string(dir.path().join("app.txt")).unwrap(), "hello sandbox");
    }
}
```

- [ ] **Step 2: Run the failing file API tests**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
cargo test file_api -- --nocapture
```

Expected: FAIL because `file_api` module types and service are missing.

- [ ] **Step 3: Implement request and response models**

Add to `OpenClaw4j-Sandbox/src/file_api.rs`:

```rust
use crate::path_guard::{PathGuard, PathGuardError};
use axum::extract::State;
use axum::routing::post;
use axum::{Json, Router};
use base64::Engine;
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::fs;
use std::path::PathBuf;

#[derive(Clone)]
pub struct FileService {
    path_guard: PathGuard,
    read_limit_bytes: usize,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct ApiResponse {
    pub success: bool,
    pub message: String,
    pub data: Value,
}

#[derive(Debug, Deserialize)]
pub struct ReadFileRequest {
    pub file: String,
    #[serde(default)]
    pub start_line: Option<usize>,
    #[serde(default)]
    pub end_line: Option<usize>,
    #[serde(default)]
    pub sudo: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct WriteFileRequest {
    pub file: String,
    pub content: String,
    #[serde(default)]
    pub encoding: Option<String>,
    #[serde(default)]
    pub append: Option<bool>,
    #[serde(default)]
    pub leading_newline: Option<bool>,
    #[serde(default)]
    pub trailing_newline: Option<bool>,
    #[serde(default)]
    pub sudo: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct ReplaceFileRequest {
    pub file: String,
    pub old_str: String,
    pub new_str: String,
    #[serde(default)]
    pub replace_all: Option<bool>,
    #[serde(default)]
    pub sudo: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct ListDirRequest {
    pub path: String,
    #[serde(default)]
    pub recursive: Option<bool>,
    #[serde(default)]
    pub show_hidden: Option<bool>,
    #[serde(default)]
    pub include_size: Option<bool>,
}
```

- [ ] **Step 4: Implement FileService basic methods**

Add:

```rust
impl FileService {
    pub fn new(path_guard: PathGuard, read_limit_bytes: usize) -> Self {
        Self {
            path_guard,
            read_limit_bytes,
        }
    }

    pub async fn read_file(&self, request: ReadFileRequest) -> ApiResponse {
        let path = match self.resolve_file("read", &request.file) {
            Ok(path) => path,
            Err(response) => return response,
        };

        let metadata = match fs::metadata(&path) {
            Ok(metadata) => metadata,
            Err(err) => return file_error("read", &path, err),
        };
        if metadata.len() as usize > self.read_limit_bytes && request.start_line.is_none() {
            return business_error(
                "read",
                &path,
                "file is too large to read without line range",
                "too_large",
                false,
            );
        }

        let content = match fs::read_to_string(&path) {
            Ok(content) => content,
            Err(err) => return file_error("read", &path, err),
        };
        let selected = select_lines(&content, request.start_line, request.end_line);
        ApiResponse::ok(
            "File read successfully",
            json!({
                "file": path.to_string_lossy(),
                "content": selected.content,
                "line_count": selected.line_count
            }),
        )
    }

    pub async fn write_file(&self, request: WriteFileRequest) -> ApiResponse {
        let path = match self.path_guard.resolve(&request.file) {
            Ok(path) => path,
            Err(err) => return path_error("write", err),
        };
        let created = !path.exists();
        if let Some(parent) = path.parent() {
            if let Err(err) = fs::create_dir_all(parent) {
                return file_error("write", &path, err);
            }
        }

        let mut bytes = match decode_content(&request.content, request.encoding.as_deref()) {
            Ok(bytes) => bytes,
            Err(message) => {
                return business_error("write", &path, &message, "decode_error", false);
            }
        };
        if request.leading_newline.unwrap_or(false) {
            let mut prefixed = b"\n".to_vec();
            prefixed.extend(bytes);
            bytes = prefixed;
        }
        if request.trailing_newline.unwrap_or(false) && !bytes.ends_with(b"\n") {
            bytes.push(b'\n');
        }

        let result = if request.append.unwrap_or(false) {
            use std::io::Write;
            fs::OpenOptions::new()
                .create(true)
                .append(true)
                .open(&path)
                .and_then(|mut file| file.write_all(&bytes))
        } else {
            fs::write(&path, &bytes)
        };
        if let Err(err) = result {
            return file_error("write", &path, err);
        }

        ApiResponse::ok(
            "File written successfully",
            json!({
                "file": path.to_string_lossy(),
                "bytes_written": bytes.len(),
                "created": created
            }),
        )
    }

    pub async fn replace_file(&self, request: ReplaceFileRequest) -> ApiResponse {
        let path = match self.resolve_file("replace", &request.file) {
            Ok(path) => path,
            Err(response) => return response,
        };
        let content = match fs::read_to_string(&path) {
            Ok(content) => content,
            Err(err) => return file_error("replace", &path, err),
        };
        if !content.contains(&request.old_str) {
            return business_error("replace", &path, "old_str was not found", "not_found", false);
        }
        let replacement_count = content.matches(&request.old_str).count();
        let updated = if request.replace_all.unwrap_or(false) {
            content.replace(&request.old_str, &request.new_str)
        } else {
            content.replacen(&request.old_str, &request.new_str, 1)
        };
        if let Err(err) = fs::write(&path, updated) {
            return file_error("replace", &path, err);
        }
        ApiResponse::ok(
            "File replaced successfully",
            json!({
                "file": path.to_string_lossy(),
                "replacements": if request.replace_all.unwrap_or(false) { replacement_count } else { 1 }
            }),
        )
    }

    fn resolve_file(&self, operation: &'static str, input: &str) -> Result<PathBuf, ApiResponse> {
        let path = self
            .path_guard
            .resolve(input)
            .map_err(|err| path_error(operation, err))?;
        Ok(path)
    }
}
```

- [ ] **Step 5: Add response helpers**

Add:

```rust
impl ApiResponse {
    pub fn ok(message: impl Into<String>, data: Value) -> Self {
        Self {
            success: true,
            message: message.into(),
            data,
        }
    }

    pub fn fail(message: impl Into<String>, data: Value) -> Self {
        Self {
            success: false,
            message: message.into(),
            data,
        }
    }
}

struct SelectedLines {
    content: String,
    line_count: usize,
}

fn select_lines(content: &str, start_line: Option<usize>, end_line: Option<usize>) -> SelectedLines {
    if start_line.is_none() && end_line.is_none() {
        return SelectedLines {
            content: content.to_string(),
            line_count: content.lines().count(),
        };
    }
    let start = start_line.unwrap_or(0);
    let end = end_line.unwrap_or(usize::MAX);
    let lines: Vec<&str> = content
        .lines()
        .enumerate()
        .filter_map(|(index, line)| (index >= start && index < end).then_some(line))
        .collect();
    SelectedLines {
        content: lines.join("\n"),
        line_count: lines.len(),
    }
}

fn decode_content(content: &str, encoding: Option<&str>) -> Result<Vec<u8>, String> {
    match encoding.unwrap_or("utf-8").to_ascii_lowercase().as_str() {
        "utf-8" | "utf8" => Ok(content.as_bytes().to_vec()),
        "raw" => Ok(content.bytes().collect()),
        "base64" => base64::engine::general_purpose::STANDARD
            .decode(content)
            .map_err(|err| format!("base64 decode failed: {err}")),
        other => Err(format!("unsupported file encoding: {other}")),
    }
}

fn path_error(operation: &'static str, error: PathGuardError) -> ApiResponse {
    ApiResponse::fail(
        error.message.clone(),
        json!({
            "path": error.input,
            "operation": operation,
            "message": error.message,
            "error_type": error.error_type,
            "retryable": false
        }),
    )
}

fn business_error(
    operation: &'static str,
    path: &PathBuf,
    message: &str,
    error_type: &'static str,
    retryable: bool,
) -> ApiResponse {
    ApiResponse::fail(
        message,
        json!({
            "path": path.to_string_lossy(),
            "operation": operation,
            "message": message,
            "error_type": error_type,
            "retryable": retryable
        }),
    )
}

fn file_error(operation: &'static str, path: &PathBuf, err: std::io::Error) -> ApiResponse {
    let error_type = match err.kind() {
        std::io::ErrorKind::NotFound => "not_found",
        std::io::ErrorKind::PermissionDenied => "permission_denied",
        std::io::ErrorKind::AlreadyExists => "already_exists",
        _ => "io_error",
    };
    ApiResponse::fail(
        format!("Failed to {operation} file: {err}"),
        json!({
            "path": path.to_string_lossy(),
            "operation": operation,
            "message": format!("Failed to {operation} file: {err}"),
            "error_type": error_type,
            "retryable": false,
            "errno": err.raw_os_error(),
            "errno_name": errno_name(err.raw_os_error())
        }),
    )
}

fn errno_name(errno: Option<i32>) -> Option<&'static str> {
    match errno {
        Some(2) => Some("ENOENT"),
        Some(13) => Some("EACCES"),
        Some(17) => Some("EEXIST"),
        Some(28) => Some("ENOSPC"),
        _ => None,
    }
}
```

- [ ] **Step 6: Add list route and route handlers**

Add handlers to `file_api.rs`:

```rust
pub fn router<S>() -> Router<S>
where
    S: Clone + Send + Sync + 'static,
    FileService: FromState<S>,
{
    Router::new()
        .route("/v1/file/read", post(read_file::<S>))
        .route("/v1/file/write", post(write_file::<S>))
        .route("/v1/file/replace", post(replace_file::<S>))
        .route("/v1/file/list", post(list_dir::<S>))
}

pub trait FromState<S> {
    fn from_state(state: &S) -> Self;
}

async fn read_file<S>(State(state): State<S>, Json(request): Json<ReadFileRequest>) -> Json<ApiResponse>
where
    FileService: FromState<S>,
{
    Json(FileService::from_state(&state).read_file(request).await)
}

async fn write_file<S>(State(state): State<S>, Json(request): Json<WriteFileRequest>) -> Json<ApiResponse>
where
    FileService: FromState<S>,
{
    Json(FileService::from_state(&state).write_file(request).await)
}

async fn replace_file<S>(
    State(state): State<S>,
    Json(request): Json<ReplaceFileRequest>,
) -> Json<ApiResponse>
where
    FileService: FromState<S>,
{
    Json(FileService::from_state(&state).replace_file(request).await)
}

async fn list_dir<S>(State(state): State<S>, Json(request): Json<ListDirRequest>) -> Json<ApiResponse>
where
    FileService: FromState<S>,
{
    Json(FileService::from_state(&state).list_dir(request).await)
}
```

Implement `list_dir` with `walkdir`:

```rust
pub async fn list_dir(&self, request: ListDirRequest) -> ApiResponse {
    let path = match self.path_guard.resolve(&request.path) {
        Ok(path) => path,
        Err(err) => return path_error("list", err),
    };
    let metadata = match fs::metadata(&path) {
        Ok(metadata) => metadata,
        Err(err) => return file_error("list", &path, err),
    };
    if !metadata.is_dir() {
        return business_error("list", &path, "path is not a directory", "invalid_target", false);
    }

    let recursive = request.recursive.unwrap_or(false);
    let show_hidden = request.show_hidden.unwrap_or(false);
    let include_size = request.include_size.unwrap_or(true);
    let walker = if recursive {
        walkdir::WalkDir::new(&path).min_depth(1)
    } else {
        walkdir::WalkDir::new(&path).min_depth(1).max_depth(1)
    };
    let mut entries = Vec::new();
    for entry in walker.into_iter().filter_map(Result::ok) {
        let name = entry.file_name().to_string_lossy();
        if !show_hidden && name.starts_with('.') {
            continue;
        }
        let metadata = entry.metadata().ok();
        entries.push(json!({
            "path": entry.path().to_string_lossy(),
            "name": name,
            "is_dir": metadata.as_ref().is_some_and(|value| value.is_dir()),
            "size": if include_size { metadata.map(|value| value.len()) } else { None }
        }));
    }
    ApiResponse::ok("Directory listed successfully", json!({ "path": path, "entries": entries }))
}
```

- [ ] **Step 7: Wire FileService into AppState**

Modify `OpenClaw4j-Sandbox/src/main.rs`:

```rust
mod file_api;
mod path_guard;
```

Extend `AppState`:

```rust
#[derive(Clone)]
pub struct AppState {
    executor: Arc<SandboxExecutor>,
    file_service: file_api::FileService,
}
```

When constructing state:

```rust
let path_guard = path_guard::PathGuard::new(config.workspace_dir.clone());
path_guard.ensure_workspace()?;
let state = AppState {
    executor: Arc::new(SandboxExecutor::new(config.clone())),
    file_service: file_api::FileService::new(path_guard, config.file_read_limit_bytes),
};
```

Implement trait:

```rust
impl file_api::FromState<AppState> for file_api::FileService {
    fn from_state(state: &AppState) -> Self {
        state.file_service.clone()
    }
}
```

Merge router:

```rust
.merge(file_api::router::<AppState>())
```

- [ ] **Step 8: Verify Task 2 passes**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
cargo test file_api -- --nocapture
```

Expected: PASS.

- [ ] **Step 9: Commit Task 2**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j
git add OpenClaw4j-Sandbox/src/file_api.rs OpenClaw4j-Sandbox/src/main.rs
git commit -m "feat: add basic sandbox file APIs"
```

## Task 3: File Search, Glob, Grep, And Str Replace Editor

**Files:**
- Modify: `OpenClaw4j-Sandbox/src/file_api.rs`

**Interfaces:**
- Consumes: `FileService`
- Produces routes: `POST /v1/file/search`, `POST /v1/file/find`, `POST /v1/file/grep`, `POST /v1/file/glob`, `POST /v1/file/str_replace_editor`
- Produces request types: `SearchFileRequest`, `FindFileRequest`, `GrepRequest`, `GlobRequest`, `StrReplaceEditorRequest`

- [ ] **Step 1: Add failing search and editor tests**

Append tests to `file_api.rs` test module:

```rust
#[tokio::test]
async fn greps_directory_with_case_insensitive_pattern() {
    let (dir, service) = service();
    fs::create_dir_all(dir.path().join("src")).unwrap();
    fs::write(dir.path().join("src/app.rs"), "fn main() { println!(\"Marker\"); }").unwrap();

    let result = service
        .grep(GrepRequest {
            path: "src".to_string(),
            pattern: "marker".to_string(),
            include: Some(vec!["*.rs".to_string()]),
            exclude: None,
            case_insensitive: Some(true),
            max_results: Some(10),
        })
        .await;

    assert!(result.success);
    assert_eq!(result.data["matches"].as_array().unwrap().len(), 1);
}

#[tokio::test]
async fn str_replace_editor_requires_unique_match_by_default() {
    let (dir, service) = service();
    fs::write(dir.path().join("app.txt"), "one fish one fish").unwrap();

    let result = service
        .str_replace_editor(StrReplaceEditorRequest {
            command: "str_replace".to_string(),
            path: "app.txt".to_string(),
            file_text: None,
            old_str: Some("one".to_string()),
            new_str: Some("two".to_string()),
            insert_line: None,
            replace_mode: None,
        })
        .await;

    assert!(!result.success);
    assert_eq!(result.data["error_type"], Value::from("ambiguous_match"));
}

#[tokio::test]
async fn str_replace_editor_replaces_all_when_requested() {
    let (dir, service) = service();
    fs::write(dir.path().join("app.txt"), "one fish one fish").unwrap();

    let result = service
        .str_replace_editor(StrReplaceEditorRequest {
            command: "str_replace".to_string(),
            path: "app.txt".to_string(),
            file_text: None,
            old_str: Some("one".to_string()),
            new_str: Some("two".to_string()),
            insert_line: None,
            replace_mode: Some("ALL".to_string()),
        })
        .await;

    assert!(result.success);
    assert_eq!(fs::read_to_string(dir.path().join("app.txt")).unwrap(), "two fish two fish");
}
```

- [ ] **Step 2: Run failing tests**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
cargo test file_api::tests::greps file_api::tests::str_replace -- --nocapture
```

Expected: FAIL because request types and methods are missing.

- [ ] **Step 3: Add request models**

Add:

```rust
#[derive(Debug, Deserialize)]
pub struct SearchFileRequest {
    pub file: String,
    pub regex: String,
    #[serde(default)]
    pub max_results: Option<usize>,
}

#[derive(Debug, Deserialize)]
pub struct FindFileRequest {
    pub path: String,
    pub glob: String,
}

#[derive(Debug, Deserialize)]
pub struct GrepRequest {
    pub path: String,
    pub pattern: String,
    #[serde(default)]
    pub include: Option<Vec<String>>,
    #[serde(default)]
    pub exclude: Option<Vec<String>>,
    #[serde(default)]
    pub case_insensitive: Option<bool>,
    #[serde(default)]
    pub max_results: Option<usize>,
}

#[derive(Debug, Deserialize)]
pub struct GlobRequest {
    pub path: String,
    pub pattern: String,
    #[serde(default)]
    pub exclude: Option<Vec<String>>,
    #[serde(default)]
    pub include_hidden: Option<bool>,
    #[serde(default)]
    pub files_only: Option<bool>,
    #[serde(default)]
    pub include_metadata: Option<bool>,
    #[serde(default)]
    pub max_results: Option<usize>,
}

#[derive(Debug, Deserialize)]
pub struct StrReplaceEditorRequest {
    pub command: String,
    pub path: String,
    #[serde(default)]
    pub file_text: Option<String>,
    #[serde(default)]
    pub old_str: Option<String>,
    #[serde(default)]
    pub new_str: Option<String>,
    #[serde(default)]
    pub insert_line: Option<usize>,
    #[serde(default)]
    pub replace_mode: Option<String>,
}
```

- [ ] **Step 4: Implement grep and search methods**

Add service methods:

```rust
pub async fn search_file(&self, request: SearchFileRequest) -> ApiResponse {
    let path = match self.resolve_file("search", &request.file) {
        Ok(path) => path,
        Err(response) => return response,
    };
    let content = match fs::read_to_string(&path) {
        Ok(content) => content,
        Err(err) => return file_error("search", &path, err),
    };
    let regex = match regex::Regex::new(&request.regex) {
        Ok(regex) => regex,
        Err(err) => {
            return business_error("search", &path, &format!("invalid regex: {err}"), "invalid_pattern", false);
        }
    };
    let mut matches = Vec::new();
    for (line_index, line) in content.lines().enumerate() {
        for found in regex.find_iter(line) {
            matches.push(json!({
                "file": path.to_string_lossy(),
                "line": line_index + 1,
                "column": found.start() + 1,
                "match": found.as_str()
            }));
            if matches.len() >= request.max_results.unwrap_or(100) {
                return ApiResponse::ok("File searched successfully", json!({ "matches": matches }));
            }
        }
    }
    ApiResponse::ok("File searched successfully", json!({ "matches": matches }))
}

pub async fn grep(&self, request: GrepRequest) -> ApiResponse {
    let root = match self.path_guard.resolve(&request.path) {
        Ok(path) => path,
        Err(err) => return path_error("grep", err),
    };
    let pattern = if request.case_insensitive.unwrap_or(false) {
        format!("(?i){}", request.pattern)
    } else {
        request.pattern.clone()
    };
    let regex = match regex::Regex::new(&pattern) {
        Ok(regex) => regex,
        Err(err) => {
            return business_error("grep", &root, &format!("invalid regex: {err}"), "invalid_pattern", false);
        }
    };
    let include = build_globset(request.include.as_deref());
    let exclude = build_globset(request.exclude.as_deref());
    let mut matches = Vec::new();
    for entry in walkdir::WalkDir::new(&root).into_iter().filter_map(Result::ok) {
        if !entry.file_type().is_file() {
            continue;
        }
        let relative = entry.path().strip_prefix(&root).unwrap_or(entry.path());
        if include.as_ref().is_some_and(|set| !set.is_match(relative)) {
            continue;
        }
        if exclude.as_ref().is_some_and(|set| set.is_match(relative)) {
            continue;
        }
        let content = match fs::read_to_string(entry.path()) {
            Ok(content) => content,
            Err(_) => continue,
        };
        for (line_index, line) in content.lines().enumerate() {
            if let Some(found) = regex.find(line) {
                matches.push(json!({
                    "file": entry.path().to_string_lossy(),
                    "line": line_index + 1,
                    "column": found.start() + 1,
                    "text": line
                }));
                if matches.len() >= request.max_results.unwrap_or(100) {
                    return ApiResponse::ok("Directory grepped successfully", json!({ "matches": matches }));
                }
            }
        }
    }
    ApiResponse::ok("Directory grepped successfully", json!({ "matches": matches }))
}
```

- [ ] **Step 5: Implement find, glob, and str_replace_editor**

Add:

```rust
pub async fn find(&self, request: FindFileRequest) -> ApiResponse {
    let root = match self.path_guard.resolve(&request.path) {
        Ok(path) => path,
        Err(err) => return path_error("find", err),
    };
    let globset = build_globset(Some(&[request.glob])).unwrap();
    let files: Vec<Value> = walkdir::WalkDir::new(&root)
        .into_iter()
        .filter_map(Result::ok)
        .filter(|entry| entry.file_type().is_file())
        .filter_map(|entry| {
            let relative = entry.path().strip_prefix(&root).ok()?;
            globset.is_match(relative).then(|| json!(entry.path().to_string_lossy()))
        })
        .collect();
    ApiResponse::ok("Files found successfully", json!({ "files": files }))
}

pub async fn glob(&self, request: GlobRequest) -> ApiResponse {
    let root = match self.path_guard.resolve(&request.path) {
        Ok(path) => path,
        Err(err) => return path_error("glob", err),
    };
    let include = match build_globset(Some(&[request.pattern])) {
        Some(set) => set,
        None => return business_error("glob", &root, "invalid glob pattern", "invalid_pattern", false),
    };
    let exclude = build_globset(request.exclude.as_deref());
    let mut files = Vec::new();
    for entry in walkdir::WalkDir::new(&root).into_iter().filter_map(Result::ok) {
        let relative = match entry.path().strip_prefix(&root) {
            Ok(relative) => relative,
            Err(_) => continue,
        };
        if !request.include_hidden.unwrap_or(false)
            && relative.components().any(|part| part.as_os_str().to_string_lossy().starts_with('.'))
        {
            continue;
        }
        if request.files_only.unwrap_or(false) && !entry.file_type().is_file() {
            continue;
        }
        if !include.is_match(relative) {
            continue;
        }
        if exclude.as_ref().is_some_and(|set| set.is_match(relative)) {
            continue;
        }
        let value = if request.include_metadata.unwrap_or(false) {
            let metadata = entry.metadata().ok();
            json!({
                "path": entry.path().to_string_lossy(),
                "is_dir": metadata.as_ref().is_some_and(|value| value.is_dir()),
                "size": metadata.map(|value| value.len())
            })
        } else {
            json!(entry.path().to_string_lossy())
        };
        files.push(value);
        if files.len() >= request.max_results.unwrap_or(200) {
            break;
        }
    }
    ApiResponse::ok("Files globbed successfully", json!({ "files": files }))
}

pub async fn str_replace_editor(&self, request: StrReplaceEditorRequest) -> ApiResponse {
    match request.command.as_str() {
        "view" => {
            self.read_file(ReadFileRequest {
                file: request.path,
                start_line: None,
                end_line: None,
                sudo: None,
            })
            .await
        }
        "create" => self.create_editor_file(request).await,
        "str_replace" => self.editor_replace(request).await,
        "insert" => self.editor_insert(request).await,
        "undo_edit" => {
            let path = self
                .path_guard
                .resolve(&request.path)
                .unwrap_or_else(|_| PathBuf::from(&request.path));
            business_error("str_replace_editor", &path, "undo_edit is not supported", "unsupported_operation", false)
        }
        _ => {
            let path = self
                .path_guard
                .resolve(&request.path)
                .unwrap_or_else(|_| PathBuf::from(&request.path));
            business_error("str_replace_editor", &path, "unknown editor command", "invalid_request", false)
        }
    }
}
```

Implement helper functions `create_editor_file`, `editor_replace`, `editor_insert`, and `build_globset` in the same module. `editor_replace` must count matches, require exactly one match when `replace_mode` is absent, and support `ALL`, `FIRST`, `LAST`.

- [ ] **Step 6: Register new routes**

In `file_api::router`, add:

```rust
.route("/v1/file/search", post(search_file::<S>))
.route("/v1/file/find", post(find_file::<S>))
.route("/v1/file/grep", post(grep::<S>))
.route("/v1/file/glob", post(glob::<S>))
.route("/v1/file/str_replace_editor", post(str_replace_editor::<S>))
```

Add handlers mirroring the existing `read_file` handler style.

- [ ] **Step 7: Verify Task 3 passes**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
cargo test file_api -- --nocapture
```

Expected: PASS.

- [ ] **Step 8: Commit Task 3**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j
git add OpenClaw4j-Sandbox/src/file_api.rs
git commit -m "feat: add sandbox file search and editor APIs"
```

## Task 4: Bash Pipe API

**Files:**
- Create: `OpenClaw4j-Sandbox/src/bash_api.rs`
- Modify: `OpenClaw4j-Sandbox/src/main.rs`

**Interfaces:**
- Consumes: `PathGuard`
- Produces: `BashService::new(path_guard: PathGuard, default_timeout_ms: u64, hard_timeout_ms: u64, output_limit_bytes: usize) -> Self`
- Produces: `BashService::exec`, `output`, `write`, `kill`, `sessions`, `create_session`, `close_session`
- Produces routes: `/v1/bash/exec`, `/v1/bash/output`, `/v1/bash/write`, `/v1/bash/kill`, `/v1/bash/sessions`, `/v1/bash/sessions/create`, `/v1/bash/sessions/{session_id}/close`

- [ ] **Step 1: Add failing bash service tests**

Create `OpenClaw4j-Sandbox/src/bash_api.rs` with tests:

```rust
#[cfg(test)]
mod tests {
    use super::{BashExecRequest, BashService};
    use crate::path_guard::PathGuard;
    use tempfile::tempdir;

    fn service() -> (tempfile::TempDir, BashService) {
        let dir = tempdir().unwrap();
        let service = BashService::new(PathGuard::new(dir.path().to_path_buf()), 5_000, 30_000, 65_536);
        (dir, service)
    }

    #[tokio::test]
    async fn exec_captures_stdout_stderr_and_exit_code() {
        let (_dir, service) = service();
        let response = service
            .exec(BashExecRequest {
                command: "printf 'out'; printf 'err' >&2; exit 7".to_string(),
                session_id: None,
                exec_dir: None,
                env: None,
                async_mode: Some(false),
                timeout: Some(5.0),
                hard_timeout: Some(30.0),
                max_output_length: None,
            })
            .await;

        assert!(response.success);
        assert_eq!(response.data["status"], "completed");
        assert_eq!(response.data["stdout"], "out");
        assert_eq!(response.data["stderr"], "err");
        assert_eq!(response.data["exit_code"], 7);
    }

    #[tokio::test]
    async fn async_exec_can_be_polled_until_completion() {
        let (_dir, service) = service();
        let started = service
            .exec(BashExecRequest {
                command: "sleep 0.1; echo done".to_string(),
                session_id: None,
                exec_dir: None,
                env: None,
                async_mode: Some(true),
                timeout: None,
                hard_timeout: Some(30.0),
                max_output_length: None,
            })
            .await;

        assert_eq!(started.data["status"], "running");
        let session_id = started.data["session_id"].as_str().unwrap().to_string();
        let command_id = started.data["command_id"].as_str().unwrap().to_string();

        let output = service
            .output(super::BashOutputRequest {
                session_id,
                command_id: Some(command_id),
                offset: Some(0),
                stderr_offset: Some(0),
                wait: Some(true),
                wait_timeout: Some(2.0),
            })
            .await;

        assert!(output.success);
        assert!(output.data["stdout"].as_str().unwrap().contains("done"));
    }
}
```

- [ ] **Step 2: Run failing bash tests**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
cargo test bash_api -- --nocapture
```

Expected: FAIL because `BashService` and request models are missing.

- [ ] **Step 3: Implement bash models and state**

Add models:

```rust
use crate::file_api::ApiResponse;
use crate::path_guard::PathGuard;
use axum::extract::{Path as AxumPath, State};
use axum::routing::{get, post};
use axum::{Json, Router};
use serde::Deserialize;
use serde_json::json;
use std::collections::HashMap;
use std::process::Stdio;
use std::sync::Arc;
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::process::{Child, ChildStdin, Command};
use tokio::sync::Mutex;
use tokio::time::{timeout, Duration, Instant};
use uuid::Uuid;

#[derive(Clone)]
pub struct BashService {
    path_guard: PathGuard,
    default_timeout_ms: u64,
    hard_timeout_ms: u64,
    output_limit_bytes: usize,
    sessions: Arc<Mutex<HashMap<String, BashSession>>>,
}

struct BashSession {
    session_id: String,
    exec_dir: String,
    status: String,
    commands: HashMap<String, Arc<Mutex<BashCommandState>>>,
    current_command_id: Option<String>,
}

struct BashCommandState {
    command_id: String,
    command: String,
    status: String,
    stdout: Vec<u8>,
    stderr: Vec<u8>,
    exit_code: Option<i32>,
    stdin: Option<ChildStdin>,
    child: Option<Child>,
}
```

Add request structs:

```rust
#[derive(Debug, Deserialize)]
pub struct BashExecRequest {
    pub command: String,
    #[serde(default)]
    pub session_id: Option<String>,
    #[serde(default)]
    pub exec_dir: Option<String>,
    #[serde(default)]
    pub env: Option<HashMap<String, Option<String>>>,
    #[serde(default)]
    pub async_mode: Option<bool>,
    #[serde(default)]
    pub timeout: Option<f64>,
    #[serde(default)]
    pub hard_timeout: Option<f64>,
    #[serde(default)]
    pub max_output_length: Option<usize>,
}

#[derive(Debug, Deserialize)]
pub struct BashOutputRequest {
    pub session_id: String,
    #[serde(default)]
    pub command_id: Option<String>,
    #[serde(default)]
    pub offset: Option<usize>,
    #[serde(default)]
    pub stderr_offset: Option<usize>,
    #[serde(default)]
    pub wait: Option<bool>,
    #[serde(default)]
    pub wait_timeout: Option<f64>,
}
```

- [ ] **Step 4: Implement exec with subprocess pipes**

Implement `BashService::exec` so it:

```rust
pub async fn exec(&self, request: BashExecRequest) -> ApiResponse {
    if request.command.trim().is_empty() {
        return ApiResponse::fail("command cannot be empty", json!({ "error_type": "invalid_request" }));
    }
    let session_id = request
        .session_id
        .clone()
        .filter(|value| !value.trim().is_empty())
        .unwrap_or_else(|| Uuid::new_v4().to_string());
    let exec_dir = match self.resolve_exec_dir(request.exec_dir.as_deref()).await {
        Ok(path) => path,
        Err(response) => return response,
    };
    let command_id = Uuid::new_v4().to_string();
    let mut command = shell_command(&request.command);
    command.current_dir(&exec_dir);
    command.stdin(Stdio::piped()).stdout(Stdio::piped()).stderr(Stdio::piped());
    if let Some(env) = &request.env {
        for (key, value) in env {
            if let Some(value) = value {
                command.env(key, value);
            }
        }
    }
    let mut child = match command.spawn() {
        Ok(child) => child,
        Err(err) => {
            return ApiResponse::fail(
                format!("failed to start command: {err}"),
                json!({ "error_type": "spawn_error" }),
            );
        }
    };
    let stdin = child.stdin.take();
    let stdout = child.stdout.take().unwrap();
    let stderr = child.stderr.take().unwrap();
    let state = Arc::new(Mutex::new(BashCommandState {
        command_id: command_id.clone(),
        command: request.command.clone(),
        status: "running".to_string(),
        stdout: Vec::new(),
        stderr: Vec::new(),
        exit_code: None,
        stdin,
        child: Some(child),
    }));
    self.register_command(&session_id, &exec_dir, &command_id, state.clone()).await;
    spawn_reader(state.clone(), stdout, true);
    spawn_reader(state.clone(), stderr, false);
    spawn_waiter(state.clone(), request.hard_timeout, self.hard_timeout_ms);

    if request.async_mode.unwrap_or(false) {
        return self.command_response(&session_id, &command_id, request.max_output_length).await;
    }
    let wait_ms = seconds_to_ms(request.timeout).unwrap_or(self.default_timeout_ms);
    let start = Instant::now();
    while start.elapsed() < Duration::from_millis(wait_ms) {
        if state.lock().await.status != "running" {
            break;
        }
        tokio::time::sleep(Duration::from_millis(20)).await;
    }
    self.command_response(&session_id, &command_id, request.max_output_length).await
}
```

Implement `shell_command` with Linux bash and Windows PowerShell fallback:

```rust
fn shell_command(command: &str) -> Command {
    if cfg!(windows) {
        let mut cmd = Command::new("powershell");
        cmd.arg("-NoLogo").arg("-NoProfile").arg("-Command").arg(command);
        cmd
    } else {
        let mut cmd = Command::new("/bin/bash");
        cmd.arg("-lc").arg(command);
        cmd
    }
}
```

- [ ] **Step 5: Implement output, write, kill, and sessions**

Implement:

```rust
pub async fn output(&self, request: BashOutputRequest) -> ApiResponse;
pub async fn write(&self, request: BashWriteRequest) -> ApiResponse;
pub async fn kill(&self, request: BashKillRequest) -> ApiResponse;
pub async fn sessions(&self) -> ApiResponse;
pub async fn create_session(&self, request: BashCreateSessionRequest) -> ApiResponse;
pub async fn close_session(&self, session_id: String) -> ApiResponse;
```

`output` must slice stdout/stderr from offsets and return:

```json
{
  "session_id": "...",
  "command_id": "...",
  "stdout": "...",
  "stderr": "...",
  "offset": 12,
  "stderr_offset": 0,
  "command": {
    "status": "completed",
    "exit_code": 0
  }
}
```

`write` writes to `BashCommandState.stdin`; `kill` calls `child.start_kill()` and sets `status="killed"` if still running.

- [ ] **Step 6: Register bash routes and state**

Add `mod bash_api;` in `main.rs`, extend `AppState`:

```rust
bash_service: bash_api::BashService,
```

Construct service with config values and the same `PathGuard`:

```rust
bash_service: bash_api::BashService::new(
    path_guard.clone(),
    config.bash_default_timeout_ms,
    config.bash_hard_timeout_ms,
    config.bash_output_limit_bytes,
),
```

Merge:

```rust
.merge(bash_api::router::<AppState>())
```

- [ ] **Step 7: Verify Task 4 passes**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
cargo test bash_api -- --nocapture
```

Expected: PASS.

- [ ] **Step 8: Commit Task 4**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j
git add OpenClaw4j-Sandbox/src/bash_api.rs OpenClaw4j-Sandbox/src/main.rs
git commit -m "feat: add sandbox bash pipe APIs"
```

## Task 5: MCP REST Wrapper And JSON-RPC Entry

**Files:**
- Create: `OpenClaw4j-Sandbox/src/mcp_api.rs`
- Modify: `OpenClaw4j-Sandbox/src/main.rs`

**Interfaces:**
- Consumes: `FileService`
- Consumes: `BashService`
- Produces routes: `POST /mcp`, `GET /v1/mcp/servers`, `GET /v1/mcp/{server_name}/tools`, `POST /v1/mcp/{server_name}/tools/{tool_name}`
- Produces tools: `file_read`, `file_write`, `file_list`, `file_replace`, `file_search`, `sandbox_execute_bash`

- [ ] **Step 1: Add failing MCP tests**

Create `OpenClaw4j-Sandbox/src/mcp_api.rs` with tests:

```rust
#[cfg(test)]
mod tests {
    use super::{call_builtin_tool, list_builtin_tools, JsonRpcRequest};
    use crate::bash_api::BashService;
    use crate::file_api::FileService;
    use crate::path_guard::PathGuard;
    use serde_json::json;
    use tempfile::tempdir;

    fn services() -> (tempfile::TempDir, FileService, BashService) {
        let dir = tempdir().unwrap();
        let guard = PathGuard::new(dir.path().to_path_buf());
        (
            dir,
            FileService::new(guard.clone(), 1024 * 1024),
            BashService::new(guard, 5_000, 30_000, 65_536),
        )
    }

    #[tokio::test]
    async fn lists_builtin_tools() {
        let tools = list_builtin_tools();
        assert!(tools.iter().any(|tool| tool["name"] == "file_read"));
        assert!(tools.iter().any(|tool| tool["name"] == "sandbox_execute_bash"));
    }

    #[tokio::test]
    async fn calls_file_write_tool() {
        let (_dir, file_service, bash_service) = services();
        let result = call_builtin_tool(
            &file_service,
            &bash_service,
            "file_write",
            json!({ "file": "mcp.txt", "content": "hello" }),
        )
        .await;
        assert!(!result.is_error);
        assert!(result.content[0].text.contains("File written successfully"));
    }

    #[test]
    fn parses_json_rpc_request() {
        let request: JsonRpcRequest = serde_json::from_value(json!({
            "jsonrpc": "2.0",
            "id": 1,
            "method": "tools/list"
        }))
        .unwrap();
        assert_eq!(request.method, "tools/list");
    }
}
```

- [ ] **Step 2: Run failing MCP tests**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
cargo test mcp_api -- --nocapture
```

Expected: FAIL because MCP module is not implemented.

- [ ] **Step 3: Implement MCP models and tool registry**

Add:

```rust
use crate::bash_api::{BashExecRequest, BashService};
use crate::file_api::{
    FileService, ListDirRequest, ReadFileRequest, ReplaceFileRequest, SearchFileRequest,
    WriteFileRequest,
};
use axum::extract::{Path, Query, State};
use axum::routing::{get, post};
use axum::{Json, Router};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};

#[derive(Debug, Deserialize)]
pub struct JsonRpcRequest {
    #[serde(default)]
    pub jsonrpc: Option<String>,
    #[serde(default)]
    pub id: Option<Value>,
    pub method: String,
    #[serde(default)]
    pub params: Option<Value>,
}

#[derive(Debug, Serialize)]
pub struct McpToolResult {
    pub content: Vec<TextContent>,
    #[serde(rename = "isError")]
    pub is_error: bool,
}

#[derive(Debug, Serialize)]
pub struct TextContent {
    #[serde(rename = "type")]
    pub content_type: &'static str,
    pub text: String,
}

pub fn list_builtin_tools() -> Vec<Value> {
    vec![
        tool("file_read", "Read a text file from the sandbox workspace", json!({"type":"object","required":["file"],"properties":{"file":{"type":"string"},"start_line":{"type":"integer"},"end_line":{"type":"integer"}}})),
        tool("file_write", "Write a file in the sandbox workspace", json!({"type":"object","required":["file","content"],"properties":{"file":{"type":"string"},"content":{"type":"string"},"append":{"type":"boolean"}}})),
        tool("file_list", "List a sandbox workspace directory", json!({"type":"object","required":["path"],"properties":{"path":{"type":"string"},"recursive":{"type":"boolean"}}})),
        tool("file_replace", "Replace text in a sandbox workspace file", json!({"type":"object","required":["file","old_str","new_str"],"properties":{"file":{"type":"string"},"old_str":{"type":"string"},"new_str":{"type":"string"},"replace_all":{"type":"boolean"}}})),
        tool("file_search", "Search a file with a regex", json!({"type":"object","required":["file","regex"],"properties":{"file":{"type":"string"},"regex":{"type":"string"}}})),
        tool("sandbox_execute_bash", "Execute a bash command in the sandbox workspace", json!({"type":"object","required":["command"],"properties":{"command":{"type":"string"},"timeout":{"type":"number"},"async_mode":{"type":"boolean"}}})),
    ]
}

fn tool(name: &str, description: &str, input_schema: Value) -> Value {
    json!({
        "name": name,
        "description": description,
        "inputSchema": input_schema
    })
}
```

- [ ] **Step 4: Implement built-in tool dispatch**

Add:

```rust
pub async fn call_builtin_tool(
    file_service: &FileService,
    bash_service: &BashService,
    tool_name: &str,
    arguments: Value,
) -> McpToolResult {
    let api_response = match tool_name {
        "file_read" => match serde_json::from_value::<ReadFileRequest>(arguments) {
            Ok(request) => file_service.read_file(request).await,
            Err(err) => argument_error(err),
        },
        "file_write" => match serde_json::from_value::<WriteFileRequest>(arguments) {
            Ok(request) => file_service.write_file(request).await,
            Err(err) => argument_error(err),
        },
        "file_list" => match serde_json::from_value::<ListDirRequest>(arguments) {
            Ok(request) => file_service.list_dir(request).await,
            Err(err) => argument_error(err),
        },
        "file_replace" => match serde_json::from_value::<ReplaceFileRequest>(arguments) {
            Ok(request) => file_service.replace_file(request).await,
            Err(err) => argument_error(err),
        },
        "file_search" => match serde_json::from_value::<SearchFileRequest>(arguments) {
            Ok(request) => file_service.search_file(request).await,
            Err(err) => argument_error(err),
        },
        "sandbox_execute_bash" => match serde_json::from_value::<BashExecRequest>(arguments) {
            Ok(request) => bash_service.exec(request).await,
            Err(err) => argument_error(err),
        },
        _ => {
            return McpToolResult {
                content: vec![TextContent {
                    content_type: "text",
                    text: json!({"error_type":"unknown_tool","message":"unknown MCP tool"}).to_string(),
                }],
                is_error: true,
            };
        }
    };
    McpToolResult {
        content: vec![TextContent {
            content_type: "text",
            text: serde_json::to_string(&api_response).unwrap_or_else(|_| "{}".to_string()),
        }],
        is_error: !api_response.success,
    }
}

fn argument_error(err: serde_json::Error) -> crate::file_api::ApiResponse {
    crate::file_api::ApiResponse {
        success: false,
        message: format!("invalid tool arguments: {err}"),
        data: json!({
            "error_type": "invalid_request",
            "message": format!("invalid tool arguments: {err}")
        }),
    }
}
```

- [ ] **Step 5: Implement JSON-RPC and REST handlers**

Implement:

```rust
pub fn router<S>() -> Router<S>
where
    S: Clone + Send + Sync + 'static,
    McpServices: FromState<S>,
{
    Router::new()
        .route("/mcp", post(json_rpc::<S>))
        .route("/v1/mcp/servers", get(list_servers))
        .route("/v1/mcp/:server_name/tools", get(list_tools))
        .route("/v1/mcp/:server_name/tools/:tool_name", post(call_tool::<S>))
}
```

JSON-RPC behavior:

```rust
match request.method.as_str() {
    "initialize" => result with protocolVersion and capabilities,
    "tools/list" => result with tools,
    "tools/call" => extract params.name and params.arguments, call built-in tool,
    _ => error code -32601,
}
```

REST wrapper behavior:

- `GET /v1/mcp/servers` returns `{ "success": true, "data": ["sandbox"] }`.
- `GET /v1/mcp/sandbox/tools` returns `{ "success": true, "data": { "tools": [...] } }`.
- unknown server returns `success=false`.
- `POST /v1/mcp/sandbox/tools/{tool}` dispatches built-in tool.

- [ ] **Step 6: Wire MCP into AppState**

In `main.rs`, add:

```rust
mod mcp_api;
```

Expose services through a `McpServices` state adapter:

```rust
impl mcp_api::FromState<AppState> for mcp_api::McpServices {
    fn from_state(state: &AppState) -> Self {
        mcp_api::McpServices {
            file_service: state.file_service.clone(),
            bash_service: state.bash_service.clone(),
        }
    }
}
```

Merge:

```rust
.merge(mcp_api::router::<AppState>())
```

- [ ] **Step 7: Verify Task 5 passes**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
cargo test mcp_api -- --nocapture
```

Expected: PASS.

- [ ] **Step 8: Commit Task 5**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j
git add OpenClaw4j-Sandbox/src/mcp_api.rs OpenClaw4j-Sandbox/src/main.rs
git commit -m "feat: add sandbox MCP tool APIs"
```

## Task 6: Route Integration, Documentation, And Final Verification

**Files:**
- Modify: `OpenClaw4j-Sandbox/src/main.rs`
- Modify: `OpenClaw4j-Sandbox/README.md`

**Interfaces:**
- Consumes: file, bash, MCP routes.
- Produces: documented examples for `/v1/file/read`, `/v1/bash/exec`, `/mcp`.

- [ ] **Step 1: Add route smoke tests**

If current project keeps handler tests inside modules, add to `main.rs`:

```rust
#[cfg(test)]
mod route_tests {
    use super::*;

    #[test]
    fn app_state_is_cloneable_for_shared_routes() {
        fn assert_clone<T: Clone>() {}
        assert_clone::<AppState>();
    }
}
```

This is a small compile-time guard that route state remains cloneable for Axum.

- [ ] **Step 2: Update README with concise examples**

Add sections to `OpenClaw4j-Sandbox/README.md`:

````markdown
## Agent 兼容接口

服务额外提供一组轻量 AIO 风格接口，供 Agent 读写 workspace 文件、运行命令和通过 MCP 调用工具。

默认 workspace：

```text
OPENCLAW_SANDBOX_WORKSPACE_DIR=/tmp/openclaw4j-workspace
```

读取文件：

```powershell
$body = @{ file = 'notes.txt' } | ConvertTo-Json
Invoke-RestMethod -Uri 'http://127.0.0.1:9010/v1/file/read' -Method Post -ContentType 'application/json' -Body $body
```

执行 Bash：

```powershell
$body = @{ command = 'pwd && ls -la'; timeout = 30 } | ConvertTo-Json
Invoke-RestMethod -Uri 'http://127.0.0.1:9010/v1/bash/exec' -Method Post -ContentType 'application/json' -Body $body
```

MCP 工具列表：

```powershell
$body = @{ jsonrpc = '2.0'; id = 1; method = 'tools/list' } | ConvertTo-Json
Invoke-RestMethod -Uri 'http://127.0.0.1:9010/mcp' -Method Post -ContentType 'application/json' -Body $body
```

首轮不包含 upload/download、file watch、浏览器、Jupyter、外部 MCP Hub 聚合和 sudo。
````

- [ ] **Step 3: Run full Rust test suite**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
cargo test
```

Expected: PASS.

- [ ] **Step 4: Optional local HTTP smoke test**

If a local Rust binary can run without port conflict:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
$env:OPENCLAW_SANDBOX_BIND='127.0.0.1:9010'
$env:OPENCLAW_SANDBOX_WORKSPACE_DIR="$PWD\output\workspace"
cargo run
```

Then in another PowerShell:

```powershell
Invoke-RestMethod -Uri 'http://127.0.0.1:9010/health'
```

Expected: `{ "status": "UP" }`. Stop the server after the smoke test.

- [ ] **Step 5: Commit Task 6**

Run:

```powershell
cd D:\IDEA_project\OpenClaw4j
git add OpenClaw4j-Sandbox/src/main.rs OpenClaw4j-Sandbox/README.md
git commit -m "docs: document sandbox agent-compatible APIs"
```

## Final Verification

- [ ] Run:

```powershell
cd D:\IDEA_project\OpenClaw4j\OpenClaw4j-Sandbox
cargo test
```

Expected: PASS.

- [ ] From root, inspect tracked changes:

```powershell
cd D:\IDEA_project\OpenClaw4j
git status --short
```

Expected: only intentional sandbox files remain modified or untracked. Existing unrelated backend/frontend dirty files may remain; do not revert them.

## Self-Review Notes

- Spec coverage: path guard, file API, bash API, MCP API, route registration, README, and `cargo test` verification are covered.
- Scope control: upload/download/watch/browser/Jupyter/external MCP Hub/sudo are explicitly excluded from implementation tasks.
- Type consistency: `PathGuard`, `FileService`, `BashService`, and MCP state adapters are the shared interfaces between tasks.
