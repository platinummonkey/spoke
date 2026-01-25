import React, { useState, useMemo } from 'react';
import { Box, Heading, VStack, Text, Link, Spinner, Alert, AlertIcon, Button, Badge, Breadcrumb, BreadcrumbItem, BreadcrumbLink, Flex, Input, InputGroup, InputLeftElement, HStack, Tooltip } from '@chakra-ui/react';
import { SearchIcon, ExternalLinkIcon } from '@chakra-ui/icons';
import { Module } from '../types';
import { Link as RouterLink } from 'react-router-dom';

interface ModuleListProps {
  modules: Module[];
  loading: boolean;
  error: Error | null;
  retry: () => void;
}

export const ModuleList: React.FC<ModuleListProps> = ({ modules = [], loading, error, retry }) => {
  const [searchQuery, setSearchQuery] = useState('');

  const filteredModules = useMemo(() => {
    if (!searchQuery.trim()) return modules;
    const query = searchQuery.toLowerCase();
    return modules.filter(module => 
      module.name.toLowerCase().includes(query) || 
      (module.description && module.description.toLowerCase().includes(query))
    );
  }, [modules, searchQuery]);

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleString('en-US', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    });
  };

  if (loading) {
    return (
      <Box p={4} display="flex" justifyContent="center" alignItems="center" minH="200px">
        <Spinner size="xl" />
      </Box>
    );
  }

  if (error) {
    return (
      <Box p={4}>
        <Alert status="error" mb={4}>
          <AlertIcon />
          {error.message}
        </Alert>
        <Button onClick={retry} colorScheme="blue">
          Retry
        </Button>
      </Box>
    );
  }

  return (
    <Box p={4}>
      <Breadcrumb mb={4}>
        <BreadcrumbItem isCurrentPage>
          <BreadcrumbLink>Modules</BreadcrumbLink>
        </BreadcrumbItem>
      </Breadcrumb>

      <Flex justify="space-between" align="center" mb={4}>
        <Heading size="lg">Available Modules</Heading>
        <InputGroup maxW="400px">
          <InputLeftElement pointerEvents="none">
            <SearchIcon color="gray.300" />
          </InputLeftElement>
          <Input
            placeholder="Search modules..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </InputGroup>
      </Flex>

      {!filteredModules || filteredModules.length === 0 ? (
        <Text>No modules found</Text>
      ) : (
        <VStack align="stretch" spacing={4}>
          {filteredModules.map((module) => {
            const latestVersion = module.versions?.[0];
            return (
              <Box
                key={module.name}
                p={4}
                borderWidth="1px"
                borderRadius="lg"
                _hover={{ shadow: 'md' }}
              >
                <Link as={RouterLink} to={`/modules/${module.name}`}>
                  <Heading size="md">{module.name}</Heading>
                </Link>
                <Text mt={2}>{module.description || 'No description available'}</Text>
                <Box mt={2} display="flex" flexDirection="column" gap={2}>
                  <HStack>
                    <Text fontSize="sm" color="gray.500">
                      {module.versions?.length || 0} version{(module.versions?.length || 0) !== 1 ? 's' : ''}
                    </Text>
                    {latestVersion && (
                      <Flex alignItems="center" gap={2}>
                        <Badge colorScheme="blue">
                          Latest: {latestVersion.version}
                        </Badge>
                        <Text fontSize="xs" color="gray.500">
                          {formatDate(latestVersion.created_at)}
                        </Text>
                      </Flex>
                    )}
                  </HStack>
                  {latestVersion && latestVersion.source_info && (
                    <HStack spacing={4} fontSize="sm" color="gray.600">
                      {latestVersion.source_info.repository !== 'unknown' && (
                        <Tooltip label="View Repository">
                          <Link href={latestVersion.source_info.repository} isExternal>
                            <HStack spacing={1}>
                              <Text>Repository</Text>
                              <ExternalLinkIcon />
                            </HStack>
                          </Link>
                        </Tooltip>
                      )}
                      {latestVersion.source_info.commit_sha !== 'unknown' && (
                        <Tooltip label="Commit SHA">
                          <Text>Commit: {latestVersion.source_info.commit_sha.substring(0, 7)}</Text>
                        </Tooltip>
                      )}
                      {latestVersion.source_info.branch !== 'unknown' && (
                        <Tooltip label="Branch">
                          <Text>Branch: {latestVersion.source_info.branch}</Text>
                        </Tooltip>
                      )}
                    </HStack>
                  )}
                </Box>
              </Box>
            );
          })}
        </VStack>
      )}
    </Box>
  );
}; 