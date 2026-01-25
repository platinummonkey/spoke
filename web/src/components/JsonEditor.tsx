import React from 'react';
import { Box, Textarea } from '@chakra-ui/react';

interface JsonEditorProps {
  value: string;
  onChange: (value: string) => void;
  readOnly?: boolean;
  height?: string;
  placeholder?: string;
}

export const JsonEditor: React.FC<JsonEditorProps> = ({
  value,
  onChange,
  readOnly = false,
  height = '300px',
  placeholder = '{}',
}) => {
  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    onChange(e.target.value);
  };

  return (
    <Box>
      <Textarea
        value={value}
        onChange={handleChange}
        readOnly={readOnly}
        placeholder={placeholder}
        height={height}
        fontFamily="monospace"
        fontSize="sm"
        bg="gray.50"
        borderRadius="md"
        borderWidth={1}
        resize="vertical"
        spellCheck={false}
        _focus={{
          borderColor: 'blue.500',
          boxShadow: '0 0 0 1px var(--chakra-colors-blue-500)',
        }}
      />
    </Box>
  );
};
