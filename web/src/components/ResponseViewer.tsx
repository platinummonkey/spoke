import React from 'react';
import {
  Box,
  VStack,
  Heading,
  Text,
  Code,
  Alert,
  AlertIcon,
  Badge,
  useClipboard,
  IconButton,
  Tooltip,
  HStack,
} from '@chakra-ui/react';
import { CopyIcon, CheckIcon } from '@chakra-ui/icons';

interface ResponseViewerProps {
  response: any;
  error?: string | null;
  loading?: boolean;
}

export const ResponseViewer: React.FC<ResponseViewerProps> = ({
  response,
  error,
  loading = false,
}) => {
  const responseJson = response ? JSON.stringify(response, null, 2) : '';
  const { hasCopied, onCopy } = useClipboard(responseJson);

  if (loading) {
    return (
      <Box p={4} bg="gray.50" borderRadius="md" borderWidth={1}>
        <Text color="gray.500">Loading...</Text>
      </Box>
    );
  }

  if (error) {
    return (
      <Alert status="error" borderRadius="md">
        <AlertIcon />
        <VStack align="stretch" spacing={1} flex={1}>
          <Text fontWeight="medium">Error</Text>
          <Text fontSize="sm">{error}</Text>
        </VStack>
      </Alert>
    );
  }

  if (!response) {
    return (
      <Box p={4} bg="gray.50" borderRadius="md" borderWidth={1}>
        <Text color="gray.500" fontSize="sm">
          No response yet. Send a request to see the response here.
        </Text>
      </Box>
    );
  }

  return (
    <VStack align="stretch" spacing={3}>
      <HStack justify="space-between">
        <HStack spacing={2}>
          <Heading size="sm">Response</Heading>
          <Badge colorScheme="green">Success</Badge>
        </HStack>
        <Tooltip label={hasCopied ? 'Copied!' : 'Copy response'}>
          <IconButton
            aria-label="Copy response"
            icon={hasCopied ? <CheckIcon /> : <CopyIcon />}
            size="sm"
            onClick={onCopy}
            colorScheme={hasCopied ? 'green' : 'gray'}
          />
        </Tooltip>
      </HStack>

      <Box
        as="pre"
        p={4}
        bg="gray.50"
        borderRadius="md"
        borderWidth={1}
        overflowX="auto"
        fontSize="sm"
        fontFamily="monospace"
        maxH="400px"
        overflowY="auto"
      >
        <Code display="block" whiteSpace="pre">
          {responseJson}
        </Code>
      </Box>
    </VStack>
  );
};
