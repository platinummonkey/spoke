import React from 'react';
import {
  Box,
  VStack,
  Heading,
  Text,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
} from '@chakra-ui/react';
import { ProtoFile, Service } from '../types';
import { ServiceBrowser } from './ServiceBrowser';

interface ApiExplorerProps {
  files: ProtoFile[];
  moduleName: string;
  version: string;
}

// Extract all services from proto files
const extractServices = (files: ProtoFile[]): Service[] => {
  const services: Service[] = [];

  for (const file of files) {
    if (file.services && file.services.length > 0) {
      services.push(...file.services);
    }
  }

  return services;
};

export const ApiExplorer: React.FC<ApiExplorerProps> = ({ files, moduleName, version }) => {
  const services = extractServices(files);

  return (
    <VStack align="stretch" spacing={6}>
      {/* Header */}
      <Box>
        <Heading size="md" mb={2}>
          API Explorer
        </Heading>
        <Text color="gray.600" fontSize="sm">
          Interactive browser for gRPC services in {moduleName} (version {version})
        </Text>
      </Box>

      {/* Info Box */}
      {services.length > 0 && (
        <Alert status="info" variant="left-accent">
          <AlertIcon />
          <Box>
            <AlertTitle fontSize="sm">How to use</AlertTitle>
            <AlertDescription fontSize="sm">
              Expand services to view their methods. Click on any method to see request and response message structures.
              Use this to understand the API before generating code examples.
            </AlertDescription>
          </Box>
        </Alert>
      )}

      {/* Service Browser */}
      <ServiceBrowser services={services} files={files} />

      {/* Empty State with helpful message */}
      {services.length === 0 && (
        <Alert status="warning">
          <AlertIcon />
          <Box>
            <AlertTitle>No gRPC Services Found</AlertTitle>
            <AlertDescription>
              This module contains only message and enum type definitions.
              Switch to the "Types" tab to browse available types, or check the "Usage Examples"
              tab to see how to use these types in your code.
            </AlertDescription>
          </Box>
        </Alert>
      )}
    </VStack>
  );
};
