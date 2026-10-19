import { type ReactNode } from "react";
import { Box, Icon, Text, VStack } from "@chakra-ui/react";
import { type IconType } from "react-icons";

/** Shown instead of an empty list: what is missing and how to add it. */
export function EmptyState({ icon, title, children, action }: { icon?: IconType; title: string; children?: ReactNode; action?: ReactNode }) {
  return (
    <Box borderWidth="1px" borderStyle="dashed" borderRadius="lg" py={12} px={6} textAlign="center" bg="white">
      <VStack spacing={3}>
        {icon && <Icon as={icon} boxSize={8} color="gray.400" />}
        <Text fontWeight="semibold">{title}</Text>
        {children && (
          <Text color="gray.500" maxW="md">
            {children}
          </Text>
        )}
        {action}
      </VStack>
    </Box>
  );
}
