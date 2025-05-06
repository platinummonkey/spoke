import React from 'react';
import { Box, Heading, VStack, Text, Link, Spinner, Alert, AlertIcon, Button, Badge } from '@chakra-ui/react';
import { Module } from '../types';
import { Link as RouterLink } from 'react-router-dom';

interface ModuleListProps {
  modules: Module[];
  loading: boolean;
  error: string | null;
  retry: () => void;
}

export const ModuleList: React.FC<ModuleListProps> = ({ modules = [], loading, error, retry }) => {
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
          {error}
        </Alert>
        <Button onClick={retry} colorScheme="blue">
          Retry
        </Button>
      </Box>
    );
  }

  return (
    <Box p={4}>
      <Heading size="lg" mb={4}>Available Modules</Heading>
      {!modules || modules.length === 0 ? (
        <Text>No modules available</Text>
      ) : (
        <VStack align="stretch" spacing={4}>
          {modules.map((module) => {
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
                <Box mt={2} display="flex" alignItems="center" gap={2}>
                  <Text fontSize="sm" color="gray.500">
                    {module.versions?.length || 0} version{(module.versions?.length || 0) !== 1 ? 's' : ''}
                  </Text>
                  {latestVersion && (
                    <Badge colorScheme="blue">
                      Latest: {latestVersion.version}
                    </Badge>
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