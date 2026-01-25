import React, { useState, useMemo } from 'react';
import { Box, Heading, VStack, Text, Link, Alert, AlertIcon, Button, Badge, Breadcrumb, BreadcrumbItem, BreadcrumbLink, Flex, Input, InputGroup, InputLeftElement, HStack, Tooltip } from '@chakra-ui/react';
import { SearchIcon, ExternalLinkIcon } from '@chakra-ui/icons';
import { Module } from '../types';
import { Link as RouterLink } from 'react-router-dom';
import { LoadingSkeleton } from './LoadingSkeleton';
import { EmptyState } from './EmptyState';

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
      <Box p={4}>
        <Breadcrumb mb={4}>
          <BreadcrumbItem isCurrentPage>
            <BreadcrumbLink>Modules</BreadcrumbLink>
          </BreadcrumbItem>
        </Breadcrumb>
        <Heading size="lg" mb={4}>Available Modules</Heading>
        <LoadingSkeleton type="module-list" count={5} />
      </Box>
    );
  }

  if (error) {
    return (
      <Box p={4}>
        <Alert
          status="error"
          variant="subtle"
          flexDirection="column"
          alignItems="center"
          justifyContent="center"
          textAlign="center"
          minH="200px"
          borderRadius="md"
          mb={4}
        >
          <AlertIcon boxSize="40px" mr={0} />
          <Heading size="md" mt={4} mb={1}>
            Failed to load modules
          </Heading>
          <Text maxW="500px" mt={2}>
            {error.message || 'An unexpected error occurred while loading the module list.'}
          </Text>
          <Button onClick={retry} colorScheme="blue" mt={4} aria-label="Retry loading modules">
            Try Again
          </Button>
        </Alert>
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
            aria-label="Search modules by name or description"
          />
        </InputGroup>
      </Flex>

      {!filteredModules || filteredModules.length === 0 ? (
        <EmptyState
          title={searchQuery ? 'No matching modules' : 'No modules available'}
          description={
            searchQuery
              ? `No modules match "${searchQuery}". Try a different search term.`
              : 'There are no modules registered in the registry yet. Push your first module to get started.'
          }
          actionLabel={searchQuery ? 'Clear Search' : undefined}
          onAction={searchQuery ? () => setSearchQuery('') : undefined}
        />
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
                transition="box-shadow 0.2s"
                role="article"
                aria-label={`Module: ${module.name}`}
              >
                <Link
                  as={RouterLink}
                  to={`/modules/${module.name}`}
                  _hover={{ textDecoration: 'none', color: 'blue.600' }}
                  aria-label={`View ${module.name} module details`}
                >
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