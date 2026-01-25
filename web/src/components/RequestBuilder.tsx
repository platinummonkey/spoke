import React, { useState, useEffect } from 'react';
import {
  Box,
  VStack,
  HStack,
  Button,
  Alert,
  AlertIcon,
  Text,
  Divider,
  SimpleGrid,
} from '@chakra-ui/react';
import { RepeatIcon } from '@chakra-ui/icons';
import { Message, Field } from '../types';
import { JsonEditor } from './JsonEditor';
import { ResponseViewer } from './ResponseViewer';
import { useProtoValidation, ValidationError } from '../hooks/useProtoValidation';

interface RequestBuilderProps {
  requestMessage: Message;
  responseMessage: Message;
}

// Generate sample value based on field type
const generateSampleValue = (field: Field): any => {
  if (field.label === 'repeated') {
    return [];
  }

  switch (field.type) {
    case 'string':
      return field.name.includes('email') ? 'user@example.com' :
             field.name.includes('name') ? 'Example Name' :
             field.name.includes('id') ? 'example-id-123' :
             'example value';

    case 'bytes':
      return 'base64-encoded-data';

    case 'int32':
    case 'int64':
    case 'uint32':
    case 'uint64':
    case 'sint32':
    case 'sint64':
    case 'fixed32':
    case 'fixed64':
    case 'sfixed32':
    case 'sfixed64':
      return field.name.includes('count') ? 10 :
             field.name.includes('age') ? 25 :
             field.name.includes('id') ? 1 :
             42;

    case 'float':
    case 'double':
      return 3.14;

    case 'bool':
      return true;

    default:
      // Message type - return empty object
      return {};
  }
};

// Generate sample request JSON from message definition
const generateSampleRequest = (message: Message): string => {
  const sampleData: Record<string, any> = {};

  for (const field of message.fields) {
    sampleData[field.name] = generateSampleValue(field);
  }

  return JSON.stringify(sampleData, null, 2);
};

// Generate mock response (for demo purposes)
const generateMockResponse = (message: Message): any => {
  const mockData: Record<string, any> = {};

  for (const field of message.fields) {
    mockData[field.name] = generateSampleValue(field);
  }

  return mockData;
};

export const RequestBuilder: React.FC<RequestBuilderProps> = ({
  requestMessage,
  responseMessage,
}) => {
  const [requestJson, setRequestJson] = useState('');
  const [response, setResponse] = useState<any>(null);
  const [responseError, setResponseError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const { validate, errors, clearErrors } = useProtoValidation();

  // Initialize with sample request
  useEffect(() => {
    setRequestJson(generateSampleRequest(requestMessage));
    clearErrors();
    setResponse(null);
    setResponseError(null);
  }, [requestMessage, clearErrors]);

  const handleGenerateSample = () => {
    setRequestJson(generateSampleRequest(requestMessage));
    clearErrors();
  };

  const handleSimulate = () => {
    // Parse JSON
    let parsedRequest;
    try {
      parsedRequest = JSON.parse(requestJson);
    } catch (error) {
      clearErrors();
      setResponseError('Invalid JSON: ' + (error as Error).message);
      return;
    }

    // Validate against message schema
    const result = validate(requestMessage, parsedRequest);

    if (!result.valid) {
      setResponseError(`Validation failed: ${result.errors.length} error(s)`);
      return;
    }

    // Clear errors and simulate request
    clearErrors();
    setResponseError(null);
    setLoading(true);

    // Simulate network delay
    setTimeout(() => {
      const mockResponse = generateMockResponse(responseMessage);
      setResponse(mockResponse);
      setLoading(false);
    }, 500);
  };

  const hasErrors = errors.length > 0;

  return (
    <VStack align="stretch" spacing={6}>
      {/* Info banner */}
      <Alert status="info" variant="left-accent" fontSize="sm">
        <AlertIcon />
        <Text>
          This is a client-side playground for testing message structures.
          Edit the request JSON below and click "Simulate" to see a mock response.
        </Text>
      </Alert>

      <SimpleGrid columns={{ base: 1, lg: 2 }} spacing={6}>
        {/* Request Section */}
        <VStack align="stretch" spacing={3}>
          <HStack justify="space-between">
            <Text fontWeight="bold" fontSize="md">
              Request ({requestMessage.name})
            </Text>
            <Button
              size="sm"
              leftIcon={<RepeatIcon />}
              onClick={handleGenerateSample}
              variant="outline"
            >
              Generate Sample
            </Button>
          </HStack>

          <JsonEditor
            value={requestJson}
            onChange={setRequestJson}
            height="400px"
            placeholder={generateSampleRequest(requestMessage)}
          />

          {/* Validation Errors */}
          {hasErrors && (
            <Alert status="error" fontSize="sm">
              <AlertIcon />
              <VStack align="stretch" spacing={1} flex={1}>
                <Text fontWeight="medium">Validation Errors:</Text>
                {errors.slice(0, 5).map((error: ValidationError, index: number) => (
                  <Text key={index} fontSize="xs">
                    • {error.path}: {error.message}
                  </Text>
                ))}
                {errors.length > 5 && (
                  <Text fontSize="xs" color="gray.600">
                    ... and {errors.length - 5} more error(s)
                  </Text>
                )}
              </VStack>
            </Alert>
          )}

          <Button
            colorScheme="blue"
            onClick={handleSimulate}
            isDisabled={!requestJson.trim()}
          >
            Simulate Request
          </Button>
        </VStack>

        {/* Response Section */}
        <VStack align="stretch" spacing={3}>
          <Text fontWeight="bold" fontSize="md">
            Response ({responseMessage.name})
          </Text>

          <ResponseViewer
            response={response}
            error={responseError}
            loading={loading}
          />
        </VStack>
      </SimpleGrid>

      <Divider />

      {/* Usage Tips */}
      <Box p={3} bg="gray.50" borderRadius="md" fontSize="sm">
        <Text fontWeight="medium" mb={2}>Tips:</Text>
        <VStack align="stretch" spacing={1} pl={4}>
          <Text>• Click "Generate Sample" to reset the request to default values</Text>
          <Text>• Validation checks field types, required fields, and message structure</Text>
          <Text>• The mock response demonstrates the expected response format</Text>
          <Text>• In a real implementation, this would call your actual gRPC service</Text>
        </VStack>
      </Box>
    </VStack>
  );
};
