// 📁 src/features/dashboard/components/SystemHealth.tsx
import React from "react";
import {
  Box,
  Text,
  VStack,
  HStack,
  Icon,
  Badge,
  Divider,
} from "@chakra-ui/react";
import {
  FiCheckCircle,
  FiAlertTriangle,
  FiXCircle,
} from "react-icons/fi";
import { SystemHealthMetric } from "../../../shared/types/dashboard"

const statusIcons: Record<string, React.ElementType> = {
  healthy: FiCheckCircle,
  warning: FiAlertTriangle,
  critical: FiXCircle,
};

const statusColors: Record<string, string> = {
  healthy: "green",
  warning: "yellow",
  critical: "red",
};

interface SystemHealthProps {
  metrics: SystemHealthMetric[];
}

export function SystemHealth({ metrics }: SystemHealthProps) {
  return (
    <Box
      bg="white"
      borderWidth="1px"
      borderColor="gray.200"
      borderRadius="xl"
      p={6}
    >
      <Text fontSize="lg" fontWeight="bold" mb={4}>
        System Health
      </Text>
      <VStack spacing={4} align="stretch">
        {metrics.map((metric, index) => (
          <React.Fragment key={metric.name}>
            <HStack justify="space-between" align="center">
              <VStack align="flex-start" spacing={1}>
                <Text fontWeight="medium" fontSize="sm">
                  {metric.name}
                </Text>
                <Text fontSize="xs" color="gray.600">
                  {metric.description}
                </Text>
              </VStack>
              <VStack align="flex-end" spacing={1}>
                <Badge
                  colorScheme={statusColors[metric.status]}
                  variant="subtle"
                >
                  <Icon as={statusIcons[metric.status]} boxSize={3} mr={1} />
                  {metric.status}
                </Badge>
                <Text fontSize="sm" fontWeight="medium">
                  {metric.value}
                </Text>
              </VStack>
            </HStack>
            {index < metrics.length - 1 && <Divider />}
          </React.Fragment>
        ))}
      </VStack>
    </Box>
  );
}
