import React, { useState } from 'react';
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
} from '@chakra-ui/react';
import { Module } from '../types';
import { ProtoTypes } from './ProtoTypes';

interface ModuleDetailProps {
  module: Module | null;
  loading: boolean;
  error: Error | null;
  retry: () => void;
}

export const ModuleDetail: React.FC<ModuleDetailProps> = ({ module, loading, error, retry }) => {
  const [selectedVersion, setSelectedVersion] = useState<string | null>(null);

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

  return (
    <VStack align="stretch" spacing={6}>
      <Box>
        <Heading size="lg">{module.name}</Heading>
        <Text mt={2} color="gray.600">
          {module.description || 'No description available'}
        </Text>
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
                      <Text fontWeight="bold">{version.version}</Text>
                      <Text fontSize="sm" color="gray.600">
                        {version.files?.length || 0} files
                      </Text>
                      {version.dependencies?.length > 0 && (
                        <Box mt={2}>
                          <Text fontSize="sm" fontWeight="medium">
                            Dependencies:
                          </Text>
                          <Box mt={1}>
                            {version.dependencies.map((dep) => (
                              <Badge key={dep} mr={2} mb={1}>
                                {dep}
                              </Badge>
                            ))}
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