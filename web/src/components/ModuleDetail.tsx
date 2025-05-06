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
} from '@chakra-ui/react';
import { ChevronRightIcon, ChevronDownIcon } from '@chakra-ui/icons';
import { Link as RouterLink } from 'react-router-dom';
import { Module } from '../types';
import { ProtoTypes } from './ProtoTypes';

interface ModuleDetailProps {
  module: Module | null;
  loading: boolean;
  error: Error | null;
  retry: () => void;
  initialVersion?: string;
}

export const ModuleDetail: React.FC<ModuleDetailProps> = ({ module, loading, error, retry, initialVersion }) => {
  const [selectedVersion, setSelectedVersion] = useState<string | null>(null);

  // Update selectedVersion when module or initialVersion changes
  useEffect(() => {
    if (!module) return;
    
    if (initialVersion) {
      setSelectedVersion(initialVersion);
    } else if (module.versions && module.versions.length > 0) {
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
        </TabList>

        <TabPanels>
          <TabPanel>
            <VStack align="stretch" spacing={4}>
              <Box>
                <Heading size="md" mb={2}>Versions</Heading>
                <VStack align="stretch" spacing={2}>
                  {module.versions?.map((version) => (
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
                      {version.dependencies?.length > 0 && (
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
                files={selectedVersionData.files}
                moduleName={module.name}
                version={selectedVersionData.version}
                versions={versions}
                onVersionChange={setSelectedVersion}
              />
            )}
          </TabPanel>
        </TabPanels>
      </Tabs>
    </VStack>
  );
}; 