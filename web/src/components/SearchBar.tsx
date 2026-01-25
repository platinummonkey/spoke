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
} from '@chakra-ui/react';
import { SearchIcon } from '@chakra-ui/icons';
import { useSearch } from '../hooks/useSearch';
import { SearchResults } from './SearchResults';

export const SearchBar: React.FC = () => {
  const { isOpen, onOpen, onClose } = useDisclosure();
  const [query, setQuery] = useState<string>('');
  const [debouncedQuery, setDebouncedQuery] = useState<string>('');
  const inputRef = useRef<HTMLInputElement>(null);
  const { search, loading, error, ready } = useSearch();

  // Debounce search query
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query);
    }, 300);

    return () => clearTimeout(timer);
  }, [query]);

  // Keyboard shortcut: CMD+K or CTRL+K
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
        event.preventDefault();
        onOpen();
      }

      if (event.key === 'Escape' && isOpen) {
        onClose();
        setQuery('');
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onOpen, onClose]);

  // Focus input when modal opens
  useEffect(() => {
    if (isOpen && inputRef.current) {
      setTimeout(() => inputRef.current?.focus(), 100);
    }
  }, [isOpen]);

  // Perform search
  const results = debouncedQuery.trim() ? search(debouncedQuery) : [];

  const handleClose = () => {
    onClose();
    setQuery('');
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
            Search...
          </Text>
          <HStack spacing={1} ml={4}>
            <Kbd fontSize="xs">⌘</Kbd>
            <Kbd fontSize="xs">K</Kbd>
          </HStack>
        </HStack>
      </Box>

      {/* Search Modal */}
      <Modal isOpen={isOpen} onClose={handleClose} size="xl">
        <ModalOverlay bg="blackAlpha.300" backdropFilter="blur(10px)" />
        <ModalContent mt="100px" bg="transparent" boxShadow="none">
          <Box>
            {/* Search Input */}
            <Box bg="white" borderRadius="md" boxShadow="lg" p={2} mb={4}>
              <InputGroup size="lg">
                <InputLeftElement>
                  <SearchIcon color="gray.400" />
                </InputLeftElement>
                <Input
                  ref={inputRef}
                  placeholder="Search modules, messages, services, methods..."
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
            </Box>

            {/* Search Results */}
            {!loading && error && (
              <Box bg="white" p={4} borderRadius="md" boxShadow="lg">
                <Text color="red.500" fontSize="sm">
                  {error}
                </Text>
              </Box>
            )}

            {!loading && !error && ready && query.trim() && (
              <SearchResults
                results={results}
                query={debouncedQuery}
                onClose={handleClose}
              />
            )}

            {!loading && !error && !ready && (
              <Box bg="white" p={4} borderRadius="md" boxShadow="lg">
                <Text color="gray.600" fontSize="sm">
                  Loading search index...
                </Text>
              </Box>
            )}

            {/* Keyboard shortcuts hint */}
            {!query.trim() && (
              <Box bg="white" p={4} borderRadius="md" boxShadow="lg">
                <Text fontSize="sm" color="gray.600" mb={3}>
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
                    Search by module name, message type, service, method, enum, or field name.
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
