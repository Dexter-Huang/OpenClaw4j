/* eslint-env jest */

const {
  normalizeAssistantContentForDisplay,
  preserveMarkdownLineBreaks,
} = require('./messageContent');

describe('normalizeAssistantContentForDisplay', () => {
  it('unwraps MCP text content that contains tool execution output', () => {
    const content = JSON.stringify({
      content: [
        {
          type: 'text',
          text: JSON.stringify({
            status: 'ok',
            stdout: '=== Python random script ===\n===== Random numbers =====\n1-100: 25\n',
            stderr: null,
            exit_code: null,
          }),
        },
      ],
      is_error: false,
    });

    expect(normalizeAssistantContentForDisplay(content)).toBe(
      '=== Python random script ===  \n===== Random numbers =====  \n1-100: 25  \n',
    );
  });

  it('keeps normal assistant text unchanged', () => {
    expect(normalizeAssistantContentForDisplay('normal reply')).toBe('normal reply');
  });

  it('formats execution output line breaks for markdown rendering', () => {
    expect(preserveMarkdownLineBreaks('line 1\nline 2\r\nline 3')).toBe(
      'line 1  \nline 2  \nline 3',
    );
  });
});
