import React from 'react';
import { ChakraProvider, Container } from '@chakra-ui/react';
import { BrowserRouter as Router, Routes, Route, useParams } from 'react-router-dom';
import { ModuleList } from './components/ModuleList';
import { ModuleDetail } from './components/ModuleDetail';
import { useModules, useModule } from './hooks/useModules';

const ModuleListPage = () => {
  const { modules, loading, error, retry } = useModules();
  return <ModuleList modules={modules} loading={loading} error={error} retry={retry} />;
};

const ModuleDetailPage = () => {
  const { moduleName } = useParams<{ moduleName: string }>();
  const { module, loading, error, retry } = useModule(moduleName || '');
  return <ModuleDetail module={module} loading={loading} error={error} retry={retry} />;
};

function App() {
  return (
    <ChakraProvider>
      <Router>
        <Container maxW="container.xl" py={8}>
          <Routes>
            <Route path="/" element={<ModuleListPage />} />
            <Route path="/modules/:moduleName" element={<ModuleDetailPage />} />
          </Routes>
        </Container>
      </Router>
    </ChakraProvider>
  );
}

export default App; 