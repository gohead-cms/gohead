import React from "react";
import {
  Box,
  Flex,
  Icon,
  Button,
  useDisclosure,
  Drawer,
  DrawerOverlay,
  DrawerContent,
  DrawerCloseButton,
  DrawerHeader,
  DrawerBody,
  DrawerFooter,
  IconButton,
} from "@chakra-ui/react";
import {
  FiHome,
  FiTrendingUp,
  FiCompass,
  FiStar,
  FiSettings,
  FiMenu,
  FiDatabase,
  FiBarChart2,
} from 'react-icons/fi';
import { IconType } from 'react-icons';

interface LinkItemProps {
  name: string;
  icon: IconType;
}

const LinkItems: Array<LinkItemProps> = [
  //{ name: 'Home', icon: FiHome },
  { name: 'Data', icon: FiDatabase },
  // { name: 'Data', icon: FiBarChart2 },
  // { name: 'Settings', icon: FiSettings },
];

export default function SimpleSidebar() {
  // Correctly destructure isOpen, onOpen, and onClose from useDisclosure
  const { isOpen, onOpen, onClose } = useDisclosure();

  return (
    <Box minH="100vh" bg="gray.100">
      {/* Mobile menu button to open the drawer */}
      <IconButton
        display={{ base: 'flex', md: 'none' }}
        onClick={onOpen}
        variant="outline"
        aria-label="open menu"
        icon={<FiMenu />}
        position="fixed"
        top="4"
        left="4"
        zIndex="10"
      />

      {/* Desktop sidebar */}
      <SidebarContent display={{ base: "none", md: "block" }} />

      {/* Mobile drawer with correct Chakra UI syntax */}
      <Drawer
        isOpen={isOpen}
        placement="left"
        onClose={onClose}
      >
        <DrawerOverlay />
        <DrawerContent>
          <DrawerCloseButton />
          <DrawerHeader>
            <Flex align="center">
              <Icon as={FiMenu} mr={2} />
              <Box>Menu</Box>
            </Flex>
          </DrawerHeader>
          <DrawerBody>
            <SidebarContent onClose={onClose} />
          </DrawerBody>
          <DrawerFooter>
            <Button colorScheme="blue" mr={3} onClick={onClose}>
              Close
            </Button>
          </DrawerFooter>
        </DrawerContent>
      </Drawer>
    </Box>
  );
}

function SidebarContent(props: { onClose?: () => void } & any) {
  return (
    <Box
      bg="white"
      w={{ base: "full", md: 60 }}      // 60 = 240px
      pos={{ base: "relative", md: "fixed" }}
      left={{ base: "auto", md: 0 }}
      top={{ base: "auto", md: 16 }}                          // Offset by header height (64px = 16)
      h={{ base: "auto", md: `calc(100vh - 64px)` }}          // Fill viewport minus header
      borderRight="1px solid"
      borderColor="gray.200"
      zIndex={9}
      {...props}
    >
      {LinkItems.map((link) => (
        <NavItem key={link.name} icon={link.icon}>
          {link.name}
        </NavItem>
      ))}
    </Box>
  );
}

function NavItem({ icon, children }: { icon: any; children: React.ReactNode }) {
  return (
    <Flex
      align="center"
      px={6}
      py={3}
      cursor="pointer"
      role="group"
      _hover={{ bg: "gray.200", color: "blue.600" }}
      fontWeight="medium"
    >
      <Icon mr={4} as={icon} />
      {children}
    </Flex>
  );
}
