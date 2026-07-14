import { request } from '@/request';
import {
  ITool,
  IToolListParams,
  IToolListResponse,
  ICreateToolRequest,
  IUpdateToolRequest,
  IToolResponse,
} from '@/types/tool';

/**
 * Tool service class
 */
export class ToolService {
  /**
   * Create a new tool
   * @param data Tool data
   * @returns Created tool
   */
  static async createTool(data: ICreateToolRequest): Promise<IToolResponse> {
    const response = await request<IToolResponse>('/console/v1/tools', {
      method: 'POST',
      data,
    });
    return response.data;
  }

  /**
   * Update an existing tool
   * @param id Tool ID
   * @param data Updated tool data
   * @returns Updated tool
   */
  static async updateTool(id: number, data: IUpdateToolRequest): Promise<IToolResponse> {
    const response = await request<IToolResponse>(`/console/v1/tools/${id}`, {
      method: 'PUT',
      data,
    });
    return response.data;
  }

  /**
   * Delete a tool
   * @param id Tool ID
   */
  static async deleteTool(id: number): Promise<void> {
    await request<void>(`/console/v1/tools/${id}`, {
      method: 'DELETE',
    });
  }

  /**
   * Get tool by ID
   * @param id Tool ID
   * @returns Tool details
   */
  static async getTool(id: number): Promise<IToolResponse> {
    const response = await request<IToolResponse>(`/console/v1/tools/${id}`);
    return response.data;
  }

  /**
   * Get all tools for current workspace
   * @returns List of tools
   */
  static async getTools(): Promise<ITool[]> {
    const response = await request<ITool[]>('/console/v1/tools');
    return response.data;
  }

  /**
   * Get tools with pagination
   * @param params Query parameters
   * @returns Paginated list of tools
   */
  static async getToolsByPage(params: IToolListParams): Promise<IToolListResponse> {
    const response = await request<IToolListResponse>('/console/v1/tools/page', {
      method: 'GET',
      params,
    });
    return response.data;
  }

  /**
   * Search tools by name
   * @param name Search term
   * @returns List of matching tools
   */
  static async searchTools(name: string): Promise<ITool[]> {
    const response = await request<ITool[]>('/console/v1/tools/search', {
      method: 'GET',
      params: { name },
    });
    return response.data;
  }

  /**
   * Get tools by plugin ID
   * @param pluginId Plugin ID
   * @returns List of tools for the plugin
   */
  static async getToolsByPlugin(pluginId: string): Promise<ITool[]> {
    const response = await request<ITool[]>(`/console/v1/tools/plugin/${pluginId}`);
    return response.data;
  }

  /**
   * Enable or disable a tool
   * @param id Tool ID
   * @param enabled Enable status
   */
  static async setToolEnabled(id: number, enabled: boolean): Promise<void> {
    await request<void>(`/console/v1/tools/${id}/enabled`, {
      method: 'PATCH',
      params: { enabled },
    });
  }
}
