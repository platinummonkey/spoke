import { useState, useEffect, useCallback } from 'react';
import lunr from 'lunr';

export interface SearchEntry {
  id: string;
  name: string;
  version: string;
  description: string;
  messages: string[];
  enums: string[];
  services: string[];
  methods: string[];
  fields: string[];
}

export interface SearchIndex {
  modules: SearchEntry[];
}

export interface SearchResult {
  entry: SearchEntry;
  score: number;
  matchedFields: string[];
}

export const useSearch = () => {
  const [index, setIndex] = useState<lunr.Index | null>(null);
  const [documents, setDocuments] = useState<SearchEntry[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  // Load and build search index
  useEffect(() => {
    const loadIndex = async () => {
      try {
        setLoading(true);
        setError(null);

        // Fetch search index
        const response = await fetch('/search-index.json');
        if (!response.ok) {
          throw new Error('Failed to load search index');
        }

        const data: SearchIndex = await response.json();
        setDocuments(data.modules);

        // Build Lunr index
        const idx = lunr(function(this: lunr.Builder) {
          this.ref('id');

          // Configure fields with boost values
          this.field('name', { boost: 10 });
          this.field('description', { boost: 5 });
          this.field('messages', { boost: 3 });
          this.field('services', { boost: 3 });
          this.field('methods', { boost: 2 });
          this.field('enums', { boost: 2 });
          this.field('fields', { boost: 1 });

          // Add documents to index
          data.modules.forEach((doc: SearchEntry) => {
            this.add({
              id: doc.id,
              name: doc.name,
              description: doc.description,
              messages: doc.messages.join(' '),
              services: doc.services.join(' '),
              methods: doc.methods.join(' '),
              enums: doc.enums.join(' '),
              fields: doc.fields.join(' '),
            });
          });
        });

        setIndex(idx);
      } catch (err) {
        console.error('Error loading search index:', err);
        setError(err instanceof Error ? err.message : 'Failed to load search index');
      } finally {
        setLoading(false);
      }
    };

    loadIndex();
  }, []);

  // Search function
  const search = useCallback((query: string): SearchResult[] => {
    if (!index || !query.trim()) {
      return [];
    }

    try {
      // Perform search
      const results = index.search(query);

      // Map results to documents with matched fields
      return results.map((result: lunr.Index.Result) => {
        const doc = documents.find((d: SearchEntry) => d.id === result.ref);
        if (!doc) {
          return null;
        }

        // Determine which fields matched
        const matchedFields: string[] = [];
        const lowerQuery = query.toLowerCase();

        if (doc.name.toLowerCase().includes(lowerQuery)) {
          matchedFields.push('name');
        }
        if (doc.description.toLowerCase().includes(lowerQuery)) {
          matchedFields.push('description');
        }
        if (doc.messages.some(m => m.toLowerCase().includes(lowerQuery))) {
          matchedFields.push('messages');
        }
        if (doc.services.some(s => s.toLowerCase().includes(lowerQuery))) {
          matchedFields.push('services');
        }
        if (doc.methods.some(m => m.toLowerCase().includes(lowerQuery))) {
          matchedFields.push('methods');
        }
        if (doc.enums.some(e => e.toLowerCase().includes(lowerQuery))) {
          matchedFields.push('enums');
        }
        if (doc.fields.some(f => f.toLowerCase().includes(lowerQuery))) {
          matchedFields.push('fields');
        }

        return {
          entry: doc,
          score: result.score,
          matchedFields,
        };
      }).filter((r: SearchResult | null): r is SearchResult => r !== null)
        .slice(0, 10); // Limit to top 10 results
    } catch (err) {
      console.error('Search error:', err);
      return [];
    }
  }, [index, documents]);

  return {
    search,
    loading,
    error,
    ready: !!index,
  };
};
