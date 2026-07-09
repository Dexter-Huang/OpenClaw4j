const fs = require('fs');
const path = require('path');

const packageName = '@spark-ai/chat';
const packageVersion = '1.1.4';
const targetPath = path.join(
  __dirname,
  '..',
  'node_modules',
  '@spark-ai',
  'chat',
  'dist',
  'sender',
  'index.js',
);

const before =
  'var nodes = Array.isArray(props.prefix) ? _toConsumableArray(props.prefix) : [props.prefix];';
const after = 'var nodes = React.Children.toArray(props.prefix);';

if (!fs.existsSync(targetPath)) {
  console.warn(`[patch] ${packageName} sender not found, skipped.`);
  process.exit(0);
}

const packageJsonPath = path.join(
  __dirname,
  '..',
  'node_modules',
  '@spark-ai',
  'chat',
  'package.json',
);

try {
  const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
  if (packageJson.version !== packageVersion) {
    console.warn(
      `[patch] ${packageName} version is ${packageJson.version}, expected ${packageVersion}; skipped.`,
    );
    process.exit(0);
  }
} catch (error) {
  console.warn(`[patch] failed to read ${packageName} package.json: ${error}`);
  process.exit(0);
}

const source = fs.readFileSync(targetPath, 'utf8');

if (source.includes(after)) {
  console.log(`[patch] ${packageName} sender prefix key patch already applied.`);
  process.exit(0);
}

if (!source.includes(before)) {
  console.warn(`[patch] ${packageName} sender patch target changed, skipped.`);
  process.exit(0);
}

// @spark-ai/chat 1.1.4 的 Sender 会把外部 prefix 和内部 zoom 按钮拼成数组。
// 外部 prefix 常见是 Fragment，直接入数组会在 React dev 模式触发 key 告警；
// React.Children.toArray 会保留渲染结果，并为外部节点补齐稳定 key。
fs.writeFileSync(targetPath, source.replace(before, after), 'utf8');
console.log(`[patch] ${packageName} sender prefix key patch applied.`);
