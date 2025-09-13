import { Flex, Text, Icon } from "@chakra-ui/react";
import { FiInbox } from "react-icons/fi";

function NoCollections() {
  return (
    <Flex
      direction="column"
      align="center"
      justify="center"
      h="60vh"
      color="gray.500"
      gap={4}
    >
      <Icon as={FiInbox} boxSize={16} color="gray.300" />
      <Text fontSize="2xl" fontWeight="semibold">
        No collections found
      </Text>
      <Text fontSize="md" color="gray.400">
        Get started by creating your first collection!
      </Text>
    </Flex>
  );
}

export default NoCollections;
