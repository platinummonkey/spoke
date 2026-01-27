import { Link } from 'react-router-dom';
import { Plugin } from '../../types/plugin';
import SecurityBadge from './SecurityBadge';
import StarRating from './StarRating';
import './PluginCard.css';

interface PluginCardProps {
  plugin: Plugin;
}

export default function PluginCard({ plugin }: PluginCardProps) {
  const formatNumber = (num: number): string => {
    if (num >= 1000000) {
      return `${(num / 1000000).toFixed(1)}M`;
    }
    if (num >= 1000) {
      return `${(num / 1000).toFixed(1)}K`;
    }
    return num.toString();
  };

  const getTypeIcon = (type: string): string => {
    const icons: Record<string, string> = {
      language: '🔤',
      validator: '✅',
      generator: '⚙️',
      runner: '🏃',
      transform: '🔄',
    };
    return icons[type] || '📦';
  };

  return (
    <Link to={`/plugins/${plugin.id}`} className="plugin-card">
      <div className="plugin-card-header">
        <div className="plugin-icon">{getTypeIcon(plugin.type)}</div>
        <div className="plugin-title-section">
          <h3 className="plugin-name">{plugin.name}</h3>
          <SecurityBadge level={plugin.security_level} />
        </div>
      </div>

      <p className="plugin-description">{plugin.description}</p>

      <div className="plugin-metadata">
        <div className="metadata-item">
          <span className="metadata-label">Author:</span>
          <span className="metadata-value">{plugin.author}</span>
        </div>

        <div className="metadata-item">
          <span className="metadata-label">Type:</span>
          <span className="metadata-value">{plugin.type}</span>
        </div>

        {plugin.latest_version && (
          <div className="metadata-item">
            <span className="metadata-label">Version:</span>
            <span className="metadata-value">{plugin.latest_version}</span>
          </div>
        )}
      </div>

      <div className="plugin-stats">
        <div className="stat-item">
          <StarRating rating={plugin.avg_rating || 0} />
          {plugin.review_count !== undefined && (
            <span className="review-count">({plugin.review_count})</span>
          )}
        </div>

        <div className="stat-item">
          <span className="download-icon">📥</span>
          <span className="download-count">{formatNumber(plugin.download_count)}</span>
        </div>
      </div>

      {plugin.license && (
        <div className="plugin-license">
          <span className="license-badge">{plugin.license}</span>
        </div>
      )}
    </Link>
  );
}
