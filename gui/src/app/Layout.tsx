import React, { useState } from "react";
import { Flex, Box } from "@chakra-ui/react";
import { Header } from "./components/Header";
import { Sidebar, SidebarContext } from './components/sidebar'

// Define sidebar widths for consistency
const SIDEBAR_WIDTH = "240px"; // 60 * 4px
const SIDEBAR_WIDTH_COLLAPSED = "80px"; // 20 * 4px

export function Layout({ children }: { children: React.ReactNode }) {
  const [isCollapsed, setIsCollapsed] = useState(false);
  const toggleSidebar = () => setIsCollapsed(!isCollapsed);

  return (
    // Provide the sidebar state to all children
    <SidebarContext.Provider value={{ isCollapsed, toggleSidebar }}>
      <Flex direction="column" minH="100vh" bg="gray.100">
        <Header />
        <Sidebar />
        <Box
          as="main"
          flex="1"
          // The left margin now dynamically changes based on the sidebar's state
          ml={{ base: 0, md: isCollapsed ? SIDEBAR_WIDTH_COLLAPSED : SIDEBAR_WIDTH }} 
          mt="64px"
          transition="margin-left .2s ease-in-out"
        >
          {children}
        </Box>
      </Flex>
    </SidebarContext.Provider>
  );
}

