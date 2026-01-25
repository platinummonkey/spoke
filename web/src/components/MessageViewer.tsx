import React from 'react';
import {
  Box,
  VStack,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Text,
  Accordion,
  AccordionItem,
  AccordionButton,
  AccordionPanel,
  AccordionIcon,
  Heading,
} from '@chakra-ui/react';
import { Message, Field } from '../types';
import { FieldTypeVisualizer } from './FieldTypeVisualizer';

interface MessageViewerProps {
  message: Message;
  showTitle?: boolean;
  compact?: boolean;
}

interface FieldRowProps {
  field: Field;
  compact?: boolean;
}

const FieldRow: React.FC<FieldRowProps> = ({ field, compact }) => {
  return (
    <Tr>
      <Td fontWeight="medium">{field.name}</Td>
      <Td>
        <FieldTypeVisualizer type={field.type} label={field.label} />
      </Td>
      {!compact && <Td isNumeric>{field.number}</Td>}
    </Tr>
  );
};

export const MessageViewer: React.FC<MessageViewerProps> = ({
  message,
  showTitle = true,
  compact = false
}) => {
  if (!message.fields || message.fields.length === 0) {
    return (
      <Box p={4} bg="gray.50" borderRadius="md">
        <Text color="gray.500" fontSize="sm">
          No fields defined
        </Text>
      </Box>
    );
  }

  return (
    <VStack align="stretch" spacing={4}>
      {showTitle && (
        <Heading size="sm">{message.name}</Heading>
      )}

      <Table size="sm" variant="simple">
        <Thead>
          <Tr>
            <Th>Field Name</Th>
            <Th>Type</Th>
            {!compact && <Th isNumeric>Field Number</Th>}
          </Tr>
        </Thead>
        <Tbody>
          {message.fields.map((field) => (
            <FieldRow key={field.name} field={field} compact={compact} />
          ))}
        </Tbody>
      </Table>

      {/* Nested Messages */}
      {message.nestedMessages && message.nestedMessages.length > 0 && (
        <Box mt={4}>
          <Heading size="xs" mb={2} color="gray.600">
            Nested Messages
          </Heading>
          <Accordion allowMultiple size="sm">
            {message.nestedMessages.map((nested) => (
              <AccordionItem key={nested.name}>
                <AccordionButton>
                  <Box flex="1" textAlign="left">
                    <Text fontSize="sm" fontWeight="medium">{nested.name}</Text>
                  </Box>
                  <AccordionIcon />
                </AccordionButton>
                <AccordionPanel pb={4}>
                  <MessageViewer message={nested} showTitle={false} compact={compact} />
                </AccordionPanel>
              </AccordionItem>
            ))}
          </Accordion>
        </Box>
      )}

      {/* Nested Enums */}
      {message.nestedEnums && message.nestedEnums.length > 0 && (
        <Box mt={4}>
          <Heading size="xs" mb={2} color="gray.600">
            Nested Enums
          </Heading>
          <Accordion allowMultiple size="sm">
            {message.nestedEnums.map((enumType) => (
              <AccordionItem key={enumType.name}>
                <AccordionButton>
                  <Box flex="1" textAlign="left">
                    <Text fontSize="sm" fontWeight="medium">{enumType.name}</Text>
                  </Box>
                  <AccordionIcon />
                </AccordionButton>
                <AccordionPanel pb={4}>
                  <Table size="sm">
                    <Thead>
                      <Tr>
                        <Th>Name</Th>
                        <Th isNumeric>Value</Th>
                      </Tr>
                    </Thead>
                    <Tbody>
                      {enumType.values.map((value) => (
                        <Tr key={value.name}>
                          <Td>{value.name}</Td>
                          <Td isNumeric>{value.number}</Td>
                        </Tr>
                      ))}
                    </Tbody>
                  </Table>
                </AccordionPanel>
              </AccordionItem>
            ))}
          </Accordion>
        </Box>
      )}
    </VStack>
  );
};
