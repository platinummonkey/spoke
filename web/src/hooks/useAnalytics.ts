import { useQuery } from '@tanstack/react-query';
import axios from 'axios';

// Types
export interface OverviewResponse {
  total_modules: number;
  total_versions: number;
  total_downloads_24h: number;
  total_downloads_7d: number;
  total_downloads_30d: number;
  active_users_24h: number;
  active_users_7d: number;
  top_language: string;
  avg_compilation_ms: number;
  cache_hit_rate: number;
}

export interface TimeSeriesPoint {
  date: string;
  value: number;
}

export interface VersionStats {
  version: string;
  downloads: number;
}

export interface ModuleStatsResponse {
  module_name: string;
  total_views: number;
  total_downloads: number;
  unique_users: number;
  downloads_by_day: TimeSeriesPoint[];
  downloads_by_language: Record<string, number>;
  popular_versions: VersionStats[];
  avg_compilation_time_ms: number;
  compilation_success_rate: number;
  last_downloaded_at?: string;
}

export interface PopularModule {
  module_name: string;
  total_downloads: number;
  total_views: number;
  active_days: number;
  avg_daily_downloads: number;
}

export interface TrendingModule {
  module_name: string;
  current_downloads: number;
  previous_downloads: number;
  growth_rate: number;
}

export interface ModuleHealth {
  module_name: string;
  version: string;
  health_score: number;
  complexity_score: number;
  maintainability_index: number;
  unused_fields: string[];
  deprecated_field_count: number;
  breaking_changes_30d: number;
  dependents_count: number;
  recommendations: string[];
}

// API Base URL
const API_BASE = '/api/v2/analytics';

// Hooks

/**
 * Fetch analytics overview KPIs
 */
export const useAnalyticsOverview = () => {
  return useQuery<OverviewResponse>({
    queryKey: ['analytics', 'overview'],
    queryFn: async () => {
      const { data } = await axios.get<OverviewResponse>(`${API_BASE}/overview`);
      return data;
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes (formerly cacheTime)
  });
};

/**
 * Fetch popular modules
 */
export const usePopularModules = (period: string = '30d', limit: number = 100) => {
  return useQuery<PopularModule[]>({
    queryKey: ['analytics', 'modules', 'popular', period, limit],
    queryFn: async () => {
      const { data } = await axios.get<PopularModule[]>(
        `${API_BASE}/modules/popular`,
        { params: { period, limit } }
      );
      return data;
    },
    staleTime: 10 * 60 * 1000, // 10 minutes
  });
};

/**
 * Fetch trending modules
 */
export const useTrendingModules = (limit: number = 50) => {
  return useQuery<TrendingModule[]>({
    queryKey: ['analytics', 'modules', 'trending', limit],
    queryFn: async () => {
      const { data } = await axios.get<TrendingModule[]>(
        `${API_BASE}/modules/trending`,
        { params: { limit } }
      );
      return data;
    },
    staleTime: 10 * 60 * 1000, // 10 minutes
  });
};

/**
 * Fetch module statistics
 */
export const useModuleStats = (moduleName: string, period: string = '30d') => {
  return useQuery<ModuleStatsResponse>({
    queryKey: ['analytics', 'module', moduleName, 'stats', period],
    queryFn: async () => {
      const { data } = await axios.get<ModuleStatsResponse>(
        `${API_BASE}/modules/${moduleName}/stats`,
        { params: { period } }
      );
      return data;
    },
    staleTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!moduleName,
  });
};

/**
 * Fetch module health assessment
 */
export const useModuleHealth = (moduleName: string, version?: string) => {
  return useQuery<ModuleHealth>({
    queryKey: ['analytics', 'module', moduleName, 'health', version],
    queryFn: async () => {
      const params = version ? { version } : undefined;
      const { data } = await axios.get<ModuleHealth>(
        `${API_BASE}/modules/${moduleName}/health`,
        { params }
      );
      return data;
    },
    staleTime: 15 * 60 * 1000, // 15 minutes (health changes slowly)
    enabled: !!moduleName,
  });
};
