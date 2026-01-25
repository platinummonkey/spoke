import React from 'react';
import {
  Box,
  VStack,
  Accordion,
  AccordionItem,
  AccordionButton,
  AccordionPanel,
  AccordionIcon,
  Text,
  Badge,
  HStack,
  Divider,
  Alert,
  AlertIcon,
} from '@chakra-ui/react';
import { Service, ProtoFile } from '../types';
import { MethodDetail } from './MethodDetail';

interface ServiceBrowserProps {
  services: Service[];
  files: ProtoFile[];
}

interface ServiceItemProps {
  service: Service;
  files: ProtoFile[];
}

const ServiceItem: React.FC<ServiceItemProps> = ({ service, files }) => {
  return (
    <Box>
      <VStack align="stretch" spacing={3}>
        {/* Service Header */}
        <Box>
          <HStack justify="space-between">
            <Text fontSize="lg" fontWeight="bold" color="blue.600">
              {service.name}
            </Text>
            <Badge colorScheme="blue" fontSize="sm">
              {service.methods.length} {service.methods.length === 1 ? 'method' : 'methods'}
            </Badge>
          </HStack>
        </Box>

        {/* Methods List */}
        <Accordion allowToggle>
          {service.methods.map((method) => (
            <AccordionItem key={method.name}>
              <AccordionButton>
                <Box flex="1" textAlign="left">
                  <HStack spacing={2}>
                    <Text fontWeight="medium">{method.name}</Text>
                    {(method.clientStreaming || method.serverStreaming) && (
                      <Badge
                        size="sm"
                        colorScheme={
                          method.clientStreaming && method.serverStreaming
                            ? 'purple'
                            : method.clientStreaming
                            ? 'blue'
                            : 'green'
                        }
                      >
                        {method.clientStreaming && method.serverStreaming
                          ? 'Bidi'
                          : method.clientStreaming
                          ? 'Client Stream'
                          : 'Server Stream'}
                      </Badge>
                    )}
                  </HStack>
                </Box>
                <AccordionIcon />
              </AccordionButton>
              <AccordionPanel pb={4}>
                <MethodDetail method={method} files={files} />
              </AccordionPanel>
            </AccordionItem>
          ))}
        </Accordion>
      </VStack>
    </Box>
  );
};

export const ServiceBrowser: React.FC<ServiceBrowserProps> = ({ services, files }) => {
  if (!services || services.length === 0) {
    return (
      <Alert status="info">
        <AlertIcon />
        No services found in this module. This module may only contain message and enum definitions.
      </Alert>
    );
  }

  return (
    <VStack align="stretch" spacing={6}>
      <Box>
        <Text fontSize="sm" color="gray.600">
          Browse {services.length} {services.length === 1 ? 'service' : 'services'} and their methods.
          Click on a method to view request/response details.
        </Text>
      </Box>

      <Divider />

      <Accordion allowMultiple defaultIndex={[0]}>
        {services.map((service) => (
          <AccordionItem key={service.name}>
            <AccordionButton>
              <Box flex="1" textAlign="left">
                <HStack spacing={3}>
                  <Text fontSize="md" fontWeight="bold">
                    {service.name}
                  </Text>
                  <Badge colorScheme="blue">
                    {service.methods.length} {service.methods.length === 1 ? 'method' : 'methods'}
                  </Badge>
                </HStack>
              </Box>
              <AccordionIcon />
            </AccordionButton>
            <AccordionPanel pb={4}>
              <ServiceItem service={service} files={files} />
            </AccordionPanel>
          </AccordionItem>
        ))}
      </Accordion>
    </VStack>
  );
};
