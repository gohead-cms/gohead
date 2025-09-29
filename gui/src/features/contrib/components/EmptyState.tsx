import React from 'react';
import { Box, VStack, Icon, Heading, Text, useColorModeValue } from '@chakra-ui/react';
import { IconType } from 'react-icons';
import { FiInbox } from 'react-icons/fi';

interface EmptyStateProps {
  icon?: IconType;
  title?: string;
  description?: string;
}

export function EmptyState({
  icon = FiInbox,
  title = "This collection is empty.",
  description = "Get started by creating your first entry.",
}: EmptyStateProps) {
  const bgColor = useColorModeValue("gray.50", "gray.800");
  const borderColor = useColorModeValue("gray.200", "gray.700");

  return (
    <Box
      textAlign="center"
      py={20}
      px={6}
      bg={bgColor}
      borderRadius="lg"
      border="1px"
      borderColor={borderColor}
    >
      <VStack spacing={4}>
        <Icon as={icon} boxSize={16} color="gray.400" />
        <Heading size="md" color={useColorModeValue("gray.700", "gray.200")}>
          {title}
        </Heading>
        <Text color="gray.500">{description}</Text>
      </VStack>
    </Box>
  );
}
