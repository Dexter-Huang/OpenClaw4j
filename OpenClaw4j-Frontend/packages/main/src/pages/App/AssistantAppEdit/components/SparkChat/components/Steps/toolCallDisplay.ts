export const READ_SKILL_FILE_TOOL_NAME = 'read_skill_file';

export type ToolCallDisplayType = 'plugin' | 'skill';

export const getToolCallDisplayType = (
  toolName?: string,
): ToolCallDisplayType => {
  return toolName === READ_SKILL_FILE_TOOL_NAME ? 'skill' : 'plugin';
};
