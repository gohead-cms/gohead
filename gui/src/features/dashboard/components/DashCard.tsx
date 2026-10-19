import type { ReactNode } from "react";
import { Link as RouterLink } from "react-router-dom";
import { Box, Flex, Heading, Link } from "@chakra-ui/react";

/** A white dashboard card with a title and an optional "see more" link. */
export function DashCard({ title, link, children }: { title: string; link?: { to: string; label: string }; children: ReactNode }) {
  return (
    <Box bg="white" borderWidth="1px" borderRadius="lg" p={5} minW={0}>
      <Flex justify="space-between" align="baseline" mb={4} gap={3}>
        <Heading size="sm">{title}</Heading>
        {link && (
          <Link as={RouterLink} to={link.to} fontSize="sm" color="purple.600" whiteSpace="nowrap">
            {link.label}
          </Link>
        )}
      </Flex>
      {children}
    </Box>
  );
}
