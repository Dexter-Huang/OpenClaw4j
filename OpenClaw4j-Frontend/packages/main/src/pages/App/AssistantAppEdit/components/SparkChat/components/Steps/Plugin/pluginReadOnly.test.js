/* eslint-env jest */

const fs = require('fs');
const path = require('path');

const source = fs.readFileSync(path.join(__dirname, 'index.tsx'), 'utf8');

const getCodeBlockTagForValue = (valueExpression) =>
  source
    .match(/<CodeBlock[\s\S]*?\/>/g)
    ?.find((tag) => tag.includes(`value={${valueExpression}}`)) || '';

describe('plugin parameter panels', () => {
  it('renders JSON input parameters in a read-only code block', () => {
    expect(getCodeBlockTagForValue('params.arguments')).toContain('readOnly');
  });

  it('renders JSON output parameters in a read-only code block', () => {
    expect(getCodeBlockTagForValue('params.output')).toContain('readOnly');
  });
});
