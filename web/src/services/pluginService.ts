import axios, { AxiosInstance } from 'axios';
import {
  Plugin,
  PluginListRequest,
  PluginListResponse,
  PluginVersion,
  PluginReview,
  PluginReviewRequest,
  PluginStats,
} from '../types/plugin';

class PluginService {
  private api: AxiosInstance;

  constructor(baseURL: string = '/api/v1') {
    this.api = axios.create({
      baseURL,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }

  // Plugin Discovery

  async listPlugins(params?: PluginListRequest): Promise<PluginListResponse> {
    const response = await this.api.get<PluginListResponse>('/plugins', { params });
    return response.data;
  }

  async searchPlugins(query: string, params?: PluginListRequest): Promise<PluginListResponse> {
    const response = await this.api.get<PluginListResponse>('/plugins/search', {
      params: { q: query, ...params },
    });
    return response.data;
  }

  async getTrendingPlugins(limit: number = 10): Promise<PluginListResponse> {
    const response = await this.api.get<PluginListResponse>('/plugins/trending', {
      params: { limit },
    });
    return response.data;
  }

  async getPlugin(id: string): Promise<Plugin> {
    const response = await this.api.get<Plugin>(`/plugins/${id}`);
    return response.data;
  }

  // Plugin Versions

  async listVersions(pluginId: string): Promise<PluginVersion[]> {
    const response = await this.api.get<PluginVersion[]>(`/plugins/${pluginId}/versions`);
    return response.data;
  }

  async getVersion(pluginId: string, version: string): Promise<PluginVersion> {
    const response = await this.api.get<PluginVersion>(
      `/plugins/${pluginId}/versions/${version}`
    );
    return response.data;
  }

  async downloadPlugin(pluginId: string, version: string): Promise<string> {
    const response = await this.api.get(`/plugins/${pluginId}/versions/${version}/download`, {
      maxRedirects: 0,
      validateStatus: (status) => status === 302,
    });
    return response.headers.location;
  }

  // Plugin Reviews

  async listReviews(
    pluginId: string,
    limit: number = 20,
    offset: number = 0
  ): Promise<PluginReview[]> {
    const response = await this.api.get<PluginReview[]>(`/plugins/${pluginId}/reviews`, {
      params: { limit, offset },
    });
    return response.data;
  }

  async createReview(pluginId: string, review: PluginReviewRequest): Promise<void> {
    await this.api.post(`/plugins/${pluginId}/reviews`, review);
  }

  // Installation Tracking

  async recordInstallation(pluginId: string, version: string): Promise<void> {
    await this.api.post(`/plugins/${pluginId}/install`, { version });
  }

  async recordUninstallation(pluginId: string): Promise<void> {
    await this.api.post(`/plugins/${pluginId}/uninstall`);
  }

  // Plugin Stats

  async getPluginStats(pluginId: string, days: number = 30): Promise<PluginStats> {
    const response = await this.api.get<PluginStats>(`/plugins/${pluginId}/stats`, {
      params: { days },
    });
    return response.data;
  }

  // Auth

  setAuthToken(token: string): void {
    this.api.defaults.headers.common['Authorization'] = `Bearer ${token}`;
  }

  setUserId(userId: string): void {
    this.api.defaults.headers.common['X-User-ID'] = userId;
  }
}

export const pluginService = new PluginService();
export default pluginService;
