import React, { useState } from 'react';
import {
  Box,
  VStack,
  HStack,
  Heading,
  Text,
  Stat,
  StatLabel,
  StatNumber,
  StatHelpText,
  Grid,
  GridItem,
  Spinner,
  Alert,
  AlertIcon,
  useColorModeValue,
  Select,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
} from '@chakra-ui/react';
import {
  useAnalyticsOverview,
  usePopularModules,
  useTrendingModules,
} from '../../hooks/useAnalytics';
import { DownloadChart } from './DownloadChart';
import { LanguageChart } from './LanguageChart';
import { TopModulesChart } from './TopModulesChart';

export const AnalyticsDashboard: React.FC = () => {
  const [period, setPeriod] = useState('30d');

  const { data: overview, isLoading: overviewLoading, error: overviewError } = useAnalyticsOverview();
  const { data: popularModules, isLoading: popularLoading } = usePopularModules(period, 10);
  const { data: trendingModules, isLoading: trendingLoading } = useTrendingModules(10);

  const bgColor = useColorModeValue('white', 'gray.800');
  const borderColor = useColorModeValue('gray.200', 'gray.700');

  if (overviewLoading) {
    return (
      <Box textAlign="center" py={10}>
        <Spinner size="xl" />
        <Text mt={4}>Loading analytics...</Text>
      </Box>
    );
  }

  if (overviewError) {
    return (
      <Alert status="error">
        <AlertIcon />
        Failed to load analytics. Please try again later.
      </Alert>
    );
  }

  if (!overview) return null;

  return (
    <VStack align="stretch" spacing={6} p={6}>
      <HStack justify="space-between" align="center">
        <Heading size="lg">Analytics Dashboard</Heading>
        <Select
          value={period}
          onChange={(e) => setPeriod(e.target.value)}
          maxW="150px"
        >
          <option value="7d">Last 7 days</option>
          <option value="30d">Last 30 days</option>
          <option value="90d">Last 90 days</option>
        </Select>
      </HStack>

      {/* KPI Cards */}
      <Grid templateColumns="repeat(auto-fit, minmax(250px, 1fr))" gap={4}>
        <GridItem>
          <Stat
            p={4}
            bg={bgColor}
            borderWidth={1}
            borderColor={borderColor}
            borderRadius="md"
          >
            <StatLabel>Total Modules</StatLabel>
            <StatNumber>{overview.total_modules.toLocaleString()}</StatNumber>
            <StatHelpText>{overview.total_versions.toLocaleString()} versions</StatHelpText>
          </Stat>
        </GridItem>

        <GridItem>
          <Stat
            p={4}
            bg={bgColor}
            borderWidth={1}
            borderColor={borderColor}
            borderRadius="md"
          >
            <StatLabel>Downloads (24h)</StatLabel>
            <StatNumber>{overview.total_downloads_24h.toLocaleString()}</StatNumber>
            <StatHelpText>{overview.total_downloads_30d.toLocaleString()} in 30d</StatHelpText>
          </Stat>
        </GridItem>

        <GridItem>
          <Stat
            p={4}
            bg={bgColor}
            borderWidth={1}
            borderColor={borderColor}
            borderRadius="md"
          >
            <StatLabel>Active Users</StatLabel>
            <StatNumber>{overview.active_users_24h.toLocaleString()}</StatNumber>
            <StatHelpText>{overview.active_users_7d.toLocaleString()} in 7d</StatHelpText>
          </Stat>
        </GridItem>

        <GridItem>
          <Stat
            p={4}
            bg={bgColor}
            borderWidth={1}
            borderColor={borderColor}
            borderRadius="md"
          >
            <StatLabel>Cache Hit Rate</StatLabel>
            <StatNumber>{(overview.cache_hit_rate * 100).toFixed(1)}%</StatNumber>
            <StatHelpText>
              {overview.cache_hit_rate > 0.7 ? 'Good' : 'Low'}
            </StatHelpText>
          </Stat>
        </GridItem>

        <GridItem>
          <Stat
            p={4}
            bg={bgColor}
            borderWidth={1}
            borderColor={borderColor}
            borderRadius="md"
          >
            <StatLabel>Top Language</StatLabel>
            <StatNumber textTransform="uppercase">{overview.top_language || 'N/A'}</StatNumber>
            <StatHelpText>Most compiled</StatHelpText>
          </Stat>
        </GridItem>

        <GridItem>
          <Stat
            p={4}
            bg={bgColor}
            borderWidth={1}
            borderColor={borderColor}
            borderRadius="md"
          >
            <StatLabel>Avg Compilation</StatLabel>
            <StatNumber>{Math.round(overview.avg_compilation_ms)}ms</StatNumber>
            <StatHelpText>Per compilation</StatHelpText>
          </Stat>
        </GridItem>
      </Grid>

      {/* Charts */}
      <Tabs variant="enclosed">
        <TabList>
          <Tab>Downloads</Tab>
          <Tab>Popular Modules</Tab>
          <Tab>Trending</Tab>
          <Tab>Languages</Tab>
        </TabList>

        <TabPanels>
          {/* Downloads Tab */}
          <TabPanel>
            <Box p={4} bg={bgColor} borderWidth={1} borderRadius="md" minH="400px">
              <Heading size="md" mb={4}>Download Trends</Heading>
              <DownloadChart period={period} />
            </Box>
          </TabPanel>

          {/* Popular Modules Tab */}
          <TabPanel>
            <Box p={4} bg={bgColor} borderWidth={1} borderRadius="md" minH="400px">
              <Heading size="md" mb={4}>Top 10 Modules by Downloads</Heading>
              {popularLoading ? (
                <Spinner />
              ) : popularModules && popularModules.length > 0 ? (
                <TopModulesChart modules={popularModules} />
              ) : (
                <Text>No data available</Text>
              )}
            </Box>
          </TabPanel>

          {/* Trending Tab */}
          <TabPanel>
            <Box p={4} bg={bgColor} borderWidth={1} borderRadius="md" minH="400px">
              <Heading size="md" mb={4}>Trending Modules (7d Growth)</Heading>
              {trendingLoading ? (
                <Spinner />
              ) : trendingModules && trendingModules.length > 0 ? (
                <VStack align="stretch" spacing={3}>
                  {trendingModules.map((module, idx) => (
                    <HStack
                      key={module.module_name}
                      p={3}
                      bg={useColorModeValue('gray.50', 'gray.700')}
                      borderRadius="md"
                      justify="space-between"
                    >
                      <HStack>
                        <Text fontWeight="bold" color="gray.500">#{idx + 1}</Text>
                        <Text fontWeight="medium">{module.module_name}</Text>
                      </HStack>
                      <HStack>
                        <Text fontSize="sm" color="gray.600">
                          {module.current_downloads.toLocaleString()} downloads
                        </Text>
                        <Text
                          fontSize="sm"
                          fontWeight="bold"
                          color={module.growth_rate > 0 ? 'green.500' : 'red.500'}
                        >
                          {module.growth_rate > 0 ? '+' : ''}
                          {(module.growth_rate * 100).toFixed(1)}%
                        </Text>
                      </HStack>
                    </HStack>
                  ))}
                </VStack>
              ) : (
                <Text>No trending data available</Text>
              )}
            </Box>
          </TabPanel>

          {/* Languages Tab */}
          <TabPanel>
            <Box p={4} bg={bgColor} borderWidth={1} borderRadius="md" minH="400px">
              <Heading size="md" mb={4}>Language Distribution</Heading>
              <LanguageChart period={period} />
            </Box>
          </TabPanel>
        </TabPanels>
      </Tabs>
    </VStack>
  );
};
