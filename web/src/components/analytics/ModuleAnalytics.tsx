import React from 'react';
import {
  Box,
  VStack,
  HStack,
  Heading,
  Text,
  Badge,
  Stat,
  StatLabel,
  StatNumber,
  StatHelpText,
  Grid,
  GridItem,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  List,
  ListItem,
  ListIcon,
  Spinner,
  Progress,
  useColorModeValue,
} from '@chakra-ui/react';
import { InfoIcon, WarningIcon, CheckCircleIcon } from '@chakra-ui/icons';
import { useModuleHealth } from '../../hooks/useAnalytics';

interface ModuleAnalyticsProps {
  moduleName: string;
  version?: string;
}

export const ModuleAnalytics: React.FC<ModuleAnalyticsProps> = ({ moduleName, version }) => {
  const { data: health, isLoading, error } = useModuleHealth(moduleName, version);

  const bgColor = useColorModeValue('white', 'gray.800');
  const borderColor = useColorModeValue('gray.200', 'gray.700');

  if (isLoading) {
    return (
      <Box textAlign="center" py={10}>
        <Spinner size="lg" />
        <Text mt={4}>Loading health assessment...</Text>
      </Box>
    );
  }

  if (error) {
    return (
      <Alert status="error">
        <AlertIcon />
        Failed to load health assessment. Please try again later.
      </Alert>
    );
  }

  if (!health) return null;

  const getHealthColor = (score: number): string => {
    if (score >= 80) return 'green';
    if (score >= 60) return 'yellow';
    if (score >= 40) return 'orange';
    return 'red';
  };

  const getHealthStatus = (score: number): 'success' | 'warning' | 'error' => {
    if (score >= 80) return 'success';
    if (score >= 60) return 'warning';
    return 'error';
  };

  const getHealthLabel = (score: number): string => {
    if (score >= 80) return 'Excellent';
    if (score >= 60) return 'Good';
    if (score >= 40) return 'Fair';
    return 'Needs Attention';
  };

  const healthColor = getHealthColor(health.health_score);
  const healthStatus = getHealthStatus(health.health_score);

  return (
    <VStack align="stretch" spacing={6}>
      <Heading size="md">Schema Health Assessment</Heading>

      {/* Overall Health Score */}
      <Alert
        status={healthStatus}
        borderRadius="md"
        flexDirection="column"
        alignItems="flex-start"
      >
        <HStack mb={2}>
          <AlertIcon />
          <AlertTitle>
            Health Score: {health.health_score.toFixed(1)}/100
          </AlertTitle>
        </HStack>
        <Box width="100%" mb={2}>
          <Progress
            value={health.health_score}
            colorScheme={healthColor}
            size="lg"
            borderRadius="md"
          />
        </Box>
        <AlertDescription>
          Status: <strong>{getHealthLabel(health.health_score)}</strong>
          {' - '}
          {health.health_score >= 80 && 'Your schema is in excellent health!'}
          {health.health_score >= 60 && health.health_score < 80 && 'Good health with room for improvement.'}
          {health.health_score < 60 && 'This schema needs attention.'}
        </AlertDescription>
      </Alert>

      {/* Metrics Grid */}
      <Grid templateColumns="repeat(auto-fit, minmax(200px, 1fr))" gap={4}>
        <GridItem>
          <Box
            p={4}
            bg={bgColor}
            borderWidth={1}
            borderColor={borderColor}
            borderRadius="md"
          >
            <Stat>
              <StatLabel>Complexity Score</StatLabel>
              <StatNumber>
                <Badge colorScheme={getHealthColor(100 - health.complexity_score)} fontSize="xl">
                  {health.complexity_score.toFixed(0)}/100
                </Badge>
              </StatNumber>
              <StatHelpText>Lower is better</StatHelpText>
            </Stat>
          </Box>
        </GridItem>

        <GridItem>
          <Box
            p={4}
            bg={bgColor}
            borderWidth={1}
            borderColor={borderColor}
            borderRadius="md"
          >
            <Stat>
              <StatLabel>Maintainability</StatLabel>
              <StatNumber>
                <Badge colorScheme={getHealthColor(health.maintainability_index)} fontSize="xl">
                  {health.maintainability_index.toFixed(0)}/100
                </Badge>
              </StatNumber>
              <StatHelpText>Higher is better</StatHelpText>
            </Stat>
          </Box>
        </GridItem>

        <GridItem>
          <Box
            p={4}
            bg={bgColor}
            borderWidth={1}
            borderColor={borderColor}
            borderRadius="md"
          >
            <Stat>
              <StatLabel>Unused Fields</StatLabel>
              <StatNumber>{health.unused_fields.length}</StatNumber>
              <StatHelpText>
                {health.unused_fields.length === 0 ? 'None found' : 'Need review'}
              </StatHelpText>
            </Stat>
          </Box>
        </GridItem>

        <GridItem>
          <Box
            p={4}
            bg={bgColor}
            borderWidth={1}
            borderColor={borderColor}
            borderRadius="md"
          >
            <Stat>
              <StatLabel>Deprecated Fields</StatLabel>
              <StatNumber>{health.deprecated_field_count}</StatNumber>
              <StatHelpText>
                {health.deprecated_field_count === 0 ? 'None' : 'Clean up needed'}
              </StatHelpText>
            </Stat>
          </Box>
        </GridItem>

        <GridItem>
          <Box
            p={4}
            bg={bgColor}
            borderWidth={1}
            borderColor={borderColor}
            borderRadius="md"
          >
            <Stat>
              <StatLabel>Breaking Changes (30d)</StatLabel>
              <StatNumber>{health.breaking_changes_30d}</StatNumber>
              <StatHelpText>
                {health.breaking_changes_30d === 0 ? 'Stable' : 'Active changes'}
              </StatHelpText>
            </Stat>
          </Box>
        </GridItem>

        <GridItem>
          <Box
            p={4}
            bg={bgColor}
            borderWidth={1}
            borderColor={borderColor}
            borderRadius="md"
          >
            <Stat>
              <StatLabel>Dependents</StatLabel>
              <StatNumber>{health.dependents_count}</StatNumber>
              <StatHelpText>Modules using this</StatHelpText>
            </Stat>
          </Box>
        </GridItem>
      </Grid>

      {/* Recommendations */}
      {health.recommendations && health.recommendations.length > 0 && (
        <Box
          p={4}
          bg={bgColor}
          borderWidth={1}
          borderColor={borderColor}
          borderRadius="md"
        >
          <Heading size="sm" mb={3}>
            <HStack>
              <InfoIcon color="blue.500" />
              <Text>Recommendations</Text>
            </HStack>
          </Heading>
          <List spacing={2}>
            {health.recommendations.map((rec, idx) => (
              <ListItem key={idx} display="flex" alignItems="flex-start">
                <ListIcon
                  as={health.health_score > 80 ? CheckCircleIcon : InfoIcon}
                  color={health.health_score > 80 ? 'green.500' : 'blue.500'}
                  mt={1}
                />
                <Text flex={1}>{rec}</Text>
              </ListItem>
            ))}
          </List>
        </Box>
      )}

      {/* Unused Fields Warning */}
      {health.unused_fields && health.unused_fields.length > 0 && (
        <Alert status="warning" borderRadius="md">
          <AlertIcon />
          <Box flex="1">
            <AlertTitle>Unused Fields Detected</AlertTitle>
            <AlertDescription>
              {health.unused_fields.length} fields have no recorded usage in the last 90 days:
              <Text mt={2} fontFamily="mono" fontSize="sm">
                {health.unused_fields.slice(0, 5).join(', ')}
                {health.unused_fields.length > 5 && ` (and ${health.unused_fields.length - 5} more)`}
              </Text>
            </AlertDescription>
          </Box>
        </Alert>
      )}

      {/* High Impact Warning */}
      {health.dependents_count > 10 && health.breaking_changes_30d > 0 && (
        <Alert status="warning" borderRadius="md">
          <AlertIcon as={WarningIcon} />
          <Box flex="1">
            <AlertTitle>High Impact Module</AlertTitle>
            <AlertDescription>
              This module has {health.dependents_count} dependents and {health.breaking_changes_30d} breaking
              change(s) in the last 30 days. Coordinate changes carefully with downstream consumers.
            </AlertDescription>
          </Box>
        </Alert>
      )}
    </VStack>
  );
};
