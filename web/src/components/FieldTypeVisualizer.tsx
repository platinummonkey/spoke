import React from 'react';
import { Badge, Tooltip } from '@chakra-ui/react';

interface FieldTypeVisualizerProps {
  type: string;
  label?: 'optional' | 'repeated' | 'required';
  size?: 'sm' | 'md' | 'lg';
}

// Determine color scheme based on type
const getTypeColor = (type: string): string => {
  // Scalar types
  if (['string', 'bytes'].includes(type)) return 'purple';
  if (['int32', 'int64', 'uint32', 'uint64', 'sint32', 'sint64', 'fixed32', 'fixed64', 'sfixed32', 'sfixed64'].includes(type)) return 'blue';
  if (['float', 'double'].includes(type)) return 'cyan';
  if (type === 'bool') return 'green';

  // Message types (custom types start with uppercase)
  if (type[0] === type[0].toUpperCase()) return 'orange';

  // Default
  return 'gray';
};

const getLabelColor = (label: string): string => {
  if (label === 'repeated') return 'blue';
  if (label === 'required') return 'red';
  return 'gray';
};

const getTypeDescription = (type: string): string => {
  const descriptions: Record<string, string> = {
    'string': 'Text value',
    'bytes': 'Binary data',
    'int32': '32-bit integer',
    'int64': '64-bit integer',
    'uint32': 'Unsigned 32-bit integer',
    'uint64': 'Unsigned 64-bit integer',
    'sint32': 'Signed 32-bit integer (zigzag encoding)',
    'sint64': 'Signed 64-bit integer (zigzag encoding)',
    'fixed32': 'Fixed 32-bit',
    'fixed64': 'Fixed 64-bit',
    'sfixed32': 'Signed fixed 32-bit',
    'sfixed64': 'Signed fixed 64-bit',
    'float': '32-bit floating point',
    'double': '64-bit floating point',
    'bool': 'Boolean value (true/false)',
  };

  return descriptions[type] || `Custom message type: ${type}`;
};

export const FieldTypeVisualizer: React.FC<FieldTypeVisualizerProps> = ({ type, label, size = 'sm' }) => {
  const typeColor = getTypeColor(type);
  const typeDescription = getTypeDescription(type);

  return (
    <>
      <Tooltip label={typeDescription} placement="top">
        <Badge colorScheme={typeColor} size={size} mr={1}>
          {type}
        </Badge>
      </Tooltip>
      {label && label !== 'optional' && (
        <Badge colorScheme={getLabelColor(label)} size={size}>
          {label}
        </Badge>
      )}
    </>
  );
};
