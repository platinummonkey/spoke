import React, { useState, useEffect } from 'react';
import {
  Box,
  VStack,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  Spinner,
  Text,
  Divider,
  Code,
} from '@chakra-ui/react';

interface MigrationGuideProps {
  moduleName: string;
  fromVersion: string;
  toVersion: string;
}

export const MigrationGuide: React.FC<MigrationGuideProps> = ({
  moduleName,
  fromVersion,
  toVersion,
}) => {
  const [guide, setGuide] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [notFound, setNotFound] = useState<boolean>(false);

  useEffect(() => {
    const fetchGuide = async () => {
      setLoading(true);
      setNotFound(false);
      setGuide(null);

      try {
        // Try to fetch manual migration guide
        // Format: /migrations/{module}/v{from}-to-v{to}.md
        const guidePath = `/migrations/${moduleName}/v${fromVersion}-to-v${toVersion}.md`;

        const response = await fetch(guidePath);

        if (response.ok) {
          const markdown = await response.text();
          setGuide(markdown);
        } else {
          setNotFound(true);
        }
      } catch (err) {
        console.error('Error fetching migration guide:', err);
        setNotFound(true);
      } finally {
        setLoading(false);
      }
    };

    if (fromVersion && toVersion && fromVersion !== toVersion) {
      fetchGuide();
    }
  }, [moduleName, fromVersion, toVersion]);

  if (loading) {
    return (
      <Box textAlign="center" py={10}>
        <Spinner size="lg" />
        <Text mt={4} color="gray.600">
          Loading migration guide...
        </Text>
      </Box>
    );
  }

  if (notFound || !guide) {
    return (
      <VStack align="stretch" spacing={4}>
        <Alert status="info" variant="left-accent">
          <AlertIcon />
          <Box flex="1">
            <AlertTitle fontSize="md">No Manual Migration Guide Available</AlertTitle>
            <AlertDescription fontSize="sm">
              A manual migration guide has not been created for upgrading from v{fromVersion} to v{toVersion}.
            </AlertDescription>
          </Box>
        </Alert>

        <Box p={4} bg="gray.50" borderRadius="md">
          <Text fontWeight="medium" mb={3}>
            Alternative Resources:
          </Text>
          <VStack align="stretch" spacing={2} pl={4}>
            <Text fontSize="sm">
              • Use the <strong>Schema Diff</strong> tab to see automated breaking change detection
            </Text>
            <Text fontSize="sm">
              • Check the <strong>API Explorer</strong> tab to understand new service methods
            </Text>
            <Text fontSize="sm">
              • Review the <strong>Types</strong> tab to see all available messages and enums
            </Text>
            <Text fontSize="sm">
              • Generate <strong>Code Examples</strong> to see how to use the new version
            </Text>
          </VStack>
        </Box>

        <Divider />

        <Box p={4} bg="blue.50" borderRadius="md" borderWidth={1} borderColor="blue.200">
          <Text fontSize="sm" fontWeight="medium" mb={2} color="blue.700">
            📝 Contribute a Migration Guide
          </Text>
          <Text fontSize="sm" color="blue.900" mb={2}>
            If you've successfully migrated from v{fromVersion} to v{toVersion}, consider contributing a migration guide!
          </Text>
          <Text fontSize="sm" color="blue.900">
            Create a markdown file at:{' '}
            <Code fontSize="xs" bg="white" p={1} borderRadius="sm">
              docs/content/migrations/{moduleName}/v{fromVersion}-to-v{toVersion}.md
            </Code>
          </Text>
        </Box>
      </VStack>
    );
  }

  // Render markdown content
  // Note: In a production app, you'd use a proper markdown renderer like react-markdown
  // For now, we'll display it as preformatted text
  return (
    <VStack align="stretch" spacing={4}>
      <Alert status="success" variant="left-accent">
        <AlertIcon />
        <AlertTitle fontSize="md">Manual Migration Guide Available</AlertTitle>
      </Alert>

      <Box
        p={4}
        bg="white"
        borderRadius="md"
        borderWidth={1}
        maxH="600px"
        overflowY="auto"
      >
        <Box
          as="pre"
          whiteSpace="pre-wrap"
          fontFamily="monospace"
          fontSize="sm"
          lineHeight="1.6"
        >
          {guide}
        </Box>
      </Box>

      <Box p={3} bg="gray.50" borderRadius="md" fontSize="sm">
        <Text color="gray.600">
          💡 <strong>Tip:</strong> This guide was manually created by maintainers.
          For automated change detection, check the <strong>Schema Diff</strong> tab.
        </Text>
      </Box>
    </VStack>
  );
};
