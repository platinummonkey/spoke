import { useState, useEffect, useCallback } from 'react';

export interface Bookmark {
  id: number;
  module_name: string;
  version: string;
  entity_path?: string;
  entity_type?: string;
  notes?: string;
  tags?: string[];
  created_at: string;
  updated_at: string;
}

const STORAGE_KEY = 'spoke_bookmarks';

export const useBookmarks = () => {
  const [bookmarks, setBookmarks] = useState<Bookmark[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Load bookmarks from API and localStorage
  const loadBookmarks = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      // Load from localStorage first (immediate)
      const localBookmarks = loadFromLocalStorage();
      setBookmarks(localBookmarks);

      // Then fetch from API (for authenticated users in future)
      try {
        const response = await fetch('/api/v2/bookmarks');
        if (response.ok) {
          const data = await response.json();
          const remoteBookmarks = data.bookmarks || [];
          setBookmarks(mergeBookmarks(localBookmarks, remoteBookmarks));
        }
      } catch (apiError) {
        // API not available, use localStorage only
        console.log('API not available, using localStorage only');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load bookmarks');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadBookmarks();
  }, [loadBookmarks]);

  // Check if a module/entity is bookmarked
  const isBookmarked = useCallback((moduleName: string, version: string, entityPath?: string) => {
    return bookmarks.some(
      b => b.module_name === moduleName &&
           b.version === version &&
           (entityPath === undefined || b.entity_path === entityPath)
    );
  }, [bookmarks]);

  // Create a new bookmark
  const createBookmark = useCallback(async (bookmark: Omit<Bookmark, 'id' | 'created_at' | 'updated_at'>) => {
    try {
      // Try API first
      try {
        const response = await fetch('/api/v2/bookmarks', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(bookmark),
        });

        if (response.ok) {
          const created = await response.json();
          setBookmarks(prev => [created, ...prev]);
          return created;
        }
      } catch (apiError) {
        // API not available, fallback to localStorage
      }

      // Fallback to localStorage
      const newBookmark: Bookmark = {
        ...bookmark,
        id: Date.now(),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };

      const updated = [newBookmark, ...bookmarks];
      setBookmarks(updated);
      saveToLocalStorage(updated);
      return newBookmark;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create bookmark');
      throw err;
    }
  }, [bookmarks]);

  // Update a bookmark
  const updateBookmark = useCallback(async (id: number, updates: Partial<Bookmark>) => {
    try {
      // Try API first
      try {
        const response = await fetch(`/api/v2/bookmarks/${id}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(updates),
        });

        if (response.ok) {
          const updated = await response.json();
          setBookmarks(prev => prev.map(b => b.id === id ? updated : b));
          return updated;
        }
      } catch (apiError) {
        // API not available, fallback to localStorage
      }

      // Fallback to localStorage
      const updated = bookmarks.map(b =>
        b.id === id ? { ...b, ...updates, updated_at: new Date().toISOString() } : b
      );
      setBookmarks(updated);
      saveToLocalStorage(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update bookmark');
      throw err;
    }
  }, [bookmarks]);

  // Delete a bookmark
  const deleteBookmark = useCallback(async (id: number) => {
    try {
      // Try API first
      try {
        const response = await fetch(`/api/v2/bookmarks/${id}`, {
          method: 'DELETE',
        });

        if (response.ok) {
          setBookmarks(prev => prev.filter(b => b.id !== id));
          return;
        }
      } catch (apiError) {
        // API not available, fallback to localStorage
      }

      // Fallback to localStorage
      const updated = bookmarks.filter(b => b.id !== id);
      setBookmarks(updated);
      saveToLocalStorage(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete bookmark');
      throw err;
    }
  }, [bookmarks]);

  // Toggle bookmark (add if not exists, remove if exists)
  const toggleBookmark = useCallback(async (moduleName: string, version: string, entityPath?: string) => {
    const existing = bookmarks.find(
      b => b.module_name === moduleName &&
           b.version === version &&
           (entityPath === undefined || b.entity_path === entityPath)
    );

    if (existing) {
      await deleteBookmark(existing.id);
    } else {
      await createBookmark({
        module_name: moduleName,
        version,
        entity_path: entityPath,
      });
    }
  }, [bookmarks, createBookmark, deleteBookmark]);

  return {
    bookmarks,
    loading,
    error,
    isBookmarked,
    createBookmark,
    updateBookmark,
    deleteBookmark,
    toggleBookmark,
    refresh: loadBookmarks,
  };
};

// localStorage helpers
function loadFromLocalStorage(): Bookmark[] {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored ? JSON.parse(stored) : [];
  } catch {
    return [];
  }
}

function saveToLocalStorage(bookmarks: Bookmark[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(bookmarks));
  } catch (err) {
    console.error('Failed to save to localStorage:', err);
  }
}

function mergeBookmarks(local: Bookmark[], remote: Bookmark[]): Bookmark[] {
  // Prefer remote and add local that don't exist remotely
  const remoteIds = new Set(remote.map(b => b.id));
  const localOnly = local.filter(b => !remoteIds.has(b.id));
  return [...remote, ...localOnly];
}
