/* eslint-env jest */

const fs = require('fs');
const path = require('path');

describe('skill detail layout', () => {
  it('sets the default active tab so overview content is rendered', () => {
    const source = fs.readFileSync(path.join(__dirname, 'Detail.tsx'), 'utf8');

    expect(source).toContain('activeTab="overview"');
  });

  it('shows a publish action for unpublished skills and calls the publish API', () => {
    const detailSource = fs.readFileSync(
      path.join(__dirname, 'Detail.tsx'),
      'utf8',
    );
    const serviceSource = fs.readFileSync(
      path.join(__dirname, '../../services/skill.ts'),
      'utf8',
    );

    expect(serviceSource).toContain('export async function publishSkill');
    expect(serviceSource).toContain('/publish`');
    expect(detailSource).toContain('publishSkill(detail.skill_code)');
    expect(detailSource).toContain('detail.status !== SkillStatus.PUBLISHED');
    expect(detailSource).toContain("dm: '发布'");
  });
});
