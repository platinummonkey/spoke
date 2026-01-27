import { SecurityLevel } from '../../types/plugin';
import './SecurityBadge.css';

interface SecurityBadgeProps {
  level: SecurityLevel;
}

export default function SecurityBadge({ level }: SecurityBadgeProps) {
  const getBadgeClass = (): string => {
    switch (level) {
      case 'official':
        return 'security-badge official';
      case 'verified':
        return 'security-badge verified';
      case 'community':
        return 'security-badge community';
      default:
        return 'security-badge';
    }
  };

  const getBadgeIcon = (): string => {
    switch (level) {
      case 'official':
        return '✓';
      case 'verified':
        return '✓';
      case 'community':
        return '○';
      default:
        return '';
    }
  };

  const getBadgeLabel = (): string => {
    switch (level) {
      case 'official':
        return 'Official';
      case 'verified':
        return 'Verified';
      case 'community':
        return 'Community';
      default:
        return level;
    }
  };

  return (
    <span className={getBadgeClass()} title={`Security Level: ${getBadgeLabel()}`}>
      {getBadgeIcon()} {getBadgeLabel()}
    </span>
  );
}
