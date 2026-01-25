import React from 'react';
import {
  Box,
  VStack,
  HStack,
  Heading,
  Text,
  Badge,
  Divider,
  SimpleGrid,
  Alert,
  AlertIcon,
  Code,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
} from '@chakra-ui/react';
import { RpcMethod, Message, ProtoFile } from '../types';
import { MessageViewer } from './MessageViewer';
import { RequestBuilder } from './RequestBuilder';

interface MethodDetailProps {
  method: RpcMethod;
  files: ProtoFile[];
}

// Helper to find a message by type name across all files
const findMessage = (files: ProtoFile[], typeName: string): Message | null => {
  // Remove leading dot if present (e.g., ".package.MessageName" -> "package.MessageName")
  const cleanTypeName = typeName.startsWith('.') ? typeName.slice(1) : typeName;

  for (const file of files) {
    // Search in top-level messages
    const message = file.messages.find(m => m.name === cleanTypeName || cleanTypeName.endsWith(`.${m.name}`));
    if (message) return message;

    // Search in nested messages (simple implementation - could be enhanced)
    for (const msg of file.messages) {
      const nested = findNestedMessage(msg, cleanTypeName);
      if (nested) return nested;
    }
  }

  return null;
};

const findNestedMessage = (message: Message, typeName: string): Message | null => {
  if (message.name === typeName || typeName.endsWith(`.${message.name}`)) {
    return message;
  }

  for (const nested of message.nestedMessages) {
    const found = findNestedMessage(nested, typeName);
    if (found) return found;
  }

  return null;
};

const getStreamingBadge = (method: RpcMethod) => {
  if (method.clientStreaming && method.serverStreaming) {
    return (
      <Badge colorScheme="purple" fontSize="sm">
        Bidirectional Streaming
      </Badge>
    );
  }
  if (method.clientStreaming) {
    return (
      <Badge colorScheme="blue" fontSize="sm">
        Client Streaming
      </Badge>
    );
  }
  if (method.serverStreaming) {
    return (
      <Badge colorScheme="green" fontSize="sm">
        Server Streaming
      </Badge>
    );
  }
  return (
    <Badge colorScheme="gray" fontSize="sm">
      Unary
    </Badge>
  );
};

export const MethodDetail: React.FC<MethodDetailProps> = ({ method, files }) => {
  const requestMessage = findMessage(files, method.inputType);
  const responseMessage = findMessage(files, method.outputType);

  return (
    <Box>
      <VStack align="stretch" spacing={4}>
        {/* Method Header */}
        <Box>
          <HStack spacing={3} mb={2}>
            <Heading size="md">{method.name}</Heading>
            {getStreamingBadge(method)}
          </HStack>
          <HStack spacing={2} fontSize="sm" color="gray.600">
            <Text>
              <Code fontSize="sm">{method.inputType}</Code> → <Code fontSize="sm">{method.outputType}</Code>
            </Text>
          </HStack>
        </Box>

        <Divider />

        {/* Tabs for Schema and Playground */}
        <Tabs>
          <TabList>
            <Tab>Message Schema</Tab>
            <Tab>Try It Out</Tab>
          </TabList>

          <TabPanels>
            {/* Schema Tab */}
            <TabPanel>
              <VStack align="stretch" spacing={4}>
                {/* Request and Response Side by Side */}
                <SimpleGrid columns={{ base: 1, md: 2 }} spacing={6}>
                  {/* Request Type */}
                  <Box>
                    <Heading size="sm" mb={3} color="blue.600">
                      Request
                    </Heading>
                    {requestMessage ? (
                      <MessageViewer message={requestMessage} showTitle={false} />
                    ) : (
                      <Alert status="warning" size="sm">
                        <AlertIcon />
                        <Text fontSize="sm">
                          Message type <Code>{method.inputType}</Code> not found in loaded files
                        </Text>
                      </Alert>
                    )}
                  </Box>

                  {/* Response Type */}
                  <Box>
                    <Heading size="sm" mb={3} color="green.600">
                      Response
                    </Heading>
                    {responseMessage ? (
                      <MessageViewer message={responseMessage} showTitle={false} />
                    ) : (
                      <Alert status="warning" size="sm">
                        <AlertIcon />
                        <Text fontSize="sm">
                          Message type <Code>{method.outputType}</Code> not found in loaded files
                        </Text>
                      </Alert>
                    )}
                  </Box>
                </SimpleGrid>

                {/* Streaming Information */}
                {(method.clientStreaming || method.serverStreaming) && (
                  <Box p={3} bg="blue.50" borderRadius="md" borderWidth={1} borderColor="blue.200">
                    <HStack spacing={2} fontSize="sm">
                      <Text fontWeight="medium">Streaming:</Text>
                      {method.clientStreaming && (
                        <Text>
                          Client sends multiple <Code fontSize="xs">{method.inputType}</Code> messages
                        </Text>
                      )}
                      {method.clientStreaming && method.serverStreaming && <Text>•</Text>}
                      {method.serverStreaming && (
                        <Text>
                          Server returns multiple <Code fontSize="xs">{method.outputType}</Code> messages
                        </Text>
                      )}
                    </HStack>
                  </Box>
                )}
              </VStack>
            </TabPanel>

            {/* Playground Tab */}
            <TabPanel>
              {requestMessage && responseMessage ? (
                <RequestBuilder
                  requestMessage={requestMessage}
                  responseMessage={responseMessage}
                />
              ) : (
                <Alert status="warning">
                  <AlertIcon />
                  <Text>
                    Cannot load playground: request or response message definition not found.
                  </Text>
                </Alert>
              )}
            </TabPanel>
          </TabPanels>
        </Tabs>
      </VStack>
    </Box>
  );
};
