import React, { useState, useEffect } from 'react';
import {
  Box,
  VStack,
  Heading,
  Text,
  Badge,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  Button,
  Spinner,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  Flex,
  Link,
  Select,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  Button as ChakraButton,
  Input,
  InputGroup,
  InputLeftElement,
} from '@chakra-ui/react';
import { ChevronRightIcon, ChevronDownIcon, SearchIcon } from '@chakra-ui/icons';
import { Link as RouterLink } from 'react-router-dom';
import { Module, ProtoFile } from '../types';
import { ProtoTypes } from './ProtoTypes';
import { ApiExplorer } from './ApiExplorer';
import { CodeExamples } from './CodeExamples';

// UsageExamples component removed - now using CodeExamples component

interface ModuleDetailProps {
  module: Module | null;
  loading: boolean;
  error: Error | null;
  retry: () => void;
  initialVersion?: string;
}

export const ModuleDetail: React.FC<ModuleDetailProps> = ({ module, loading, error, retry, initialVersion }) => {
  const [selectedVersion, setSelectedVersion] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

  // Update selectedVersion when module or initialVersion changes
  useEffect(() => {
    if (!module) return;
    
    if (initialVersion) {
      setSelectedVersion(initialVersion);
    } else if (module.versions && module.versions.length > 0) {
      // Default to the first version (newest) if no initialVersion is provided
      setSelectedVersion(module.versions[0].version);
    }
  }, [module, initialVersion]);

  if (loading) {
    return (
      <Box textAlign="center" py={10}>
        <Spinner size="xl" />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert status="error" mb={4}>
        <AlertIcon />
        <AlertTitle>Error loading module</AlertTitle>
        <AlertDescription>{error.message}</AlertDescription>
        <Button ml="auto" onClick={retry}>
          Retry
        </Button>
      </Alert>
    );
  }

  if (!module) {
    return (
      <Alert status="warning">
        <AlertIcon />
        <AlertTitle>Module not found</AlertTitle>
        <AlertDescription>The requested module could not be found.</AlertDescription>
      </Alert>
    );
  }

  // Versions are already sorted by newest first from the API
  const versions = module.versions?.map(v => v.version) || [];
  const selectedVersionData = module.versions?.find(v => v.version === selectedVersion) || module.versions?.[0];

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

  // Filter versions based on search query
  const filteredVersions = module.versions?.filter(version => {
    if (!searchQuery) return true;
    
    const query = searchQuery.toLowerCase();
    
    // Check version info
    if (
      version.version.toLowerCase().includes(query) ||
      version.source_info.repository.toLowerCase().includes(query) ||
      version.source_info.commit_sha.toLowerCase().includes(query) ||
      version.source_info.branch.toLowerCase().includes(query)
    ) {
      return true;
    }
    
    // Check dependencies
    if (version.dependencies) {
      return version.dependencies.some(dep => dep.toLowerCase().includes(query));
    }
    
    return false;
  }) || [];

  return (
    <VStack align="stretch" spacing={6}>
      <Breadcrumb spacing="8px" separator={<ChevronRightIcon color="gray.500" />}>
        <BreadcrumbItem>
          <BreadcrumbLink as={RouterLink} to="/">
            Modules
          </BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbItem isCurrentPage>
          <BreadcrumbLink>{module.name}</BreadcrumbLink>
        </BreadcrumbItem>
      </Breadcrumb>

      <Box>
        <Flex justify="space-between" align="center">
          <Box>
            <Heading size="lg">{module.name}</Heading>
            <Text mt={2} color="gray.600">
              {module.description || 'No description available'}
            </Text>
          </Box>
          <Menu>
            <MenuButton
              as={ChakraButton}
              rightIcon={<ChevronDownIcon />}
              size="sm"
              width="300px"
              textAlign="left"
            >
              {selectedVersion || 'Select version'}
            </MenuButton>
            <MenuList>
              {versions.map((v) => {
                const versionData = module.versions?.find(ver => ver.version === v);
                return (
                  <MenuItem
                    key={v}
                    onClick={() => setSelectedVersion(v)}
                    display="flex"
                    flexDirection="column"
                    alignItems="flex-start"
                  >
                    <Text fontWeight="bold">{v}</Text>
                    <Text fontSize="xs" color="gray.500">
                      {versionData ? formatDate(versionData.created_at) : ''}
                    </Text>
                  </MenuItem>
                );
              })}
            </MenuList>
          </Menu>
        </Flex>
      </Box>

      <Tabs>
        <TabList>
          <Tab>Overview</Tab>
          <Tab>Types</Tab>
          <Tab>API Explorer</Tab>
          <Tab>Usage Examples</Tab>
        </TabList>

        <TabPanels>
          <TabPanel>
            <VStack align="stretch" spacing={4}>
              <Box>
                <Flex justify="space-between" align="center" mb={4}>
                  <Heading size="md">Versions</Heading>
                  <InputGroup maxW="400px">
                    <InputLeftElement pointerEvents="none">
                      <SearchIcon color="gray.300" />
                    </InputLeftElement>
                    <Input
                      placeholder="Search by version, repository, commit, or branch..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                    />
                  </InputGroup>
                </Flex>

                {/* Add Download Section */}
                {selectedVersionData && (
                  <Box mb={6} p={4} borderWidth={1} borderRadius="md" bg="gray.50">
                    <Heading size="sm" mb={3}>Download Compiled Version</Heading>
                    <Flex align="center" gap={4}>
                      <Select
                        placeholder="Select language"
                        width="200px"
                        onChange={(e) => {
                          const language = e.target.value;
                          if (language) {
                            window.location.href = `/api/modules/${module.name}/versions/${selectedVersionData.version}/download/${language}`;
                          }
                        }}
                      >
                        <option value="go">Go</option>
                        <option value="python">Python</option>
                      </Select>
                      <Text fontSize="sm" color="gray.600">
                        Select a language to download the compiled version
                      </Text>
                    </Flex>
                  </Box>
                )}

                <VStack align="stretch" spacing={2}>
                  {filteredVersions.map((version) => (
                    <Box
                      key={version.version}
                      p={3}
                      borderWidth={1}
                      borderRadius="md"
                      bg={version === selectedVersionData ? 'blue.50' : 'white'}
                      cursor="pointer"
                      onClick={() => setSelectedVersion(version.version)}
                    >
                      <Flex justify="space-between" align="center">
                        <Text fontWeight="bold">{version.version}</Text>
                        <Text fontSize="xs" color="gray.500">
                          {formatDate(version.created_at)}
                        </Text>
                      </Flex>
                      <Text fontSize="sm" color="gray.600">
                        {version.files?.length || 0} files
                      </Text>
                      <Box mt={2}>
                        <Text fontSize="sm" color="gray.600">
                          Source: {version.source_info.repository} ({version.source_info.branch})
                        </Text>
                        <Text fontSize="xs" color="gray.500">
                          Commit: {version.source_info.commit_sha}
                        </Text>
                      </Box>
                      {version.dependencies && version.dependencies.length > 0 && (
                        <Box mt={2}>
                          <Text fontSize="sm" fontWeight="medium">
                            Dependencies:
                          </Text>
                          <Box mt={1}>
                            {version.dependencies.map((dep) => {
                              const [moduleName, depVersion] = dep.split('@');
                              return (
                                <Link
                                  key={dep}
                                  as={RouterLink}
                                  to={`/modules/${moduleName}/versions/${depVersion}`}
                                  _hover={{ textDecoration: 'none' }}
                                >
                                  <Badge
                                    mr={2}
                                    mb={1}
                                    _hover={{ bg: 'blue.100' }}
                                    transition="background-color 0.2s"
                                  >
                                    {dep}
                                  </Badge>
                                </Link>
                              );
                            })}
                          </Box>
                        </Box>
                      )}
                    </Box>
                  ))}
                </VStack>
              </Box>
            </VStack>
          </TabPanel>

          <TabPanel>
            {selectedVersionData && (
              <ProtoTypes
                files={selectedVersionData.files as unknown as ProtoFile[]}
                moduleName={module.name}
                version={selectedVersionData.version}
              />
            )}
          </TabPanel>

          <TabPanel>
            {selectedVersionData && (
              <ApiExplorer
                files={selectedVersionData.files as unknown as ProtoFile[]}
                moduleName={module.name}
                version={selectedVersionData.version}
              />
            )}
          </TabPanel>

          <TabPanel>
            {selectedVersionData && (
              <CodeExamples
                moduleName={module.name}
                version={selectedVersionData.version}
              />
            )}
          </TabPanel>
        </TabPanels>
      </Tabs>
    </VStack>
  );
}; 