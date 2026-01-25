import { ChakraProvider, Container, Box, Heading, HStack } from '@chakra-ui/react';
import { BrowserRouter as Router, Routes, Route, useParams, Link as RouterLink } from 'react-router-dom';
import { ModuleList } from './components/ModuleList';
import { ModuleDetail } from './components/ModuleDetail';
import { SearchBar } from './components/SearchBar';
import { useModules, useModule } from './hooks/useModules';

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
              <Box width="400px">
                <SearchBar />
              </Box>
            </HStack>
          </Box>

          {/* Routes */}
          <Routes>
            <Route path="/" element={<ModuleListPage />} />
            <Route path="/modules/:moduleName" element={<ModuleDetailPage />} />
            <Route path="/modules/:moduleName/versions/:version" element={<ModuleDetailPage />} />
          </Routes>
        </Container>
      </Router>
    </ChakraProvider>
  );
}

export default App; 