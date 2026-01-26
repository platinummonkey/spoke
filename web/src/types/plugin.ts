// Plugin Marketplace Types

export interface Plugin {
  id: string;
  name: string;
  description: string;
  author: string;
  license: string;
  homepage: string;
  repository: string;
  type: PluginType;
  security_level: SecurityLevel;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  verified_at?: string;
  verified_by?: string;
  download_count: number;
  latest_version?: string;
  avg_rating?: number;
  review_count?: number;
}

export type PluginType = 'language' | 'validator' | 'generator' | 'runner' | 'transform';

export type SecurityLevel = 'official' | 'verified' | 'community';

export interface PluginVersion {
  id: number;
  plugin_id: string;
  version: string;
  api_version: string;
  manifest_url: string;
  download_url: string;
  checksum: string;
  size_bytes: number;
  downloads: number;
  created_at: string;
}

export interface PluginReview {
  id: number;
  plugin_id: string;
  user_id: string;
  user_name?: string;
  rating: number;
  review: string;
  created_at: string;
  updated_at: string;
}

export interface PluginListRequest {
  type?: PluginType;
  security_level?: SecurityLevel;
  tags?: string[];
  search?: string;
  limit?: number;
  offset?: number;
  sort_by?: 'downloads' | 'rating' | 'created_at';
  sort_order?: 'asc' | 'desc';
}

export interface PluginListResponse {
  plugins: Plugin[];
  total: number;
  limit: number;
  offset: number;
}

export interface PluginReviewRequest {
  rating: number;
  review: string;
}

export interface PluginStats {
  plugin_id: string;
  total_downloads: number;
  total_installations: number;
  active_installations: number;
  avg_rating: number;
  review_count: number;
  daily_stats?: DailyStat[];
  top_versions?: VersionDownload[];
}

export interface DailyStat {
  date: string;
  downloads: number;
  installations: number;
  uninstallations: number;
  active_installations: number;
}

export interface VersionDownload {
  version: string;
  downloads: number;
}
