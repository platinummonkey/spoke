import { useState, useEffect, useCallback } from 'react';

export interface SavedSearch {
  id: number;
  name: string;
  query: string;
  filters?: Record<string, any>;
  description?: string;
  is_shared?: boolean;
  created_at: string;
  updated_at: string;
}

const STORAGE_KEY = 'spoke_saved_searches';

export const useSavedSearches = () => {
  const [searches, setSearches] = useState<SavedSearch[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Load saved searches from API and localStorage
  const loadSearches = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      // Load from localStorage first (immediate)
      const localSearches = loadFromLocalStorage();
      setSearches(localSearches);

      // Then fetch from API (for authenticated users in future)
      try {
        const response = await fetch('/api/v2/saved-searches');
        if (response.ok) {
          const data = await response.json();
          // Merge local and remote, preferring remote
          const remoteSearches = data.saved_searches || [];
          setSearches(mergeSearches(localSearches, remoteSearches));
        }
      } catch (apiError) {
        // API not available, use localStorage only
        console.log('API not available, using localStorage only');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load saved searches');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadSearches();
  }, [loadSearches]);

  // Create a new saved search
  const createSearch = useCallback(async (search: Omit<SavedSearch, 'id' | 'created_at' | 'updated_at'>) => {
    try {
      // Try API first
      try {
        const response = await fetch('/api/v2/saved-searches', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(search),
        });

        if (response.ok) {
          const created = await response.json();
          setSearches(prev => [created, ...prev]);
          return created;
        }
      } catch (apiError) {
        // API not available, fallback to localStorage
      }

      // Fallback to localStorage
      const newSearch: SavedSearch = {
        ...search,
        id: Date.now(), // Use timestamp as ID for local storage
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };

      const updated = [newSearch, ...searches];
      setSearches(updated);
      saveToLocalStorage(updated);
      return newSearch;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create saved search');
      throw err;
    }
  }, [searches]);

  // Update a saved search
  const updateSearch = useCallback(async (id: number, updates: Partial<SavedSearch>) => {
    try {
      // Try API first
      try {
        const response = await fetch(`/api/v2/saved-searches/${id}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(updates),
        });

        if (response.ok) {
          const updated = await response.json();
          setSearches(prev => prev.map(s => s.id === id ? updated : s));
          return updated;
        }
      } catch (apiError) {
        // API not available, fallback to localStorage
      }

      // Fallback to localStorage
      const updated = searches.map(s =>
        s.id === id ? { ...s, ...updates, updated_at: new Date().toISOString() } : s
      );
      setSearches(updated);
      saveToLocalStorage(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update saved search');
      throw err;
    }
  }, [searches]);

  // Delete a saved search
  const deleteSearch = useCallback(async (id: number) => {
    try {
      // Try API first
      try {
        const response = await fetch(`/api/v2/saved-searches/${id}`, {
          method: 'DELETE',
        });

        if (response.ok) {
          setSearches(prev => prev.filter(s => s.id !== id));
          return;
        }
      } catch (apiError) {
        // API not available, fallback to localStorage
      }

      // Fallback to localStorage
      const updated = searches.filter(s => s.id !== id);
      setSearches(updated);
      saveToLocalStorage(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete saved search');
      throw err;
    }
  }, [searches]);

  return {
    searches,
    loading,
    error,
    createSearch,
    updateSearch,
    deleteSearch,
    refresh: loadSearches,
  };
};

// localStorage helpers
function loadFromLocalStorage(): SavedSearch[] {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored ? JSON.parse(stored) : [];
  } catch {
    return [];
  }
}

function saveToLocalStorage(searches: SavedSearch[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(searches));
  } catch (err) {
    console.error('Failed to save to localStorage:', err);
  }
}

function mergeSearches(local: SavedSearch[], remote: SavedSearch[]): SavedSearch[] {
  // For now, prefer remote and add local that don't exist remotely
  // In future, implement proper sync logic
  const remoteIds = new Set(remote.map(s => s.id));
  const localOnly = local.filter(s => !remoteIds.has(s.id));
  return [...remote, ...localOnly];
}
