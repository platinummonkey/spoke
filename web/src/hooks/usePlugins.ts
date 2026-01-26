import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import pluginService from '../services/pluginService';
import {
  PluginListRequest,
  PluginReviewRequest,
} from '../types/plugin';

// Query Keys
export const pluginKeys = {
  all: ['plugins'] as const,
  lists: () => [...pluginKeys.all, 'list'] as const,
  list: (params?: PluginListRequest) => [...pluginKeys.lists(), params] as const,
  details: () => [...pluginKeys.all, 'detail'] as const,
  detail: (id: string) => [...pluginKeys.details(), id] as const,
  versions: (id: string) => [...pluginKeys.detail(id), 'versions'] as const,
  reviews: (id: string) => [...pluginKeys.detail(id), 'reviews'] as const,
  stats: (id: string) => [...pluginKeys.detail(id), 'stats'] as const,
  trending: () => [...pluginKeys.all, 'trending'] as const,
};

// List Plugins Hook
export function usePlugins(params?: PluginListRequest) {
  return useQuery({
    queryKey: pluginKeys.list(params),
    queryFn: () => pluginService.listPlugins(params),
    staleTime: 60 * 1000, // 1 minute
  });
}

// Search Plugins Hook
export function useSearchPlugins(query: string, params?: PluginListRequest) {
  return useQuery({
    queryKey: [...pluginKeys.lists(), 'search', query, params],
    queryFn: () => pluginService.searchPlugins(query, params),
    enabled: query.length > 0,
    staleTime: 30 * 1000, // 30 seconds
  });
}

// Trending Plugins Hook
export function useTrendingPlugins(limit: number = 10) {
  return useQuery({
    queryKey: [...pluginKeys.trending(), limit],
    queryFn: () => pluginService.getTrendingPlugins(limit),
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

// Get Plugin Details Hook
export function usePlugin(id: string) {
  return useQuery({
    queryKey: pluginKeys.detail(id),
    queryFn: () => pluginService.getPlugin(id),
    enabled: !!id,
  });
}

// List Plugin Versions Hook
export function usePluginVersions(pluginId: string) {
  return useQuery({
    queryKey: pluginKeys.versions(pluginId),
    queryFn: () => pluginService.listVersions(pluginId),
    enabled: !!pluginId,
  });
}

// List Plugin Reviews Hook
export function usePluginReviews(pluginId: string, limit: number = 20, offset: number = 0) {
  return useQuery({
    queryKey: [...pluginKeys.reviews(pluginId), limit, offset],
    queryFn: () => pluginService.listReviews(pluginId, limit, offset),
    enabled: !!pluginId,
  });
}

// Get Plugin Stats Hook
export function usePluginStats(pluginId: string, days: number = 30) {
  return useQuery({
    queryKey: [...pluginKeys.stats(pluginId), days],
    queryFn: () => pluginService.getPluginStats(pluginId, days),
    enabled: !!pluginId,
  });
}

// Create Review Mutation
export function useCreateReview(pluginId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (review: PluginReviewRequest) =>
      pluginService.createReview(pluginId, review),
    onSuccess: () => {
      // Invalidate and refetch reviews and plugin data
      queryClient.invalidateQueries({ queryKey: pluginKeys.reviews(pluginId) });
      queryClient.invalidateQueries({ queryKey: pluginKeys.detail(pluginId) });
    },
  });
}

// Record Installation Mutation
export function useRecordInstallation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ pluginId, version }: { pluginId: string; version: string }) =>
      pluginService.recordInstallation(pluginId, version),
    onSuccess: (_, variables) => {
      // Invalidate plugin stats
      queryClient.invalidateQueries({ queryKey: pluginKeys.stats(variables.pluginId) });
      queryClient.invalidateQueries({ queryKey: pluginKeys.detail(variables.pluginId) });
    },
  });
}

// Record Uninstallation Mutation
export function useRecordUninstallation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (pluginId: string) => pluginService.recordUninstallation(pluginId),
    onSuccess: (_, pluginId) => {
      // Invalidate plugin stats
      queryClient.invalidateQueries({ queryKey: pluginKeys.stats(pluginId) });
    },
  });
}
