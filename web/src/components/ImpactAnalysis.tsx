import React, { useEffect, useState } from 'react';
import {
  Box,
  VStack,
  HStack,
  Text,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  Badge,
  Divider,
  Spinner,
  Center,
  Link,
  Wrap,
  WrapItem,
  Heading,
  Accordion,
  AccordionItem,
  AccordionButton,
  AccordionPanel,
  AccordionIcon,
  Code,
} from '@chakra-ui/react';
import { WarningIcon, CheckCircleIcon, WarningTwoIcon } from '@chakra-ui/icons';
import { Link as RouterLink } from 'react-router-dom';

interface Dependency {
  module: string;
  version: string;
  type: string;
}

interface ImpactAnalysisData {
  module: string;
  version: string;
  direct_dependents: Dependency[];
  transitive_dependents: Dependency[];
  total_impact: number;
}

interface ImpactAnalysisProps {
  moduleName: string;
  version: string;
}

export const ImpactAnalysis: React.FC<ImpactAnalysisProps> = ({ moduleName, version }) => {
  const [data, setData] = useState<ImpactAnalysisData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchImpact = async () => {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch(`/modules/${moduleName}/versions/${version}/impact`);
        if (!response.ok) {
          throw new Error(`Failed to fetch impact: ${response.statusText}`);
        }

        const impactData: ImpactAnalysisData = await response.json();
        setData(impactData);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch impact analysis');
        console.error('Impact analysis error:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchImpact();
  }, [moduleName, version]);

  if (loading) {
    return (
      <Center minH="300px">
        <VStack spacing={4}>
          <Spinner size="xl" color="blue.500" />
          <Text color="gray.600">Analyzing impact...</Text>
        </VStack>
      </Center>
    );
  }

  if (error) {
    return (
      <Alert status="error">
        <AlertIcon />
        <AlertTitle>Failed to load impact analysis</AlertTitle>
        <AlertDescription>{error}</AlertDescription>
      </Alert>
    );
  }

  if (!data) {
    return (
      <Alert status="info">
        <AlertIcon />
        <AlertTitle>No impact data available</AlertTitle>
        <AlertDescription>Unable to analyze the impact of changes to this module.</AlertDescription>
      </Alert>
    );
  }

  // Determine severity level
  const getSeverity = () => {
    if (data.total_impact === 0) return 'success';
    if (data.total_impact <= 5) return 'info';
    if (data.total_impact <= 10) return 'warning';
    return 'error';
  };

  const severity = getSeverity();

  const getIcon = () => {
    switch (severity) {
      case 'success':
        return <CheckCircleIcon />;
      case 'warning':
        return <WarningIcon />;
      case 'error':
        return <WarningTwoIcon />;
      default:
        return <AlertIcon />;
    }
  };

  const getSeverityMessage = () => {
    if (data.total_impact === 0) {
      return 'No modules depend on this version. Changes are safe.';
    }
    if (data.total_impact <= 5) {
      return 'Low impact - A small number of modules will be affected.';
    }
    if (data.total_impact <= 10) {
      return 'Medium impact - Several modules will need updates.';
    }
    return 'High impact - Many modules depend on this. Breaking changes should be avoided.';
  };

  return (
    <VStack align="stretch" spacing={6}>
      {/* Impact Summary */}
      <Alert status={severity} variant="left-accent">
        {getIcon()}
        <Box flex={1}>
          <AlertTitle>
            Impact Analysis for {moduleName}@{version}
          </AlertTitle>
          <AlertDescription>
            <Text mt={2}>{getSeverityMessage()}</Text>
            <HStack mt={3} spacing={4}>
              <Badge colorScheme={severity === 'success' ? 'green' : severity === 'error' ? 'red' : 'yellow'} fontSize="md">
                Total Impact: {data.total_impact} {data.total_impact === 1 ? 'module' : 'modules'}
              </Badge>
              {data.direct_dependents.length > 0 && (
                <Badge colorScheme="orange" fontSize="sm">
                  {data.direct_dependents.length} direct
                </Badge>
              )}
              {data.transitive_dependents.length > 0 && (
                <Badge colorScheme="yellow" fontSize="sm">
                  {data.transitive_dependents.length} transitive
                </Badge>
              )}
            </HStack>
          </AlertDescription>
        </Box>
      </Alert>

      {/* Breaking Change Warning */}
      {data.total_impact > 0 && (
        <Alert status="warning" variant="subtle">
          <AlertIcon />
          <Box>
            <AlertTitle fontSize="md">Breaking Changes Warning</AlertTitle>
            <AlertDescription fontSize="sm">
              Making breaking changes to this module will require updates to {data.total_impact}{' '}
              {data.total_impact === 1 ? 'module' : 'modules'}. Consider:
              <VStack align="stretch" mt={2} spacing={1} fontSize="sm">
                <Text>• Publishing a new major version instead of updating this version</Text>
                <Text>• Maintaining backward compatibility with deprecated fields</Text>
                <Text>• Coordinating with dependent module owners</Text>
                <Text>• Providing migration guides and examples</Text>
              </VStack>
            </AlertDescription>
          </Box>
        </Alert>
      )}

      {/* Direct Dependents */}
      {data.direct_dependents.length > 0 && (
        <Box>
          <Heading size="md" mb={4}>
            Direct Dependents ({data.direct_dependents.length})
          </Heading>
          <Text fontSize="sm" color="gray.600" mb={3}>
            These modules directly import {moduleName}@{version}. They will be immediately affected by changes.
          </Text>
          <Wrap spacing={2}>
            {data.direct_dependents.map((dep, index) => (
              <WrapItem key={index}>
                <Link
                  as={RouterLink}
                  to={`/modules/${dep.module}?version=${dep.version}`}
                  _hover={{ textDecoration: 'none' }}
                >
                  <Badge
                    colorScheme="orange"
                    fontSize="sm"
                    p={2}
                    borderRadius="md"
                    _hover={{ bg: 'orange.100', transform: 'scale(1.05)' }}
                    transition="all 0.2s"
                    cursor="pointer"
                  >
                    {dep.module}@{dep.version}
                  </Badge>
                </Link>
              </WrapItem>
            ))}
          </Wrap>
        </Box>
      )}

      <Divider />

      {/* Transitive Dependents */}
      {data.transitive_dependents.length > 0 && (
        <Box>
          <Heading size="md" mb={4}>
            Transitive Dependents ({data.transitive_dependents.length})
          </Heading>
          <Text fontSize="sm" color="gray.600" mb={3}>
            These modules indirectly depend on {moduleName}@{version} through other dependencies.
          </Text>
          
          {data.transitive_dependents.length > 10 ? (
            <Accordion allowToggle>
              <AccordionItem>
                <h2>
                  <AccordionButton>
                    <Box flex="1" textAlign="left" fontSize="sm">
                      <Badge colorScheme="yellow" mr={2}>
                        {data.transitive_dependents.length}
                      </Badge>
                      Click to view all transitive dependents
                    </Box>
                    <AccordionIcon />
                  </AccordionButton>
                </h2>
                <AccordionPanel pb={4}>
                  <Wrap spacing={2}>
                    {data.transitive_dependents.map((dep, index) => (
                      <WrapItem key={index}>
                        <Link
                          as={RouterLink}
                          to={`/modules/${dep.module}?version=${dep.version}`}
                          _hover={{ textDecoration: 'none' }}
                        >
                          <Badge
                            colorScheme="yellow"
                            fontSize="xs"
                            p={2}
                            borderRadius="md"
                            _hover={{ bg: 'yellow.100', transform: 'scale(1.05)' }}
                            transition="all 0.2s"
                            cursor="pointer"
                          >
                            {dep.module}@{dep.version}
                          </Badge>
                        </Link>
                      </WrapItem>
                    ))}
                  </Wrap>
                </AccordionPanel>
              </AccordionItem>
            </Accordion>
          ) : (
            <Wrap spacing={2}>
              {data.transitive_dependents.map((dep, index) => (
                <WrapItem key={index}>
                  <Link
                    as={RouterLink}
                    to={`/modules/${dep.module}?version=${dep.version}`}
                    _hover={{ textDecoration: 'none' }}
                  >
                    <Badge
                      colorScheme="yellow"
                      fontSize="sm"
                      p={2}
                      borderRadius="md"
                      _hover={{ bg: 'yellow.100', transform: 'scale(1.05)' }}
                      transition="all 0.2s"
                      cursor="pointer"
                    >
                      {dep.module}@{dep.version}
                    </Badge>
                  </Link>
                </WrapItem>
              ))}
            </Wrap>
          )}
        </Box>
      )}

      {/* No Impact Case */}
      {data.total_impact === 0 && (
        <Box bg="green.50" p={6} borderRadius="md" borderWidth={1} borderColor="green.200">
          <VStack spacing={3}>
            <CheckCircleIcon boxSize={12} color="green.500" />
            <Heading size="md" color="green.700">
              Safe to Modify
            </Heading>
            <Text textAlign="center" color="gray.700" fontSize="sm">
              No other modules depend on this version. You can make breaking changes without affecting other parts of the system.
            </Text>
            <Text textAlign="center" color="gray.600" fontSize="xs" fontStyle="italic">
              This is a good opportunity to refactor or make significant improvements.
            </Text>
          </VStack>
        </Box>
      )}

      {/* Best Practices */}
      {data.total_impact > 0 && (
        <Box bg="blue.50" p={4} borderRadius="md" borderWidth={1} borderColor="blue.200">
          <Heading size="sm" mb={3} color="blue.700">
            Best Practices for Schema Changes
          </Heading>
          <VStack align="stretch" spacing={2} fontSize="sm" color="gray.700">
            <HStack align="start">
              <Text fontWeight="bold" minW="140px">Additive Changes:</Text>
              <Text>Add new fields, messages, or services (backward compatible)</Text>
            </HStack>
            <HStack align="start">
              <Text fontWeight="bold" minW="140px">Field Numbers:</Text>
              <Text>Never reuse or change field numbers (breaks binary compatibility)</Text>
            </HStack>
            <HStack align="start">
              <Text fontWeight="bold" minW="140px">Deprecation:</Text>
              <Text>Mark fields as deprecated before removing in a major version</Text>
            </HStack>
            <HStack align="start">
              <Text fontWeight="bold" minW="140px">New Versions:</Text>
              <Text>Create a new major version for breaking changes (e.g., v2.0.0)</Text>
            </HStack>
            <HStack align="start">
              <Text fontWeight="bold" minW="140px">Testing:</Text>
              <Text>Test with all direct dependents before publishing changes</Text>
            </HStack>
          </VStack>
        </Box>
      )}

      {/* API Endpoint Reference */}
      <Box bg="gray.50" p={4} borderRadius="md" borderWidth={1}>
        <Text fontSize="xs" color="gray.600" fontWeight="bold" mb={2}>
          API Endpoint
        </Text>
        <Code fontSize="xs" p={2} borderRadius="md" display="block">
          GET /modules/{moduleName}/versions/{version}/impact
        </Code>
      </Box>
    </VStack>
  );
};
