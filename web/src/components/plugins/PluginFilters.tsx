import { PluginType, SecurityLevel } from '../../types/plugin';
import './PluginFilters.css';

interface PluginFiltersProps {
  searchQuery: string;
  onSearchChange: (query: string) => void;
  selectedType: PluginType | 'all';
  onTypeChange: (type: PluginType | 'all') => void;
  selectedSecurityLevel: SecurityLevel | 'all';
  onSecurityLevelChange: (level: SecurityLevel | 'all') => void;
  sortBy: 'downloads' | 'rating' | 'created_at';
  onSortByChange: (sortBy: 'downloads' | 'rating' | 'created_at') => void;
}

export default function PluginFilters({
  searchQuery,
  onSearchChange,
  selectedType,
  onTypeChange,
  selectedSecurityLevel,
  onSecurityLevelChange,
  sortBy,
  onSortByChange,
}: PluginFiltersProps) {
  return (
    <div className="plugin-filters">
      <div className="search-section">
        <input
          type="text"
          className="search-input"
          placeholder="Search plugins..."
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
        />
      </div>

      <div className="filter-row">
        <div className="filter-group">
          <label htmlFor="type-filter">Type:</label>
          <select
            id="type-filter"
            className="filter-select"
            value={selectedType}
            onChange={(e) => onTypeChange(e.target.value as PluginType | 'all')}
          >
            <option value="all">All Types</option>
            <option value="language">Language</option>
            <option value="validator">Validator</option>
            <option value="generator">Generator</option>
            <option value="runner">Runner</option>
            <option value="transform">Transform</option>
          </select>
        </div>

        <div className="filter-group">
          <label htmlFor="security-filter">Security Level:</label>
          <select
            id="security-filter"
            className="filter-select"
            value={selectedSecurityLevel}
            onChange={(e) => onSecurityLevelChange(e.target.value as SecurityLevel | 'all')}
          >
            <option value="all">All Levels</option>
            <option value="official">Official</option>
            <option value="verified">Verified</option>
            <option value="community">Community</option>
          </select>
        </div>

        <div className="filter-group">
          <label htmlFor="sort-filter">Sort By:</label>
          <select
            id="sort-filter"
            className="filter-select"
            value={sortBy}
            onChange={(e) => onSortByChange(e.target.value as 'downloads' | 'rating' | 'created_at')}
          >
            <option value="downloads">Most Downloaded</option>
            <option value="rating">Highest Rated</option>
            <option value="created_at">Newest</option>
          </select>
        </div>
      </div>
    </div>
  );
}
