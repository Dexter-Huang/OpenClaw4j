/* eslint-env jest */

const fs = require('fs');
const path = require('path');

describe('skill selector query', () => {
  it('does not restrict the drawer list to published skills by default', () => {
    const source = fs.readFileSync(path.join(__dirname, 'index.tsx'), 'utf8');

    expect(source).not.toContain('status: SkillStatus.PUBLISHED');
  });
});
