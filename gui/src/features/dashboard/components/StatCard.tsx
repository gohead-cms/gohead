import React from "react";
import {
  Box,
  Flex,
  Text,
  Icon,
  Stat,
  StatLabel,
  StatNumber,
  StatHelpText,
  StatArrow,
} from "@chakra-ui/react";

interface StatCardProps {
  title: string;
  value: string | number;
  icon: React.ElementType;
  change?: number;
  colorScheme?: string;
}

export function StatCard({ title, value, icon, change, colorScheme = "blue" }: StatCardProps) {
  return (
    <Box
      p={6}
      bg="white"
      borderWidth="1px"
      borderColor="gray.200"
      borderRadius="xl"
      shadow="sm"
    >
      <Flex justify="space-between" align="center">
        <Stat>
          <StatLabel fontSize="sm" color="gray.600">
            {title}
          </StatLabel>
          <StatNumber fontSize="2xl" fontWeight="bold">
            {value}
          </StatNumber>
          {change !== undefined && (
            <StatHelpText>
              <StatArrow type={change >= 0 ? "increase" : "decrease"} />
              {Math.abs(change)}%
            </StatHelpText>
          )}
        </Stat>
        <Box
          p={3}
          bg={`${colorScheme}.50`}
          borderRadius="lg"
        >
          <Icon as={icon} boxSize={6} color={`${colorScheme}.500`} />
        </Box>
      </Flex>
    </Box>
  );
}
