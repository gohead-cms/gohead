/* ---------- layouts/PageShell.tsx ----------
   One generic shell you wrap every routed page in.
   It gives you: fixed header, full-width scroll-able content. */

import { Flex, Box } from "@chakra-ui/react";
import Header from "./Header";
// We no longer need to import the Sidebar
// import Sidebar from "./Sidebar"; 

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <Flex direction="column" minH="100vh">
      <Header />
      {/* The Flex wrapper with direction="row" has been removed as it's no longer needed.
        The Box component now directly becomes the second child of the main column Flex.
      */}
      <Box
        as="main"
        // REMOVED: The margin-left property is gone, allowing the content to take the full width.
        // ml={{ base: 0, md: 60 }} 
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