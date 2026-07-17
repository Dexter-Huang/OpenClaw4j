package wrapper

import (
	"path/filepath"
	"strings"
)

func BuildRunner(language Language, workDir string) string {
	switch language {
	case Python:
		return buildPythonRunner(workDir)
	case JavaScript:
		return buildJavaScriptRunner(workDir)
	default:
		return ""
	}
}

func slashPath(path string) string {
	return strings.ReplaceAll(filepath.ToSlash(path), `\`, `\\`)
}

func buildPythonRunner(workDir string) string {
	paramsPath := slashPath(filepath.Join(workDir, "params.json"))
	userPath := slashPath(filepath.Join(workDir, "user.py"))
	resultPath := slashPath(filepath.Join(workDir, "result.json"))
	return `
import inspect
import json
import runpy
import sys
import traceback

params_path = r"` + paramsPath + `"
user_path = r"` + userPath + `"
result_path = r"` + resultPath + `"

try:
    with open(params_path, "r", encoding="utf-8") as f:
        params = json.load(f)
    namespace = runpy.run_path(user_path)
    main_fn = namespace.get("main")
    if not callable(main_fn):
        raise RuntimeError("main function is required")
    signature = inspect.signature(main_fn)
    if len(signature.parameters) == 0:
        result = main_fn()
    else:
        result = main_fn(params)
    with open(result_path, "w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False)
except Exception:
    traceback.print_exc(file=sys.stderr)
    sys.exit(1)
`
}

func buildJavaScriptRunner(workDir string) string {
	paramsPath := slashPath(filepath.Join(workDir, "params.json"))
	userPath := slashPath(filepath.Join(workDir, "user.js"))
	resultPath := slashPath(filepath.Join(workDir, "result.json"))
	return `
const fs = require('fs');
const vm = require('vm');

const paramsPath = "` + paramsPath + `";
const userPath = "` + userPath + `";
const resultPath = "` + resultPath + `";

(async () => {
  const params = JSON.parse(fs.readFileSync(paramsPath, 'utf8'));
  const sandbox = {
    console,
    module: { exports: {} },
    exports: {},
    require,
    process: { env: process.env },
    setTimeout,
    clearTimeout,
    setInterval,
    clearInterval,
  };
  sandbox.globalThis = sandbox;
  const context = vm.createContext(sandbox);
  const userCode = fs.readFileSync(userPath, 'utf8');
  vm.runInContext(userCode, context, { filename: userPath, timeout: 1000 });
  const mainFn = context.main || context.module.exports.main || context.exports.main;
  if (typeof mainFn !== 'function') {
    throw new Error('main function is required');
  }
  const result = mainFn.length === 0 ? await mainFn() : await mainFn(params);
  fs.writeFileSync(resultPath, JSON.stringify(result), 'utf8');
})().catch((err) => {
  console.error(err && err.stack ? err.stack : String(err));
  process.exit(1);
});
`
}
