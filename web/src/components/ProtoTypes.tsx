import React, { useState, useEffect } from 'react';
import {
  Box,
  VStack,
  Heading,
  Text,
  Accordion,
  AccordionItem,
  AccordionButton,
  AccordionPanel,
  AccordionIcon,
  Badge,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Code,
  Button,
  useToast,
  Spinner,
  IconButton,
  Tooltip,
  HStack,
  Flex,
} from '@chakra-ui/react';
import { CopyIcon, DownloadIcon } from '@chakra-ui/icons';
import { ProtoFile, Message, Enum, Service } from '../types';
import { getFile } from '../api/client';
import Prism from 'prismjs';
import 'prismjs/components/prism-protobuf';
import 'prismjs/themes/prism.css';

interface ProtoTypesProps {
  files: ProtoFile[];
  moduleName: string;
  version: string;
  versions?: string[];
  onVersionChange?: (version: string) => void;
}

const MessageFields: React.FC<{ message: Message }> = ({ message }) => (
  <VStack align="stretch" spacing={2}>
    {message.fields && message.fields.length > 0 && (
      <Table size="sm">
        <Thead>
          <Tr>
            <Th>Field</Th>
            <Th>Type</Th>
            <Th>Number</Th>
            <Th>Label</Th>
          </Tr>
        </Thead>
        <Tbody>
          {message.fields.map((field) => (
            <Tr key={field.name}>
              <Td>{field.name}</Td>
              <Td>
                <Code>{field.type}</Code>
              </Td>
              <Td>{field.number}</Td>
              <Td>
                <Badge colorScheme={field.label === 'repeated' ? 'blue' : 'gray'}>
                  {field.label}
                </Badge>
              </Td>
            </Tr>
          ))}
        </Tbody>
      </Table>
    )}
    {message.nestedMessages && message.nestedMessages.length > 0 && (
      <Box mt={4}>
        <Heading size="sm" mb={2}>Nested Messages</Heading>
        <Accordion allowMultiple>
          {message.nestedMessages.map((nested) => (
            <AccordionItem key={nested.name}>
              <AccordionButton>
                <Box flex="1" textAlign="left">
                  <Text fontWeight="bold">{nested.name}</Text>
                </Box>
                <AccordionIcon />
              </AccordionButton>
              <AccordionPanel pb={4}>
                <MessageFields message={nested} />
              </AccordionPanel>
            </AccordionItem>
          ))}
        </Accordion>
      </Box>
    )}
    {message.nestedEnums && message.nestedEnums.length > 0 && (
      <Box mt={4}>
        <Heading size="sm" mb={2}>Nested Enums</Heading>
        <Accordion allowMultiple>
          {message.nestedEnums.map((nested) => (
            <AccordionItem key={nested.name}>
              <AccordionButton>
                <Box flex="1" textAlign="left">
                  <Text fontWeight="bold">{nested.name}</Text>
                </Box>
                <AccordionIcon />
              </AccordionButton>
              <AccordionPanel pb={4}>
                <EnumValues enum={nested} />
              </AccordionPanel>
            </AccordionItem>
          ))}
        </Accordion>
      </Box>
    )}
  </VStack>
);

const EnumValues: React.FC<{ enum: Enum }> = ({ enum: enumType }) => (
  <Table size="sm">
    <Thead>
      <Tr>
        <Th>Name</Th>
        <Th>Value</Th>
      </Tr>
    </Thead>
    <Tbody>
      {enumType.values.map((value) => (
        <Tr key={value.name}>
          <Td>{value.name}</Td>
          <Td>{value.number}</Td>
        </Tr>
      ))}
    </Tbody>
  </Table>
);

const RpcMethods: React.FC<{ service: Service }> = ({ service }) => (
  <Table size="sm">
    <Thead>
      <Tr>
        <Th>Method</Th>
        <Th>Input Type</Th>
        <Th>Output Type</Th>
        <Th>Streaming</Th>
      </Tr>
    </Thead>
    <Tbody>
      {service.methods.map((method) => (
        <Tr key={method.name}>
          <Td>{method.name}</Td>
          <Td>
            <Code>{method.inputType}</Code>
          </Td>
          <Td>
            <Code>{method.outputType}</Code>
          </Td>
          <Td>
            {method.clientStreaming && method.serverStreaming ? (
              <Badge colorScheme="purple">Bidirectional</Badge>
            ) : method.clientStreaming ? (
              <Badge colorScheme="blue">Client</Badge>
            ) : method.serverStreaming ? (
              <Badge colorScheme="green">Server</Badge>
            ) : (
              <Badge colorScheme="gray">Unary</Badge>
            )}
          </Td>
        </Tr>
      ))}
    </Tbody>
  </Table>
);

const FileContent: React.FC<{
  moduleName: string;
  version: string;
  path: string;
}> = ({ moduleName, version, path }) => {
  const [content, setContent] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const toast = useToast();

  const fetchContent = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await getFile(moduleName, version, path);
      console.log('Fetched file content:', { path, response });
      setContent(response.content);
    } catch (err) {
      console.error('Error fetching file content:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch file content');
      toast({
        title: 'Error',
        description: 'Failed to fetch file content',
        status: 'error',
        duration: 5000,
        isClosable: true,
      });
    } finally {
      setLoading(false);
    }
  };

  const copyToClipboard = () => {
    if (content) {
      navigator.clipboard.writeText(content);
      toast({
        title: 'Copied',
        description: 'File content copied to clipboard',
        status: 'success',
        duration: 2000,
        isClosable: true,
      });
    }
  };

  const downloadFile = () => {
    if (content) {
      const blob = new Blob([content], { type: 'text/plain' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = path.split('/').pop() || 'proto';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }
  };

  // Fetch content when component mounts
  useEffect(() => {
    fetchContent();
  }, [moduleName, version, path]);

  const renderContent = () => {
    if (loading) {
      return (
        <Box textAlign="center" py={4}>
          <Spinner size="sm" />
        </Box>
      );
    }

    if (error) {
      return (
        <Box>
          <Text color="red.500">{error}</Text>
          <Button size="sm" mt={2} onClick={fetchContent}>
            Retry
          </Button>
        </Box>
      );
    }

    if (!content) {
      return (
        <Box textAlign="center" py={4}>
          <Button size="sm" onClick={fetchContent}>
            Load File Content
          </Button>
        </Box>
      );
    }

    const highlightedContent = Prism.highlight(
      String(content),
      Prism.languages.protobuf,
      'protobuf'
    );

    return (
      <Box
        as="pre"
        p={4}
        bg="gray.50"
        borderRadius="md"
        overflowX="auto"
        fontSize="sm"
        dangerouslySetInnerHTML={{ __html: highlightedContent }}
      />
    );
  };

  return (
    <Accordion allowToggle>
      <AccordionItem>
        <Flex>
          <AccordionButton flex="1">
            <Box flex="1" textAlign="left">
              <Text fontWeight="bold">File Content</Text>
            </Box>
            <AccordionIcon />
          </AccordionButton>
          <HStack spacing={2} px={2} align="center">
            <Tooltip label="Copy to clipboard">
              <IconButton
                aria-label="Copy to clipboard"
                icon={<CopyIcon />}
                size="sm"
                onClick={copyToClipboard}
              />
            </Tooltip>
            <Tooltip label="Download proto file">
              <IconButton
                aria-label="Download proto file"
                icon={<DownloadIcon />}
                size="sm"
                onClick={downloadFile}
              />
            </Tooltip>
          </HStack>
        </Flex>
        <AccordionPanel pb={4}>
          {renderContent()}
        </AccordionPanel>
      </AccordionItem>
    </Accordion>
  );
};

export const ProtoTypes: React.FC<ProtoTypesProps> = ({
  files,
  moduleName,
  version,
}) => {
  console.log('ProtoTypes props:', { files, moduleName, version });
  
  return (
    <VStack align="stretch" spacing={6}>
      {files.map((file) => {
        console.log('Processing file:', file);
        return (
          <Box key={file.path} p={4} borderWidth={1} borderRadius="md">
            <Heading size="md" mb={4}>
              {file.path}
            </Heading>

            <FileContent moduleName={moduleName} version={version} path={file.path} />

            {file.messages && file.messages.length > 0 && (
              <Box mb={6}>
                <Heading size="sm" mb={2}>Messages</Heading>
                <Accordion allowMultiple>
                  {file.messages.map((message) => {
                    console.log('Processing message:', message);
                    return (
                      <AccordionItem key={message.name}>
                        <AccordionButton>
                          <Box flex="1" textAlign="left">
                            <Text fontWeight="bold">{message.name}</Text>
                          </Box>
                          <AccordionIcon />
                        </AccordionButton>
                        <AccordionPanel pb={4}>
                          <MessageFields message={message} />
                        </AccordionPanel>
                      </AccordionItem>
                    );
                  })}
                </Accordion>
              </Box>
            )}

            {file.enums && file.enums.length > 0 && (
              <Box mb={6}>
                <Heading size="sm" mb={2}>Enums</Heading>
                <Accordion allowMultiple>
                  {file.enums.map((enumType) => {
                    console.log('Processing enum:', enumType);
                    return (
                      <AccordionItem key={enumType.name}>
                        <AccordionButton>
                          <Box flex="1" textAlign="left">
                            <Text fontWeight="bold">{enumType.name}</Text>
                          </Box>
                          <AccordionIcon />
                        </AccordionButton>
                        <AccordionPanel pb={4}>
                          <EnumValues enum={enumType} />
                        </AccordionPanel>
                      </AccordionItem>
                    );
                  })}
                </Accordion>
              </Box>
            )}

            {file.services && file.services.length > 0 && (
              <Box>
                <Heading size="sm" mb={2}>Services</Heading>
                <Accordion allowMultiple>
                  {file.services.map((service) => {
                    console.log('Processing service:', service);
                    return (
                      <AccordionItem key={service.name}>
                        <AccordionButton>
                          <Box flex="1" textAlign="left">
                            <Text fontWeight="bold">{service.name}</Text>
                          </Box>
                          <AccordionIcon />
                        </AccordionButton>
                        <AccordionPanel pb={4}>
                          <RpcMethods service={service} />
                        </AccordionPanel>
                      </AccordionItem>
                    );
                  })}
                </Accordion>
              </Box>
            )}
          </Box>
        );
      })}
    </VStack>
  );
}; 