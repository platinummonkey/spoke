import { useState } from 'react';
import { PluginType, SecurityLevel } from '../../types/plugin';
import { usePlugins } from '../../hooks/usePlugins';
import PluginCard from './PluginCard';
import PluginFilters from './PluginFilters';
import './PluginMarketplace.css';

export default function PluginMarketplace() {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedType, setSelectedType] = useState<PluginType | 'all'>('all');
  const [selectedSecurityLevel, setSelectedSecurityLevel] = useState<SecurityLevel | 'all'>('all');
  const [sortBy, setSortBy] = useState<'downloads' | 'rating' | 'created_at'>('downloads');
  const [page, setPage] = useState(0);

  const limit = 20;
  const offset = page * limit;

  const { data, isLoading, error } = usePlugins({
    type: selectedType !== 'all' ? selectedType : undefined,
    security_level: selectedSecurityLevel !== 'all' ? selectedSecurityLevel : undefined,
    search: searchQuery || undefined,
    sort_by: sortBy,
    sort_order: 'desc',
    limit,
    offset,
  });

  const totalPages = data ? Math.ceil(data.total / limit) : 0;

  if (error) {
    return (
      <div className="marketplace-error">
        <h2>Error Loading Plugins</h2>
        <p>{error instanceof Error ? error.message : 'An error occurred'}</p>
      </div>
    );
  }

  return (
    <div className="plugin-marketplace">
      <header className="marketplace-header">
        <h1>Plugin Marketplace</h1>
        <p>Discover and install plugins to extend Spoke's functionality</p>
      </header>

      <PluginFilters
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        selectedType={selectedType}
        onTypeChange={setSelectedType}
        selectedSecurityLevel={selectedSecurityLevel}
        onSecurityLevelChange={setSelectedSecurityLevel}
        sortBy={sortBy}
        onSortByChange={setSortBy}
      />

      {isLoading ? (
        <div className="loading-state">
          <div className="spinner" />
          <p>Loading plugins...</p>
        </div>
      ) : (
        <>
          <div className="marketplace-stats">
            <span>{data?.total || 0} plugins found</span>
          </div>

          {data && data.plugins.length > 0 ? (
            <>
              <div className="plugin-grid">
                {data.plugins.map((plugin) => (
                  <PluginCard key={plugin.id} plugin={plugin} />
                ))}
              </div>

              {totalPages > 1 && (
                <div className="pagination">
                  <button
                    onClick={() => setPage(p => Math.max(0, p - 1))}
                    disabled={page === 0}
                    className="pagination-button"
                  >
                    Previous
                  </button>

                  <span className="pagination-info">
                    Page {page + 1} of {totalPages}
                  </span>

                  <button
                    onClick={() => setPage(p => Math.min(totalPages - 1, p + 1))}
                    disabled={page >= totalPages - 1}
                    className="pagination-button"
                  >
                    Next
                  </button>
                </div>
              )}
            </>
          ) : (
            <div className="empty-state">
              <h3>No plugins found</h3>
              <p>Try adjusting your filters or search query</p>
            </div>
          )}
        </>
      )}
    </div>
  );
}
