import React, { Suspense } from 'react';
import { ChakraProvider, Container, Box, Heading, HStack, Spinner, Center, Button } from '@chakra-ui/react';
import { BrowserRouter as Router, Routes, Route, useParams, Link as RouterLink } from 'react-router-dom';
import { StarIcon, InfoIcon } from '@chakra-ui/icons';
import { EnhancedSearchBar } from './components/EnhancedSearchBar';
import { ErrorBoundary } from './components/ErrorBoundary';
import { useModules, useModule } from './hooks/useModules';

// Lazy load components for code splitting
const ModuleList = React.lazy(() => import('./components/ModuleList').then(m => ({ default: m.ModuleList })));
const ModuleDetail = React.lazy(() => import('./components/ModuleDetail').then(m => ({ default: m.ModuleDetail })));
const UserFeatures = React.lazy(() => import('./components/UserFeatures').then(m => ({ default: m.UserFeatures })));
const AnalyticsDashboard = React.lazy(() => import('./components/analytics/AnalyticsDashboard').then(m => ({ default: m.AnalyticsDashboard })));

const ModuleListPage = () => {
  const { modules, loading, error, retry } = useModules();
  return <ModuleList modules={modules} loading={loading} error={error} retry={retry} />;
};

const ModuleDetailPage = () => {
  const { moduleName, version } = useParams<{ moduleName: string; version?: string }>();
  const { module, loading, error, retry } = useModule(moduleName || '');
  return <ModuleDetail module={module} loading={loading} error={error} retry={retry} initialVersion={version} />;
};

function App() {
  return (
    <ChakraProvider>
      <Router>
        <Container maxW="container.xl" py={8}>
          {/* Header with Search */}
          <Box mb={8}>
            <HStack justify="space-between" align="center" mb={6}>
              <Heading
                as={RouterLink}
                to="/"
                size="lg"
                cursor="pointer"
                _hover={{ color: 'blue.600' }}
                transition="color 0.2s"
              >
                Spoke Registry
              </Heading>
              <HStack spacing={4}>
                <Button
                  as={RouterLink}
                  to="/library"
                  leftIcon={<StarIcon />}
                  size="sm"
                  variant="ghost"
                  colorScheme="orange"
                >
                  My Library
                </Button>
                <Button
                  as={RouterLink}
                  to="/analytics"
                  leftIcon={<InfoIcon />}
                  size="sm"
                  variant="ghost"
                  colorScheme="blue"
                >
                  Analytics
                </Button>
                <Box width="400px">
                  <EnhancedSearchBar />
                </Box>
              </HStack>
            </HStack>
          </Box>

          {/* Routes */}
          <ErrorBoundary>
            <Suspense
              fallback={
                <Center minH="400px">
                  <Spinner size="xl" color="blue.500" />
                </Center>
              }
            >
              <Routes>
                <Route path="/" element={<ModuleListPage />} />
                <Route path="/library" element={<UserFeatures />} />
                <Route path="/analytics" element={<AnalyticsDashboard />} />
                <Route path="/modules/:moduleName" element={<ModuleDetailPage />} />
                <Route path="/modules/:moduleName/versions/:version" element={<ModuleDetailPage />} />
              </Routes>
            </Suspense>
          </ErrorBoundary>
        </Container>
      </Router>
    </ChakraProvider>
  );
}

export default App; 