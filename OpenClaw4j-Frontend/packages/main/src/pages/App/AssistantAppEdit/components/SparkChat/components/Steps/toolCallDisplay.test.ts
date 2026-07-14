import { getToolCallDisplayType } from './toolCallDisplay';

describe('getToolCallDisplayType', () => {
  it('treats read_skill_file as a skill call', () => {
    expect(getToolCallDisplayType('read_skill_file')).toBe('skill');
  });

  it('keeps regular tool calls displayed as plugin calls', () => {
    expect(getToolCallDisplayType('query_weather')).toBe('plugin');
  });
});
