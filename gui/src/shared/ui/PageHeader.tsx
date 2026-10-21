import { type ReactNode } from "react";
import { Box, Flex, Heading, HStack, Text } from "@chakra-ui/react";

/** Title, one-line description and actions at the top of a page. */
export function PageHeader({ title, description, actions }: { title: ReactNode; description?: ReactNode; actions?: ReactNode }) {
  return (
    <Flex justify="space-between" align={{ base: "start", md: "center" }} gap={4} mb={6} direction={{ base: "column", md: "row" }}>
      <Box>
        <Heading size="2xl" fontWeight="bold">{title}</Heading>
        {description && (
          <Text color="gray.500" mt={1}>
            {description}
          </Text>
        )}
      </Box>
      {actions && <HStack gap={2}>{actions}</HStack>}
    </Flex>
  );
}

/** Standard page padding and width. */
export function Page({ children, maxW = "7xl" }: { children: ReactNode; maxW?: string }) {
  return (
    <Box px={{ base: 4, md: 8 }} py={8} maxW={maxW} mx="auto" w="full">
      {children}
    </Box>
  );
}
