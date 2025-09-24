import { Flex, Box } from "@chakra-ui/react";
import  Header  from "./Header";
import { Sidebar } from "./Sidebar";

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <Flex direction="column" minH="100vh" bg="gray.100">
      <Header />
      <Sidebar />
      <Box
        as="main"
        flex="1"
        ml={{ base: 0, md: 60 }} 
        mt="64px" // Offset content by header height
      >
        {children}
      </Box>
    </Flex>
  );
}
