import { PluginVersion } from '../../types/plugin';
import pluginService from '../../services/pluginService';
import './VersionList.css';

interface VersionListProps {
  versions: PluginVersion[];
  pluginId: string;
}

export default function VersionList({ versions, pluginId }: VersionListProps) {
  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
  };

  const handleDownload = async (version: string) => {
    try {
      const url = await pluginService.downloadPlugin(pluginId, version);
      window.location.href = url;
    } catch (error) {
      alert('Failed to download plugin');
    }
  };

  if (versions.length === 0) {
    return (
      <div className="empty-versions">
        <p>No versions available</p>
      </div>
    );
  }

  return (
    <div className="version-list">
      <table className="versions-table">
        <thead>
          <tr>
            <th>Version</th>
            <th>API Version</th>
            <th>Size</th>
            <th>Downloads</th>
            <th>Released</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {versions.map((version) => (
            <tr key={version.id}>
              <td className="version-column">
                <span className="version-number">{version.version}</span>
              </td>
              <td className="api-version-column">{version.api_version}</td>
              <td className="size-column">{formatBytes(version.size_bytes)}</td>
              <td className="downloads-column">{version.downloads.toLocaleString()}</td>
              <td className="date-column">
                {new Date(version.created_at).toLocaleDateString()}
              </td>
              <td className="actions-column">
                <button
                  onClick={() => handleDownload(version.version)}
                  className="download-button"
                  title="Download this version"
                >
                  Download
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
