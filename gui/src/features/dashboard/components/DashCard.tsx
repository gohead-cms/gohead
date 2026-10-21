import type { ReactNode } from "react";
import { Link as RouterLink } from "react-router-dom";
import { Box, Flex, Heading, Link } from "@chakra-ui/react";

/** A white dashboard card with a title and an optional "see more" link. */
export function DashCard({ title, link, children }: { title: string; link?: { to: string; label: string }; children: ReactNode }) {
  return (
    <Box bg="white" borderWidth="1px" borderRadius="lg" p={5} minW={0}>
      <Flex justify="space-between" align="baseline" mb={4} gap={3}>
        <Heading size="md" fontWeight="bold">{title}</Heading>
        {link && (
          <Link asChild fontSize="sm" color="purple.600" whiteSpace="nowrap">
            <RouterLink to={link.to}>{link.label}</RouterLink>
          </Link>
        )}
      </Flex>
      {children}
    </Box>
  );
}
