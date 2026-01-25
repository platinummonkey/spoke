import React, { useState, useEffect } from 'react';
import {
  Box,
  VStack,
  HStack,
  Select,
  Spinner,
  Alert,
  AlertIcon,
  Code,
  useClipboard,
  IconButton,
  Tooltip,
  Text,
  Badge,
} from '@chakra-ui/react';
import { CopyIcon, CheckIcon } from '@chakra-ui/icons';

interface CodeExamplesProps {
  moduleName: string;
  version: string;
}

interface Language {
  id: string;
  name: string;
  displayName: string;
}

// List of supported languages (matches backend)
const LANGUAGES: Language[] = [
  { id: 'go', name: 'Go', displayName: 'Go (Protocol Buffers)' },
  { id: 'python', name: 'Python', displayName: 'Python (Protocol Buffers)' },
  { id: 'java', name: 'Java', displayName: 'Java (Protocol Buffers)' },
  { id: 'cpp', name: 'C++', displayName: 'C++ (Protocol Buffers)' },
  { id: 'csharp', name: 'C#', displayName: 'C# (Protocol Buffers)' },
  { id: 'rust', name: 'Rust', displayName: 'Rust (prost)' },
  { id: 'typescript', name: 'TypeScript', displayName: 'TypeScript (ts-proto)' },
  { id: 'javascript', name: 'JavaScript', displayName: 'JavaScript (google-protobuf)' },
  { id: 'dart', name: 'Dart', displayName: 'Dart (protobuf)' },
  { id: 'swift', name: 'Swift', displayName: 'Swift (SwiftProtobuf)' },
  { id: 'kotlin', name: 'Kotlin', displayName: 'Kotlin (protobuf-kotlin)' },
  { id: 'objectivec', name: 'Objective-C', displayName: 'Objective-C (Protobuf)' },
  { id: 'ruby', name: 'Ruby', displayName: 'Ruby (google-protobuf)' },
  { id: 'php', name: 'PHP', displayName: 'PHP (Protobuf)' },
  { id: 'scala', name: 'Scala', displayName: 'Scala (ScalaPB)' },
];

export const CodeExamples: React.FC<CodeExamplesProps> = ({ moduleName, version }) => {
  const [language, setLanguage] = useState<string>('go');
  const [code, setCode] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const { hasCopied, onCopy } = useClipboard(code);

  useEffect(() => {
    const fetchExample = async () => {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch(
          `/api/v1/modules/${moduleName}/versions/${version}/examples/${language}`
        );

        if (!response.ok) {
          throw new Error(`Failed to fetch example: ${response.statusText}`);
        }

        const exampleCode = await response.text();
        setCode(exampleCode);
      } catch (err) {
        console.error('Error fetching example:', err);
        setError(err instanceof Error ? err.message : 'Failed to fetch example');
        setCode('');
      } finally {
        setLoading(false);
      }
    };

    fetchExample();
  }, [moduleName, version, language]);

  const selectedLanguage = LANGUAGES.find(lang => lang.id === language);

  return (
    <VStack align="stretch" spacing={4}>
      {/* Language Selector */}
      <HStack justify="space-between" align="center">
        <HStack spacing={3}>
          <Text fontWeight="medium">Language:</Text>
          <Select
            value={language}
            onChange={(e) => setLanguage(e.target.value)}
            width="300px"
          >
            {LANGUAGES.map(lang => (
              <option key={lang.id} value={lang.id}>
                {lang.displayName}
              </option>
            ))}
          </Select>
          {selectedLanguage && (
            <Badge colorScheme="blue" fontSize="sm">
              {selectedLanguage.name}
            </Badge>
          )}
        </HStack>

        <Tooltip label={hasCopied ? 'Copied!' : 'Copy code'}>
          <IconButton
            aria-label="Copy code"
            icon={hasCopied ? <CheckIcon /> : <CopyIcon />}
            size="sm"
            onClick={onCopy}
            colorScheme={hasCopied ? 'green' : 'gray'}
            isDisabled={!code || loading}
          />
        </Tooltip>
      </HStack>

      {/* Code Display */}
      {loading && (
        <Box textAlign="center" py={10}>
          <Spinner size="lg" />
          <Text mt={4} color="gray.600">
            Generating {selectedLanguage?.name} example...
          </Text>
        </Box>
      )}

      {error && (
        <Alert status="error">
          <AlertIcon />
          {error}
        </Alert>
      )}

      {!loading && !error && code && (
        <Box
          p={4}
          bg="gray.50"
          borderRadius="md"
          borderWidth={1}
          overflowX="auto"
          maxH="600px"
          overflowY="auto"
        >
          <Code
            display="block"
            whiteSpace="pre"
            fontFamily="monospace"
            fontSize="sm"
            bg="transparent"
          >
            {code}
          </Code>
        </Box>
      )}

      {!loading && !error && !code && (
        <Alert status="info">
          <AlertIcon />
          No example available for {selectedLanguage?.name}
        </Alert>
      )}

      {/* Usage Instructions */}
      <Box p={3} bg="blue.50" borderRadius="md" fontSize="sm">
        <Text fontWeight="medium" mb={2}>
          Usage Instructions:
        </Text>
        <VStack align="stretch" spacing={1} pl={4}>
          <Text>• Select a language from the dropdown to see a complete working example</Text>
          <Text>• Examples show how to connect to a gRPC server and call service methods</Text>
          <Text>• Copy the code and adapt it to your specific use case</Text>
          <Text>• Make sure to install the required dependencies for your language</Text>
        </VStack>
      </Box>
    </VStack>
  );
};
