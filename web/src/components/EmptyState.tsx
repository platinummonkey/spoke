import React from 'react';
import {
  Box,
  VStack,
  Heading,
  Text,
  Button,
  Icon,
} from '@chakra-ui/react';
import { SearchIcon, WarningIcon } from '@chakra-ui/icons';

interface EmptyStateProps {
  title: string;
  description: string;
  icon?: 'search' | 'warning';
  actionLabel?: string;
  onAction?: () => void;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  title,
  description,
  icon = 'search',
  actionLabel,
  onAction,
}) => {
  const IconComponent = icon === 'search' ? SearchIcon : WarningIcon;

  return (
    <Box
      textAlign="center"
      py={12}
      px={6}
      bg="gray.50"
      borderRadius="md"
      borderWidth={1}
      borderColor="gray.200"
    >
      <VStack spacing={4}>
        <Icon
          as={IconComponent}
          boxSize={12}
          color="gray.400"
          aria-hidden="true"
        />
        <Heading size="md" color="gray.700">
          {title}
        </Heading>
        <Text color="gray.600" maxW="400px">
          {description}
        </Text>
        {actionLabel && onAction && (
          <Button
            colorScheme="blue"
            onClick={onAction}
            mt={2}
            aria-label={actionLabel}
          >
            {actionLabel}
          </Button>
        )}
      </VStack>
    </Box>
  );
};
