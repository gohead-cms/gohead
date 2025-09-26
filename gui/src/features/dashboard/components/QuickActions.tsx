// 📁 src/features/dashboard/components/QuickActions.tsx
import React from "react";
import {
  Box,
  Text,
  SimpleGrid,
  Button,
  Icon,
  VStack,
  HStack,
} from "@chakra-ui/react";
import {
  FiPlus,
  FiDatabase,
  FiCpu,
  FiUpload,
  FiActivity,
} from "react-icons/fi";
import { QuickAction } from "../../../shared/types/dashboard"

const quickActions: QuickAction[] = [
  {
    name: "Create Collection",
    description: "Add new content collection",
    icon: FiDatabase,
    href: "/collections/new",
    color: "blue",
  },
  {
    name: "Add Agent",
    description: "Configure new AI agent",
    icon: FiCpu,
    href: "/agents/new",
    color: "green",
  },
  {
    name: "Import Content",
    description: "Bulk import data",
    icon: FiUpload,
    href: "/import",
    color: "purple",
  },
  {
    name: "View Logs",
    description: "System activity logs",
    icon: FiActivity,
    href: "/logs",
    color: "orange",
  },
];

export function QuickActions() {
  return (
    <Box
      bg="white"
      borderWidth="1px"
      borderColor="gray.200"
      borderRadius="xl"
      p={6}
    >
      <Text fontSize="lg" fontWeight="bold" mb={4}>
        Quick Actions
      </Text>
      <SimpleGrid columns={{ base: 1, md: 2 }} spacing={4}>
        {quickActions.map((action) => (
          <Button
            key={action.name}
            as="a"
            href={action.href}
            variant="outline"
            colorScheme={action.color}
            h="auto"
            p={4}
            justifyContent="flex-start"
          >
            <VStack align="flex-start" spacing={1}>
              <HStack spacing={2}>
                <Icon as={action.icon} boxSize={4} />
                <Text fontWeight="medium" fontSize="sm">
                  {action.name}
                </Text>
              </HStack>
              <Text fontSize="xs" color="gray.600">
                {action.description}
              </Text>
            </VStack>
          </Button>
        ))}
      </SimpleGrid>
    </Box>
  );
}
