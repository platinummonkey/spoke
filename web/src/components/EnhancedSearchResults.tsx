import React from 'react';
import {
  Box,
  VStack,
  HStack,
  Text,
  Badge,
  Link,
  Divider,
  Tag,
  Tooltip,
  Wrap,
  WrapItem,
} from '@chakra-ui/react';
import { Link as RouterLink } from 'react-router-dom';
import { EnhancedSearchResult } from '../hooks/useEnhancedSearch';

interface EnhancedSearchResultsProps {
  results: EnhancedSearchResult[];
  totalCount: number;
  query: string;
  onClose: () => void;
}

const highlightText = (text: string, query: string): React.ReactNode => {
  if (!query.trim()) {
    return text;
  }

  // Remove filter syntax from query for highlighting
  const cleanQuery = query.replace(/([\w-]+):"([^"]+)"|(\w+):(\S+)/g, '').trim();
  if (!cleanQuery) {
    return text;
  }

  const parts = text.split(new RegExp(`(${cleanQuery})`, 'gi'));
  return (
    <>
      {parts.map((part, index) =>
        part.toLowerCase() === cleanQuery.toLowerCase() ? (
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

const getEntityTypeColor = (entityType: string): string => {
  switch (entityType) {
    case 'message':
      return 'purple';
    case 'enum':
      return 'teal';
    case 'service':
      return 'orange';
    case 'method':
      return 'pink';
    case 'field':
      return 'cyan';
    default:
      return 'gray';
  }
};

const formatRelevanceScore = (rank: number): string => {
  return (rank * 100).toFixed(1) + '%';
};

export const EnhancedSearchResults: React.FC<EnhancedSearchResultsProps> = ({
  results,
  totalCount,
  query,
  onClose,
}) => {
  return (
    <Box
      bg="white"
      borderRadius="md"
      boxShadow="xl"
      maxW="1200px"
      maxH="600px"
      overflowY="auto"
      borderWidth={1}
    >
      <Box p={3} borderBottomWidth={1} bg="gray.50">
        <HStack justify="space-between">
          <Text fontSize="sm" color="gray.600">
            Found {totalCount} result{totalCount > 1 ? 's' : ''} for "{query}"
          </Text>
          <Text fontSize="xs" color="gray.500">
            Showing {results.length} of {totalCount}
          </Text>
        </HStack>
      </Box>

      <VStack align="stretch" spacing={0} divider={<Divider />}>
        {results.map((result) => (
          <Link
            key={result.id}
            as={RouterLink}
            to={`/modules/${result.module_name}?version=${result.version}`}
            onClick={onClose}
            _hover={{ textDecoration: 'none', bg: 'blue.50' }}
            transition="background-color 0.2s"
          >
            <Box p={4}>
              <VStack align="stretch" spacing={3}>
                {/* Entity type and name */}
                <HStack justify="space-between" align="start">
                  <HStack spacing={2} flex={1}>
                    <Badge colorScheme={getEntityTypeColor(result.entity_type)} fontSize="xs">
                      {result.entity_type}
                    </Badge>
                    <Text fontWeight="bold" fontSize="md">
                      {highlightText(result.entity_name, query)}
                    </Text>
                  </HStack>
                  <Tooltip label="Relevance score" placement="left">
                    <Badge colorScheme="green" variant="subtle" fontSize="xs">
                      {formatRelevanceScore(result.rank)}
                    </Badge>
                  </Tooltip>
                </HStack>

                {/* Full path */}
                <HStack spacing={2} fontSize="xs" color="gray.600">
                  <Text fontWeight="medium">Path:</Text>
                  <Code fontSize="xs">{highlightText(result.full_path, query)}</Code>
                </HStack>

                {/* Parent path (if different from full path) */}
                {result.parent_path && result.parent_path !== result.full_path && (
                  <HStack spacing={2} fontSize="xs" color="gray.500">
                    <Text fontWeight="medium">Parent:</Text>
                    <Code fontSize="xs">{result.parent_path}</Code>
                  </HStack>
                )}

                {/* Module and version */}
                <HStack spacing={3}>
                  <HStack spacing={1} fontSize="xs">
                    <Text color="gray.600" fontWeight="medium">
                      Module:
                    </Text>
                    <Text color="blue.600">{result.module_name}</Text>
                  </HStack>
                  <HStack spacing={1} fontSize="xs">
                    <Text color="gray.600" fontWeight="medium">
                      Version:
                    </Text>
                    <Badge colorScheme="gray" variant="subtle" fontSize="xs">
                      {result.version}
                    </Badge>
                  </HStack>
                  {result.proto_file_path && (
                    <HStack spacing={1} fontSize="xs">
                      <Text color="gray.600" fontWeight="medium">
                        File:
                      </Text>
                      <Text color="gray.500">{result.proto_file_path}</Text>
                    </HStack>
                  )}
                </HStack>

                {/* Description */}
                {result.description && (
                  <Text fontSize="sm" color="gray.700" noOfLines={2}>
                    {highlightText(result.description, query)}
                  </Text>
                )}

                {/* Comments */}
                {result.comments && (
                  <Box bg="gray.50" p={2} borderRadius="md" fontSize="xs">
                    <Text color="gray.600" fontWeight="medium" mb={1}>
                      Comments:
                    </Text>
                    <Text color="gray.600" noOfLines={2}>
                      {result.comments}
                    </Text>
                  </Box>
                )}

                {/* Field-specific information */}
                {result.entity_type === 'field' && (
                  <Wrap spacing={2}>
                    {result.field_type && (
                      <WrapItem>
                        <Tag size="sm" colorScheme="blue" variant="subtle">
                          <Text fontSize="xs">{result.field_type}</Text>
                        </Tag>
                      </WrapItem>
                    )}
                    {result.field_number !== undefined && (
                      <WrapItem>
                        <Tag size="sm" colorScheme="gray" variant="subtle">
                          <Text fontSize="xs">Field #{result.field_number}</Text>
                        </Tag>
                      </WrapItem>
                    )}
                    {result.is_repeated && (
                      <WrapItem>
                        <Tag size="sm" colorScheme="green" variant="subtle">
                          <Text fontSize="xs">repeated</Text>
                        </Tag>
                      </WrapItem>
                    )}
                    {result.is_optional && (
                      <WrapItem>
                        <Tag size="sm" colorScheme="purple" variant="subtle">
                          <Text fontSize="xs">optional</Text>
                        </Tag>
                      </WrapItem>
                    )}
                  </Wrap>
                )}

                {/* Method-specific information */}
                {result.entity_type === 'method' && (
                  <HStack spacing={4} fontSize="xs">
                    {result.method_input_type && (
                      <HStack spacing={1}>
                        <Text color="gray.600" fontWeight="medium">
                          Input:
                        </Text>
                        <Code fontSize="xs">{result.method_input_type}</Code>
                      </HStack>
                    )}
                    {result.method_output_type && (
                      <HStack spacing={1}>
                        <Text color="gray.600" fontWeight="medium">
                          Output:
                        </Text>
                        <Code fontSize="xs">{result.method_output_type}</Code>
                      </HStack>
                    )}
                  </HStack>
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

// Add Code component import helper
const Code: React.FC<{ children: React.ReactNode; fontSize?: string }> = ({ children, fontSize = 'xs' }) => (
  <Box
    as="code"
    px={1}
    py={0.5}
    borderRadius="sm"
    bg="gray.100"
    fontSize={fontSize}
    fontFamily="mono"
  >
    {children}
  </Box>
);
