/* eslint-env jest */

const {
  TOOL_PANEL_MAX_HEIGHT,
  getToolPanelExpandedHeight,
} = require('./panelLayout');

describe('getToolPanelExpandedHeight', () => {
  it('shrinks short parameter blocks below the maximum height', () => {
    expect(getToolPanelExpandedHeight('{"language":"python"}')).toBeLessThan(
      TOOL_PANEL_MAX_HEIGHT,
    );
  });

  it('caps long parameter blocks at the maximum height', () => {
    const value = Array.from({ length: 20 }, (_, index) => `line ${index}`).join('\n');

    expect(getToolPanelExpandedHeight(value)).toBe(TOOL_PANEL_MAX_HEIGHT);
  });
});
