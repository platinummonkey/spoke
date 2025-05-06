import { useState, useEffect } from 'react';
import { Module } from '../types';
import { getModule } from '../api/client';

export const useModule = (name: string) => {
  const [module, setModule] = useState<Module | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchModule = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await getModule(name);
      setModule(data);
    } catch (err) {
      console.error('Error fetching module:', err);
      setError(err instanceof Error ? err : new Error('Failed to fetch module'));
      setModule(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchModule();
  }, [name]);

  return { module, loading, error, retry: fetchModule };
}; 