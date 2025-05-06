import { useState, useEffect, useCallback } from 'react';
import { Module } from '../types';
import { getModules, getModule } from '../api/client';

// Simple in-memory cache
const cache = new Map<string, { data: any; timestamp: number }>();
const CACHE_DURATION = 5 * 60 * 1000; // 5 minutes

const getCachedData = <T>(key: string): T | null => {
  const cached = cache.get(key);
  if (cached && Date.now() - cached.timestamp < CACHE_DURATION) {
    return cached.data as T;
  }
  return null;
};

const setCachedData = <T>(key: string, data: T) => {
  cache.set(key, { data, timestamp: Date.now() });
};

export const useModules = (retryCount = 3) => {
  const [modules, setModules] = useState<Module[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [retryAttempt, setRetryAttempt] = useState(0);

  const fetchModules = useCallback(async () => {
    try {
      setLoading(true);
      // Check cache first
      const cachedModules = getCachedData<Module[]>('modules');
      if (cachedModules) {
        setModules(cachedModules);
        setError(null);
        setLoading(false);
        return;
      }

      const data = await getModules();
      setModules(data);
      setError(null);
      // Cache the result
      setCachedData('modules', data);
    } catch (err) {
      console.error('Error fetching modules:', err);
      if (retryAttempt < retryCount) {
        setRetryAttempt(prev => prev + 1);
        // Exponential backoff
        setTimeout(() => {
          fetchModules();
        }, Math.min(1000 * Math.pow(2, retryAttempt), 10000));
      } else {
        setError('Failed to fetch modules. Please try again.');
      }
    } finally {
      setLoading(false);
    }
  }, [retryAttempt, retryCount]);

  useEffect(() => {
    fetchModules();
  }, [fetchModules]);

  const retry = useCallback(() => {
    setRetryAttempt(0);
    setError(null);
    fetchModules();
  }, [fetchModules]);

  return { modules, loading, error, retry };
};

export const useModule = (moduleName: string, retryCount = 3) => {
  const [module, setModule] = useState<Module | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [retryAttempt, setRetryAttempt] = useState(0);

  const fetchModule = useCallback(async () => {
    if (!moduleName) return;

    try {
      setLoading(true);
      // Check cache first
      const cachedModule = getCachedData<Module>(`module-${moduleName}`);
      if (cachedModule) {
        setModule(cachedModule);
        setError(null);
        setLoading(false);
        return;
      }

      const data = await getModule(moduleName);
      setModule(data);
      setError(null);
      // Cache the result
      setCachedData(`module-${moduleName}`, data);
    } catch (err) {
      console.error('Error fetching module:', err);
      if (retryAttempt < retryCount) {
        setRetryAttempt(prev => prev + 1);
        // Exponential backoff
        setTimeout(() => {
          fetchModule();
        }, Math.min(1000 * Math.pow(2, retryAttempt), 10000));
      } else {
        setError('Failed to fetch module. Please try again.');
      }
    } finally {
      setLoading(false);
    }
  }, [moduleName, retryAttempt, retryCount]);

  useEffect(() => {
    fetchModule();
  }, [fetchModule]);

  const retry = useCallback(() => {
    setRetryAttempt(0);
    setError(null);
    fetchModule();
  }, [fetchModule]);

  return { module, loading, error, retry };
}; 