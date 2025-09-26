import React from "react";
import { Flex, Spinner, Text, VStack } from "@chakra-ui/react";

interface PageLoaderProps {
  text?: string;
}

export function PageLoader({ text = "Loading..." }: PageLoaderProps) {
  return (
    <Flex justify="center" align="center" h="calc(100vh - 200px)">
      <VStack spacing={4}>
        <Spinner size="xl" color="purple.500" />
        <Text color="gray.500">{text}</Text>
      </VStack>
    </Flex>
  );
}
