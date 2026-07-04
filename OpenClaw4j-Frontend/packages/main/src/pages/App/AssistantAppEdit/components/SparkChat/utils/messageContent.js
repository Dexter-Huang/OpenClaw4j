function parseJsonObject(value) {
  if (typeof value !== 'string') {
    return null;
  }

  try {
    const parsed = JSON.parse(value);
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? parsed
      : null;
  } catch (error) {
    return null;
  }
}

function joinMcpTextParts(content) {
  if (!Array.isArray(content)) {
    return null;
  }

  const textParts = content
    .filter((item) => item?.type === 'text' && typeof item.text === 'string')
    .map((item) => item.text);

  return textParts.length ? textParts.join('\n') : null;
}

function preserveMarkdownLineBreaks(value) {
  return value.replace(/\r\n|\r|\n/g, '  \n');
}

function extractToolExecutionOutput(value) {
  const parsed = parseJsonObject(value);
  if (!parsed) {
    return null;
  }

  const hasExecutionShape =
    Object.prototype.hasOwnProperty.call(parsed, 'stdout') ||
    Object.prototype.hasOwnProperty.call(parsed, 'stderr') ||
    Object.prototype.hasOwnProperty.call(parsed, 'exit_code');

  if (!hasExecutionShape) {
    return null;
  }

  const stdout = typeof parsed.stdout === 'string' ? parsed.stdout : '';
  const stderr = typeof parsed.stderr === 'string' ? parsed.stderr : '';

  if (stdout && stderr) {
    return preserveMarkdownLineBreaks(
      `${stdout}${stdout.endsWith('\n') ? '' : '\n'}${stderr}`,
    );
  }

  const output = stdout || stderr;
  return output ? preserveMarkdownLineBreaks(output) : null;
}

function normalizeAssistantContentForDisplay(content) {
  if (typeof content !== 'string') {
    return content;
  }

  const parsed = parseJsonObject(content);
  if (!parsed || !Array.isArray(parsed.content)) {
    return content;
  }

  const textContent = joinMcpTextParts(parsed.content);
  if (!textContent) {
    return content;
  }

  return extractToolExecutionOutput(textContent) || textContent;
}

module.exports = {
  normalizeAssistantContentForDisplay,
  preserveMarkdownLineBreaks,
};
