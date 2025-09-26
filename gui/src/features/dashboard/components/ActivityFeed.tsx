import React from "react";
import {
  Box,
  VStack,
  HStack,
  Text,
  Icon,
  Badge,
  Avatar,
  Divider,
} from "@chakra-ui/react";
import {
  FiFileText,
  FiCpu,
  FiSettings,
  FiUser,
  FiCheckCircle,
  FiAlertTriangle,
  FiXCircle,
  FiInfo,
} from "react-icons/fi";
import { ActivityItem } from "../../../shared/types/dashboard"

const activityIcons: Record<string, React.ElementType> = {
  content: FiFileText,
  agent: FiCpu,
  system: FiSettings,
  user: FiUser,
};

const statusIcons: Record<string, React.ElementType> = {
  success: FiCheckCircle,
  warning: FiAlertTriangle,
  error: FiXCircle,
  info: FiInfo,
};

const statusColors: Record<string, string> = {
  success: "green",
  warning: "yellow",
  error: "red",
  info: "blue",
};

interface ActivityFeedProps {
  activities: ActivityItem[];
}

export function ActivityFeed({ activities }: ActivityFeedProps) {
  return (
    <Box
      bg="white"
      borderWidth="1px"
      borderColor="gray.200"
      borderRadius="xl"
      p={6}
    >
      <Text fontSize="lg" fontWeight="bold" mb={4}>
        Recent Activity
      </Text>
      <VStack spacing={4} align="stretch">
        {activities.map((activity, index) => (
          <React.Fragment key={activity.id}>
            <HStack spacing={4} align="flex-start">
              <Avatar
                size="sm"
                bg={`${statusColors[activity.status]}.100`}
                icon={
                  <Icon
                    as={activityIcons[activity.type]}
                    color={`${statusColors[activity.status]}.500`}
                  />
                }
              />
              <Box flex={1}>
                <HStack justify="space-between" mb={1}>
                  <Text fontWeight="medium" fontSize="sm">
                    {activity.title}
                  </Text>
                  <Badge
                    colorScheme={statusColors[activity.status]}
                    variant="subtle"
                    size="sm"
                  >
                    <Icon as={statusIcons[activity.status]} boxSize={3} mr={1} />
                    {activity.status}
                  </Badge>
                </HStack>
                <Text fontSize="xs" color="gray.600" mb={1}>
                  {activity.description}
                </Text>
                <Text fontSize="xs" color="gray.400">
                  {activity.timestamp} {activity.user && `• ${activity.user}`}
                </Text>
              </Box>
            </HStack>
            {index < activities.length - 1 && <Divider />}
          </React.Fragment>
        ))}
      </VStack>
    </Box>
  );
}
