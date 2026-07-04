const TOOL_PANEL_COLLAPSED_HEIGHT = 64;
const TOOL_PANEL_MAX_HEIGHT = 128;
const TOOL_PANEL_LINE_HEIGHT = 20;
const TOOL_PANEL_CHROME_HEIGHT = 28;

function countRenderableLines(value) {
  if (typeof value !== 'string' || value.length === 0) {
    return 1;
  }

  try {
    return JSON.stringify(JSON.parse(value), null, 2).split('\n').length;
  } catch (error) {
    return value.split(/\r\n|\r|\n/).length;
  }
}

function getToolPanelExpandedHeight(value) {
  const estimatedHeight =
    TOOL_PANEL_CHROME_HEIGHT + countRenderableLines(value) * TOOL_PANEL_LINE_HEIGHT;

  return Math.min(
    Math.max(estimatedHeight, TOOL_PANEL_COLLAPSED_HEIGHT),
    TOOL_PANEL_MAX_HEIGHT,
  );
}

module.exports = {
  TOOL_PANEL_MAX_HEIGHT,
  getToolPanelExpandedHeight,
};
