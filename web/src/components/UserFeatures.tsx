import React from 'react';
import {
  Box,
  Container,
  Heading,
  SimpleGrid,
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  VStack,
} from '@chakra-ui/react';
import { Link as RouterLink } from 'react-router-dom';
import { ChevronRightIcon } from '@chakra-ui/icons';
import { SavedSearches } from './SavedSearches';
import { Bookmarks } from './Bookmarks';

export const UserFeatures: React.FC = () => {
  return (
    <Container maxW="container.xl" py={8}>
      <VStack align="stretch" spacing={6}>
        <Breadcrumb spacing="8px" separator={<ChevronRightIcon color="gray.500" />}>
          <BreadcrumbItem>
            <BreadcrumbLink as={RouterLink} to="/">
              Home
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbItem isCurrentPage>
            <BreadcrumbLink>My Library</BreadcrumbLink>
          </BreadcrumbItem>
        </Breadcrumb>

        <Heading size="lg">My Library</Heading>

        <SimpleGrid columns={{ base: 1, md: 2 }} spacing={8}>
          <Box bg="white" p={6} borderRadius="lg" borderWidth={1} shadow="sm">
            <SavedSearches />
          </Box>

          <Box bg="white" p={6} borderRadius="lg" borderWidth={1} shadow="sm">
            <Bookmarks />
          </Box>
        </SimpleGrid>
      </VStack>
    </Container>
  );
};
