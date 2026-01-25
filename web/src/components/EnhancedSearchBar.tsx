import React, { useState, useEffect, useRef } from 'react';
import {
  Box,
  Input,
  InputGroup,
  InputLeftElement,
  InputRightElement,
  Modal,
  ModalOverlay,
  ModalContent,
  Spinner,
  Text,
  Kbd,
  HStack,
  VStack,
  useDisclosure,
  Tag,
  TagLabel,
  TagCloseButton,
  Button,
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverHeader,
  PopoverBody,
  PopoverArrow,
  PopoverCloseButton,
  Wrap,
  WrapItem,
  Divider,
  Code,
  IconButton,
} from '@chakra-ui/react';
import { SearchIcon, InfoIcon, CloseIcon } from '@chakra-ui/icons';
import { useEnhancedSearch } from '../hooks/useEnhancedSearch';
import { EnhancedSearchResults } from './EnhancedSearchResults';

export const EnhancedSearchBar: React.FC = () => {
  const { isOpen, onOpen, onClose } = useDisclosure();
  const inputRef = useRef<HTMLInputElement>(null);
  const [showSuggestions, setShowSuggestions] = useState(false);

  const {
    query,
    setQuery,
    results,
    totalCount,
    loading,
    error,
    suggestions,
    fetchSuggestions,
    filters,
    removeFilter,
    addFilter,
    clear,
  } = useEnhancedSearch({ debounceMs: 300, limit: 50 });

  // Keyboard shortcut: CMD+K or CTRL+K
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
        event.preventDefault();
        onOpen();
      }

      if (event.key === 'Escape' && isOpen) {
        onClose();
        clear();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onOpen, onClose, clear]);

  // Focus input when modal opens
  useEffect(() => {
    if (isOpen && inputRef.current) {
      setTimeout(() => inputRef.current?.focus(), 100);
    }
  }, [isOpen]);

  // Fetch suggestions when query changes
  useEffect(() => {
    if (query.trim().length >= 2) {
      fetchSuggestions(query.trim());
      setShowSuggestions(true);
    } else {
      setShowSuggestions(false);
    }
  }, [query, fetchSuggestions]);

  const handleClose = () => {
    onClose();
    clear();
    setShowSuggestions(false);
  };

  const handleSuggestionClick = (suggestion: string) => {
    setQuery(suggestion);
    setShowSuggestions(false);
  };

  const handleFilterClick = (type: string, value: string) => {
    addFilter(type, value);
    setShowSuggestions(false);
  };

  return (
    <>
      {/* Search trigger button */}
      <Box
        onClick={onOpen}
        cursor="pointer"
        bg="gray.100"
        _hover={{ bg: 'gray.200' }}
        borderRadius="md"
        px={4}
        py={2}
        transition="background-color 0.2s"
      >
        <HStack spacing={2}>
          <SearchIcon color="gray.500" boxSize={4} />
          <Text fontSize="sm" color="gray.600">
            Advanced Search...
          </Text>
          <HStack spacing={1} ml={4}>
            <Kbd fontSize="xs">⌘</Kbd>
            <Kbd fontSize="xs">K</Kbd>
          </HStack>
        </HStack>
      </Box>

      {/* Search Modal */}
      <Modal isOpen={isOpen} onClose={handleClose} size="4xl">
        <ModalOverlay bg="blackAlpha.300" backdropFilter="blur(10px)" />
        <ModalContent mt="100px" bg="transparent" boxShadow="none">
          <Box>
            {/* Search Input with Filters */}
            <Box bg="white" borderRadius="md" boxShadow="lg" p={3} mb={4}>
              <VStack align="stretch" spacing={3}>
                {/* Filter chips */}
                {filters.length > 0 && (
                  <Wrap spacing={2}>
                    {filters.map((filter, index) => (
                      <WrapItem key={index}>
                        <Tag
                          size="sm"
                          colorScheme={getFilterColorScheme(filter.type)}
                          borderRadius="full"
                        >
                          <TagLabel>{filter.display}</TagLabel>
                          <TagCloseButton onClick={() => removeFilter(filter)} />
                        </Tag>
                      </WrapItem>
                    ))}
                    <WrapItem>
                      <IconButton
                        aria-label="Clear all filters"
                        icon={<CloseIcon />}
                        size="xs"
                        variant="ghost"
                        onClick={clear}
                      />
                    </WrapItem>
                  </Wrap>
                )}

                {/* Search input with advanced filters popover */}
                <HStack spacing={2}>
                  <InputGroup size="lg" flex={1}>
                    <InputLeftElement>
                      <SearchIcon color="gray.400" />
                    </InputLeftElement>
                    <Input
                      ref={inputRef}
                      placeholder="Search protobuf schemas (e.g., 'user entity:message type:string')"
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                      border="none"
                      _focus={{ outline: 'none' }}
                      fontSize="md"
                    />
                    {loading && (
                      <InputRightElement>
                        <Spinner size="sm" color="gray.400" />
                      </InputRightElement>
                    )}
                  </InputGroup>

                  {/* Advanced Filters Popover */}
                  <Popover placement="bottom-end">
                    <PopoverTrigger>
                      <IconButton
                        aria-label="Advanced filters"
                        icon={<InfoIcon />}
                        size="md"
                        variant="ghost"
                      />
                    </PopoverTrigger>
                    <PopoverContent width="400px">
                      <PopoverArrow />
                      <PopoverCloseButton />
                      <PopoverHeader fontWeight="bold">Advanced Search Filters</PopoverHeader>
                      <PopoverBody>
                        <VStack align="stretch" spacing={4}>
                          <Box>
                            <Text fontWeight="medium" mb={2} fontSize="sm">
                              Entity Type
                            </Text>
                            <Wrap spacing={2}>
                              {['message', 'enum', 'service', 'method', 'field'].map((type) => (
                                <WrapItem key={type}>
                                  <Button
                                    size="xs"
                                    onClick={() => handleFilterClick('entity', type)}
                                  >
                                    {type}
                                  </Button>
                                </WrapItem>
                              ))}
                            </Wrap>
                          </Box>

                          <Divider />

                          <Box>
                            <Text fontWeight="medium" mb={2} fontSize="sm">
                              Query Syntax Examples
                            </Text>
                            <VStack align="stretch" spacing={1} fontSize="xs">
                              <Code>entity:message</Code>
                              <Text color="gray.600">Find messages</Text>

                              <Code>type:string</Code>
                              <Text color="gray.600">Find string fields</Text>

                              <Code>module:user</Code>
                              <Text color="gray.600">Search in user module</Text>

                              <Code>has-comment:true</Code>
                              <Text color="gray.600">Only entities with comments</Text>

                              <Code>user entity:message type:string</Code>
                              <Text color="gray.600">Combined filters</Text>
                            </VStack>
                          </Box>
                        </VStack>
                      </PopoverBody>
                    </PopoverContent>
                  </Popover>
                </HStack>

                {/* Suggestions */}
                {showSuggestions && suggestions.length > 0 && !loading && (
                  <Box bg="gray.50" borderRadius="md" p={2}>
                    <Text fontSize="xs" color="gray.600" mb={1}>
                      Suggestions:
                    </Text>
                    <Wrap spacing={2}>
                      {suggestions.map((suggestion, index) => (
                        <WrapItem key={index}>
                          <Button
                            size="xs"
                            variant="outline"
                            onClick={() => handleSuggestionClick(suggestion)}
                          >
                            {suggestion}
                          </Button>
                        </WrapItem>
                      ))}
                    </Wrap>
                  </Box>
                )}
              </VStack>
            </Box>

            {/* Search Results or Info */}
            {error && (
              <Box bg="white" p={4} borderRadius="md" boxShadow="lg">
                <Text color="red.500" fontSize="sm">
                  {error}
                </Text>
              </Box>
            )}

            {!loading && !error && query.trim() && results.length > 0 && (
              <EnhancedSearchResults
                results={results}
                totalCount={totalCount}
                query={query}
                onClose={handleClose}
              />
            )}

            {!loading && !error && query.trim() && results.length === 0 && (
              <Box bg="white" p={4} borderRadius="md" boxShadow="lg">
                <Text color="gray.600" fontSize="sm">
                  No results found for "{query}"
                </Text>
                <Text color="gray.500" fontSize="xs" mt={2}>
                  Try different keywords or use filters like entity:message or type:string
                </Text>
              </Box>
            )}

            {/* Keyboard shortcuts hint */}
            {!query.trim() && (
              <Box bg="white" p={4} borderRadius="md" boxShadow="lg">
                <Text fontSize="sm" color="gray.600" mb={3} fontWeight="medium">
                  Quick Search Tips:
                </Text>
                <VStack align="stretch" spacing={2} fontSize="xs" color="gray.500">
                  <HStack>
                    <Kbd>⌘K</Kbd>
                    <Text>or</Text>
                    <Kbd>Ctrl+K</Kbd>
                    <Text>- Open search</Text>
                  </HStack>
                  <HStack>
                    <Kbd>ESC</Kbd>
                    <Text>- Close search</Text>
                  </HStack>
                  <Text mt={2}>
                    Search by entity name, module, field type, or use advanced filters.
                  </Text>
                </VStack>
              </Box>
            )}
          </Box>
        </ModalContent>
      </Modal>
    </>
  );
};

// Helper to get filter color scheme
function getFilterColorScheme(type: string): string {
  switch (type) {
    case 'entity':
      return 'purple';
    case 'field-type':
      return 'blue';
    case 'module':
      return 'green';
    case 'version':
      return 'orange';
    case 'has-comment':
      return 'cyan';
    default:
      return 'gray';
  }
}
