import { useState, useEffect, useCallback, useRef } from 'react';

export interface SearchFilter {
  type: 'entity' | 'field-type' | 'module' | 'version' | 'has-comment';
  value: string;
  display: string;
}

export interface EnhancedSearchResult {
  // Entity identification
  id: number;
  entity_type: string;
  entity_name: string;
  full_path: string;
  parent_path?: string;

  // Module/version info
  module_name: string;
  version: string;

  // Proto file context
  proto_file_path?: string;
  line_number?: number;

  // Content
  description?: string;
  comments?: string;

  // Field-specific
  field_type?: string;
  field_number?: number;
  is_repeated?: boolean;
  is_optional?: boolean;

  // Method-specific
  method_input_type?: string;
  method_output_type?: string;

  // Search relevance
  rank: number;

  // Metadata
  metadata?: Record<string, any>;
}

export interface EnhancedSearchResponse {
  results: EnhancedSearchResult[];
  total_count: number;
  query: string;
}

export interface UseEnhancedSearchOptions {
  debounceMs?: number;
  limit?: number;
}

export const useEnhancedSearch = (options: UseEnhancedSearchOptions = {}) => {
  const { debounceMs = 300, limit = 50 } = options;

  const [query, setQuery] = useState<string>('');
  const [debouncedQuery, setDebouncedQuery] = useState<string>('');
  const [results, setResults] = useState<EnhancedSearchResult[]>([]);
  const [totalCount, setTotalCount] = useState<number>(0);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [loadingSuggestions, setLoadingSuggestions] = useState<boolean>(false);

  const abortControllerRef = useRef<AbortController | null>(null);

  // Debounce query
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query);
    }, debounceMs);

    return () => clearTimeout(timer);
  }, [query, debounceMs]);

  // Parse filters from query
  const parseFilters = useCallback((queryStr: string): SearchFilter[] => {
    const filters: SearchFilter[] = [];
    const filterPattern = /([\w-]+):"([^"]+)"|(\w+):(\S+)/g;
    let match;

    while ((match = filterPattern.exec(queryStr)) !== null) {
      const filterName = match[1] || match[3];
      const filterValue = match[2] || match[4];

      let type: SearchFilter['type'] = 'entity';
      let display = '';

      switch (filterName) {
        case 'entity':
          type = 'entity';
          display = `Entity: ${filterValue}`;
          break;
        case 'type':
          type = 'field-type';
          display = `Type: ${filterValue}`;
          break;
        case 'module':
          type = 'module';
          display = `Module: ${filterValue}`;
          break;
        case 'version':
          type = 'version';
          display = `Version: ${filterValue}`;
          break;
        case 'has-comment':
          type = 'has-comment';
          display = 'Has comments';
          break;
        default:
          continue;
      }

      filters.push({ type, value: filterValue, display });
    }

    return filters;
  }, []);

  // Remove filter from query
  const removeFilter = useCallback((filter: SearchFilter) => {
    const filterPattern = new RegExp(
      `${filter.type === 'field-type' ? 'type' : filter.type}:("|)${filter.value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}("|)\\s*`,
      'gi'
    );
    setQuery((prev) => prev.replace(filterPattern, '').trim());
  }, []);

  // Add filter to query
  const addFilter = useCallback((type: string, value: string) => {
    setQuery((prev) => {
      const hasWhitespace = value.includes(' ');
      const filterStr = hasWhitespace ? `${type}:"${value}"` : `${type}:${value}`;
      return prev ? `${prev} ${filterStr}` : filterStr;
    });
  }, []);

  // Fetch search results
  useEffect(() => {
    if (!debouncedQuery.trim()) {
      setResults([]);
      setTotalCount(0);
      setError(null);
      return;
    }

    const fetchResults = async () => {
      // Cancel previous request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      abortControllerRef.current = new AbortController();

      setLoading(true);
      setError(null);

      try {
        const params = new URLSearchParams({
          q: debouncedQuery,
          limit: String(limit),
        });

        const response = await fetch(`/api/v2/search?${params}`, {
          signal: abortControllerRef.current.signal,
        });

        if (!response.ok) {
          throw new Error(`Search failed: ${response.statusText}`);
        }

        const data: EnhancedSearchResponse = await response.json();
        setResults(data.results);
        setTotalCount(data.total_count);
      } catch (err: any) {
        if (err.name !== 'AbortError') {
          console.error('Search error:', err);
          setError(err.message || 'Failed to search');
        }
      } finally {
        setLoading(false);
      }
    };

    fetchResults();
  }, [debouncedQuery, limit]);

  // Fetch suggestions
  const fetchSuggestions = useCallback(async (prefix: string) => {
    if (!prefix.trim() || prefix.length < 2) {
      setSuggestions([]);
      return;
    }

    setLoadingSuggestions(true);

    try {
      const params = new URLSearchParams({
        prefix,
        limit: '5',
      });

      const response = await fetch(`/api/v2/search/suggestions?${params}`);
      if (!response.ok) {
        throw new Error('Failed to fetch suggestions');
      }

      const data = await response.json();
      setSuggestions(data.suggestions || []);
    } catch (err) {
      console.error('Suggestions error:', err);
      setSuggestions([]);
    } finally {
      setLoadingSuggestions(false);
    }
  }, []);

  // Clear search
  const clear = useCallback(() => {
    setQuery('');
    setResults([]);
    setTotalCount(0);
    setError(null);
    setSuggestions([]);
  }, []);

  return {
    query,
    setQuery,
    results,
    totalCount,
    loading,
    error,
    suggestions,
    loadingSuggestions,
    fetchSuggestions,
    filters: parseFilters(query),
    removeFilter,
    addFilter,
    clear,
  };
};
