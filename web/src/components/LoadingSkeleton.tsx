import React from 'react';
import {
  Box,
  Skeleton,
  SkeletonText,
  VStack,
  HStack,
} from '@chakra-ui/react';

interface LoadingSkeletonProps {
  type: 'module-list' | 'module-detail' | 'search-results';
  count?: number;
}

export const LoadingSkeleton: React.FC<LoadingSkeletonProps> = ({ type, count = 3 }) => {
  switch (type) {
    case 'module-list':
      return (
        <VStack align="stretch" spacing={4}>
          {Array.from({ length: count }).map((_, i) => (
            <Box key={i} p={4} borderWidth={1} borderRadius="md">
              <Skeleton height="24px" width="200px" mb={3} />
              <SkeletonText noOfLines={2} spacing={2} />
              <HStack mt={3} spacing={2}>
                <Skeleton height="20px" width="80px" />
                <Skeleton height="20px" width="100px" />
              </HStack>
            </Box>
          ))}
        </VStack>
      );

    case 'module-detail':
      return (
        <VStack align="stretch" spacing={6}>
          <Box>
            <Skeleton height="32px" width="300px" mb={2} />
            <SkeletonText noOfLines={2} spacing={2} />
          </Box>
          <Box>
            <Skeleton height="40px" mb={4} />
            <VStack align="stretch" spacing={3}>
              {Array.from({ length: count }).map((_, i) => (
                <Box key={i} p={4} borderWidth={1} borderRadius="md">
                  <Skeleton height="20px" width="150px" mb={2} />
                  <SkeletonText noOfLines={3} spacing={2} />
                </Box>
              ))}
            </VStack>
          </Box>
        </VStack>
      );

    case 'search-results':
      return (
        <VStack align="stretch" spacing={2}>
          {Array.from({ length: count }).map((_, i) => (
            <Box key={i} p={4} borderWidth={1} borderRadius="md">
              <HStack justify="space-between" mb={2}>
                <Skeleton height="20px" width="200px" />
                <Skeleton height="16px" width="60px" />
              </HStack>
              <SkeletonText noOfLines={2} spacing={2} />
              <HStack mt={2} spacing={2}>
                <Skeleton height="18px" width="70px" />
                <Skeleton height="18px" width="90px" />
              </HStack>
            </Box>
          ))}
        </VStack>
      );

    default:
      return null;
  }
};
