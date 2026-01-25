import React from 'react';
import {
  Box,
  VStack,
  HStack,
  Text,
  Badge,
  Link,
  Divider,
} from '@chakra-ui/react';
import { Link as RouterLink } from 'react-router-dom';
import { SearchResult } from '../hooks/useSearch';

interface SearchResultsProps {
  results: SearchResult[];
  query: string;
  onClose: () => void;
}

const highlightText = (text: string, query: string): React.ReactNode => {
  if (!query.trim()) {
    return text;
  }

  const parts = text.split(new RegExp(`(${query})`, 'gi'));
  return (
    <>
      {parts.map((part, index) =>
        part.toLowerCase() === query.toLowerCase() ? (
          <Box as="mark" key={index} bg="yellow.200" fontWeight="bold">
            {part}
          </Box>
        ) : (
          <span key={index}>{part}</span>
        )
      )}
    </>
  );
};

const getMatchBadgeColor = (field: string): string => {
  switch (field) {
    case 'name':
      return 'blue';
    case 'description':
      return 'green';
    case 'messages':
      return 'purple';
    case 'services':
      return 'orange';
    case 'methods':
      return 'pink';
    case 'enums':
      return 'teal';
    case 'fields':
      return 'cyan';
    default:
      return 'gray';
  }
};

export const SearchResults: React.FC<SearchResultsProps> = ({
  results,
  query,
  onClose,
}) => {
  if (results.length === 0) {
    return (
      <Box p={4} bg="white" borderRadius="md" boxShadow="lg" maxW="600px">
        <Text color="gray.600" fontSize="sm">
          No results found for "{query}"
        </Text>
        <Text color="gray.500" fontSize="xs" mt={2}>
          Try searching for module names, message types, services, or methods.
        </Text>
      </Box>
    );
  }

  return (
    <Box
      bg="white"
      borderRadius="md"
      boxShadow="xl"
      maxW="600px"
      maxH="500px"
      overflowY="auto"
      borderWidth={1}
    >
      <Box p={3} borderBottomWidth={1} bg="gray.50">
        <Text fontSize="sm" color="gray.600">
          Found {results.length} result{results.length > 1 ? 's' : ''} for "{query}"
        </Text>
      </Box>

      <VStack align="stretch" spacing={0} divider={<Divider />}>
        {results.map((result) => (
          <Link
            key={result.entry.id}
            as={RouterLink}
            to={`/modules/${result.entry.name}/versions/${result.entry.version}`}
            onClick={onClose}
            _hover={{ textDecoration: 'none', bg: 'blue.50' }}
            transition="background-color 0.2s"
          >
            <Box p={4}>
              <VStack align="stretch" spacing={2}>
                {/* Module name and version */}
                <HStack justify="space-between">
                  <Text fontWeight="bold" fontSize="md">
                    {highlightText(result.entry.name, query)}
                  </Text>
                  <Badge colorScheme="gray" fontSize="xs">
                    {result.entry.version}
                  </Badge>
                </HStack>

                {/* Description */}
                {result.entry.description && (
                  <Text fontSize="sm" color="gray.600" noOfLines={2}>
                    {highlightText(result.entry.description, query)}
                  </Text>
                )}

                {/* Matched fields badges */}
                {result.matchedFields.length > 0 && (
                  <HStack spacing={1} flexWrap="wrap">
                    <Text fontSize="xs" color="gray.500">
                      Matched:
                    </Text>
                    {result.matchedFields.map((field) => (
                      <Badge
                        key={field}
                        colorScheme={getMatchBadgeColor(field)}
                        fontSize="xs"
                        variant="subtle"
                      >
                        {field}
                      </Badge>
                    ))}
                  </HStack>
                )}

                {/* Preview matched content */}
                {result.matchedFields.includes('messages') &&
                  result.entry.messages.length > 0 && (
                    <Box fontSize="xs" color="gray.500">
                      <Text as="span" fontWeight="medium">
                        Messages:{' '}
                      </Text>
                      {result.entry.messages.slice(0, 3).join(', ')}
                      {result.entry.messages.length > 3 && '...'}
                    </Box>
                  )}

                {result.matchedFields.includes('services') &&
                  result.entry.services.length > 0 && (
                    <Box fontSize="xs" color="gray.500">
                      <Text as="span" fontWeight="medium">
                        Services:{' '}
                      </Text>
                      {result.entry.services.slice(0, 3).join(', ')}
                      {result.entry.services.length > 3 && '...'}
                    </Box>
                  )}

                {result.matchedFields.includes('methods') &&
                  result.entry.methods.length > 0 && (
                    <Box fontSize="xs" color="gray.500">
                      <Text as="span" fontWeight="medium">
                        Methods:{' '}
                      </Text>
                      {result.entry.methods.slice(0, 3).join(', ')}
                      {result.entry.methods.length > 3 && '...'}
                    </Box>
                  )}
              </VStack>
            </Box>
          </Link>
        ))}
      </VStack>

      <Box p={3} borderTopWidth={1} bg="gray.50">
        <Text fontSize="xs" color="gray.500" textAlign="center">
          Press ESC to close • Click to view details
        </Text>
      </Box>
    </Box>
  );
};
