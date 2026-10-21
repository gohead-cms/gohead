import type { IconType } from "react-icons";
import { Link as RouterLink } from "react-router-dom";
import { Box, Flex, Icon, LinkBox, LinkOverlay, Text } from "@chakra-ui/react";

interface StatCardProps {
  label: string;
  value: string | number;
  detail?: string;
  icon: IconType;
  to: string;
  color?: string;
}

/** A count with a short detail line; the whole card links to its page. */
export function StatCard({ label, value, detail, icon, to, color = "purple" }: StatCardProps) {
  return (
    <LinkBox bg="white" borderWidth="1px" borderRadius="lg" p={5} _hover={{ borderColor: `${color}.300` }} transition="border-color 0.15s" minW={0}>
      <Flex justify="space-between" align="start" gap={3}>
        <Box minW={0}>
          <LinkOverlay asChild>
            <RouterLink to={to}>
              <Text fontSize="sm" color="gray.500">
                {label}
              </Text>
            </RouterLink>
          </LinkOverlay>
          <Text fontSize="2xl" fontWeight="bold" lineHeight="short">
            {typeof value === "number" ? value.toLocaleString() : value}
          </Text>
          {detail && (
            <Text fontSize="xs" color="gray.500" lineClamp={1}>
              {detail}
            </Text>
          )}
        </Box>
        <Flex p={2} bg={`${color}.50`} borderRadius="md" flexShrink={0}>
          <Icon as={icon} boxSize={5} color={`${color}.500`} />
        </Flex>
      </Flex>
    </LinkBox>
  );
}
