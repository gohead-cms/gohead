import { Flex, Box } from "@chakra-ui/react";
import Header from "./Header";


export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <Flex direction="column" minH="100vh">
      <Header />
      {}
      <Box
        as="main"
        flex="1"
        minH="calc(100vh - 64px)"
        pt={16} // This padding-top is important to prevent content from hiding under the fixed Header.
        position="relative"
        // Note: overflowY="auto" might not be needed here if the ReactFlow component handles its own scrolling.
        // You can keep it or remove it based on desired behavior.
      >
        {children}
      </Box>
    </Flex>
  );
}