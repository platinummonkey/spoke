import React, { useState, useEffect } from 'react';
import {
  Box,
  VStack,
  HStack,
  Text,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  Accordion,
  AccordionItem,
  AccordionButton,
  AccordionPanel,
  AccordionIcon,
  Badge,
  Code,
  Spinner,
  Divider,
} from '@chakra-ui/react';

interface Change {
  type: string;
  severity: 'breaking' | 'non_breaking' | 'warning';
  location: string;
  old_value?: string;
  new_value?: string;
  description: string;
  migration_tip?: string;
}

interface DiffResult {
  from_version: string;
  to_version: string;
  changes: Change[];
}

interface SchemaDiffProps {
  moduleName: string;
  fromVersion: string;
  toVersion: string;
}

const getSeverityColor = (severity: string): string => {
  switch (severity) {
    case 'breaking':
      return 'red';
    case 'non_breaking':
      return 'green';
    case 'warning':
      return 'yellow';
    default:
      return 'gray';
  }
};

const getSeverityLabel = (severity: string): string => {
  switch (severity) {
    case 'breaking':
      return 'Breaking';
    case 'non_breaking':
      return 'Non-Breaking';
    case 'warning':
      return 'Warning';
    default:
      return severity;
  }
};

const getChangeTypeLabel = (type: string): string => {
  return type
    .split('_')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
};

export const SchemaDiff: React.FC<SchemaDiffProps> = ({
  moduleName,
  fromVersion,
  toVersion,
}) => {
  const [diff, setDiff] = useState<DiffResult | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchDiff = async () => {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch(`/api/v1/modules/${moduleName}/diff`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            from_version: fromVersion,
            to_version: toVersion,
          }),
        });

        if (!response.ok) {
          throw new Error(`Failed to fetch diff: ${response.statusText}`);
        }

        const result = await response.json();
        setDiff(result);
      } catch (err) {
        console.error('Error fetching diff:', err);
        setError(err instanceof Error ? err.message : 'Failed to fetch diff');
      } finally {
        setLoading(false);
      }
    };

    if (fromVersion && toVersion && fromVersion !== toVersion) {
      fetchDiff();
    }
  }, [moduleName, fromVersion, toVersion]);

  if (loading) {
    return (
      <Box textAlign="center" py={10}>
        <Spinner size="lg" />
        <Text mt={4} color="gray.600">
          Analyzing changes between v{fromVersion} and v{toVersion}...
        </Text>
      </Box>
    );
  }

  if (error) {
    return (
      <Alert status="error">
        <AlertIcon />
        <AlertTitle>Error</AlertTitle>
        <AlertDescription>{error}</AlertDescription>
      </Alert>
    );
  }

  if (!diff) {
    return (
      <Alert status="info">
        <AlertIcon />
        <Text>Select two different versions to compare.</Text>
      </Alert>
    );
  }

  if (!diff.changes || diff.changes.length === 0) {
    return (
      <Alert status="success">
        <AlertIcon />
        <AlertTitle>No Changes Detected</AlertTitle>
        <AlertDescription>
          Versions {fromVersion} and {toVersion} are identical.
        </AlertDescription>
      </Alert>
    );
  }

  const breakingChanges = diff.changes.filter(c => c.severity === 'breaking');
  const nonBreakingChanges = diff.changes.filter(c => c.severity === 'non_breaking');
  const warnings = diff.changes.filter(c => c.severity === 'warning');

  return (
    <VStack align="stretch" spacing={6}>
      {/* Summary Alert */}
      {breakingChanges.length > 0 && (
        <Alert status="error" variant="left-accent">
          <AlertIcon />
          <Box flex="1">
            <AlertTitle fontSize="md">
              {breakingChanges.length} Breaking Change{breakingChanges.length > 1 ? 's' : ''} Detected
            </AlertTitle>
            <AlertDescription fontSize="sm">
              Upgrading from v{fromVersion} to v{toVersion} requires code changes to handle breaking changes.
            </AlertDescription>
          </Box>
        </Alert>
      )}

      {breakingChanges.length === 0 && nonBreakingChanges.length > 0 && (
        <Alert status="success" variant="left-accent">
          <AlertIcon />
          <Box flex="1">
            <AlertTitle fontSize="md">Backward Compatible</AlertTitle>
            <AlertDescription fontSize="sm">
              All changes are non-breaking. Upgrade from v{fromVersion} to v{toVersion} is safe.
            </AlertDescription>
          </Box>
        </Alert>
      )}

      {/* Statistics */}
      <HStack spacing={4} p={4} bg="gray.50" borderRadius="md">
        <Box textAlign="center" flex={1}>
          <Text fontSize="2xl" fontWeight="bold" color="red.600">
            {breakingChanges.length}
          </Text>
          <Text fontSize="sm" color="gray.600">
            Breaking
          </Text>
        </Box>
        <Divider orientation="vertical" height="50px" />
        <Box textAlign="center" flex={1}>
          <Text fontSize="2xl" fontWeight="bold" color="green.600">
            {nonBreakingChanges.length}
          </Text>
          <Text fontSize="sm" color="gray.600">
            Non-Breaking
          </Text>
        </Box>
        <Divider orientation="vertical" height="50px" />
        <Box textAlign="center" flex={1}>
          <Text fontSize="2xl" fontWeight="bold" color="yellow.600">
            {warnings.length}
          </Text>
          <Text fontSize="sm" color="gray.600">
            Warnings
          </Text>
        </Box>
        <Divider orientation="vertical" height="50px" />
        <Box textAlign="center" flex={1}>
          <Text fontSize="2xl" fontWeight="bold">
            {diff.changes.length}
          </Text>
          <Text fontSize="sm" color="gray.600">
            Total Changes
          </Text>
        </Box>
      </HStack>

      {/* Changes List */}
      <Box>
        <Text fontWeight="bold" mb={3}>
          Detailed Changes
        </Text>
        <Accordion allowMultiple defaultIndex={breakingChanges.length > 0 ? [0] : []}>
          {diff.changes.map((change, index) => (
            <AccordionItem key={index}>
              <AccordionButton>
                <Box flex="1" textAlign="left">
                  <HStack spacing={3}>
                    <Badge colorScheme={getSeverityColor(change.severity)} fontSize="sm">
                      {getSeverityLabel(change.severity)}
                    </Badge>
                    <Badge variant="outline" fontSize="sm">
                      {getChangeTypeLabel(change.type)}
                    </Badge>
                    <Text fontSize="sm" fontWeight="medium">
                      {change.description}
                    </Text>
                  </HStack>
                </Box>
                <AccordionIcon />
              </AccordionButton>
              <AccordionPanel pb={4}>
                <VStack align="stretch" spacing={3}>
                  {/* Location */}
                  <Box>
                    <Text fontSize="sm" fontWeight="medium" mb={1}>
                      Location:
                    </Text>
                    <Code fontSize="sm" p={2} borderRadius="md" display="block">
                      {change.location}
                    </Code>
                  </Box>

                  {/* Old/New Values */}
                  {(change.old_value || change.new_value) && (
                    <HStack align="start" spacing={4}>
                      {change.old_value && (
                        <Box flex={1}>
                          <Text fontSize="sm" fontWeight="medium" mb={1}>
                            Old:
                          </Text>
                          <Code
                            fontSize="sm"
                            p={2}
                            borderRadius="md"
                            display="block"
                            bg="red.50"
                            borderColor="red.200"
                            borderWidth={1}
                          >
                            {change.old_value}
                          </Code>
                        </Box>
                      )}
                      {change.new_value && (
                        <Box flex={1}>
                          <Text fontSize="sm" fontWeight="medium" mb={1}>
                            New:
                          </Text>
                          <Code
                            fontSize="sm"
                            p={2}
                            borderRadius="md"
                            display="block"
                            bg="green.50"
                            borderColor="green.200"
                            borderWidth={1}
                          >
                            {change.new_value}
                          </Code>
                        </Box>
                      )}
                    </HStack>
                  )}

                  {/* Migration Tip */}
                  {change.migration_tip && (
                    <Box p={3} bg="blue.50" borderRadius="md" borderWidth={1} borderColor="blue.200">
                      <Text fontSize="sm" fontWeight="medium" mb={1} color="blue.700">
                        💡 Migration Tip:
                      </Text>
                      <Text fontSize="sm" color="blue.900">
                        {change.migration_tip}
                      </Text>
                    </Box>
                  )}
                </VStack>
              </AccordionPanel>
            </AccordionItem>
          ))}
        </Accordion>
      </Box>
    </VStack>
  );
};
