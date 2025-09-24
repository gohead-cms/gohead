import React from "react";
import {
  Box,
  Flex,
  Icon,
  useDisclosure,
  Drawer,
  DrawerOverlay,
  DrawerContent,
  DrawerCloseButton,
  IconButton,
  Collapse,
  Text,
  useColorModeValue,
  BoxProps,
} from "@chakra-ui/react";
import {
  FiHome,
  FiDatabase,
  FiCpu,
  FiZap,
  FiSettings,
  FiMenu,
  FiLayout,
  FiFileText,
  FiUsers,
  FiCode,
  FiChevronDown,
} from 'react-icons/fi';
import { IconType } from 'react-icons';
import { NavLink, useLocation } from "react-router-dom";

// Define the structure for nested links
interface NavItemProps {
  name: string;
  icon: IconType;
  href?: string;
  children?: NavItemProps[];
}

// The multi-level menu structure
const LinkItems: Array<NavItemProps> = [
  { name: 'Dashboard', icon: FiHome, href: '/dashboard' },
  {
    name: 'Data Management',
    icon: FiDatabase,
    children: [
      { name: 'Collections', icon: FiFileText, href: '/collections' },
      { name: 'Schema Designer', icon: FiLayout, href: '/workspace' },
    ],
  },
  {
    name: 'Agent Management',
    icon: FiCpu,
    children: [
        { name: 'Agents', icon: FiUsers, href: '/agents' },
        { name: 'LLM Primitives', icon: FiCode, href: '/primitives' },
    ],
  },
  { name: 'Automation', icon: FiZap, href: '/automation' },
  { name: 'Settings', icon: FiSettings, href: '/settings' },
];

export function Sidebar() {
  const { isOpen, onOpen, onClose } = useDisclosure();

  return (
    <Box>
      {/* Desktop Sidebar */}
      <SidebarContent display={{ base: "none", md: "block" }} />
      {/* Mobile Drawer */}
      <Drawer
        isOpen={isOpen}
        placement="left"
        onClose={onClose}
        returnFocusOnClose={false}
        onOverlayClick={onClose}
        size="full"
      >
        <DrawerOverlay />
        <DrawerContent>
          <SidebarContent onClose={onClose} />
        </DrawerContent>
      </Drawer>
      {/* Mobile Menu Button */}
      <IconButton
        display={{ base: 'flex', md: 'none' }}
        onClick={onOpen}
        variant="outline"
        aria-label="open menu"
        icon={<FiMenu />}
        position="fixed"
        top="4"
        left="4"
        zIndex="overlay"
      />
    </Box>
  );
}

interface SidebarContentProps extends BoxProps {
  onClose?: () => void;
}

function SidebarContent({ onClose, ...rest }: SidebarContentProps) {
  return (
    <Box
      transition="3s ease"
      bg={useColorModeValue('white', 'gray.900')}
      borderRight="1px"
      borderRightColor={useColorModeValue('gray.200', 'gray.700')}
      w={{ base: 'full', md: 60 }}
      pos="fixed"
      h="full"
      {...rest}
    >
      <Flex h="20" alignItems="center" mx="8" justifyContent="space-between">
        <Text fontSize="2xl" fontWeight="bold">
          GoHead
        </Text>
        {/* FIX: Conditionally render the close button only when onClose is passed */}
        {onClose && (
            <DrawerCloseButton display={{ base: 'flex', md: 'none' }} onClick={onClose} />
        )}
      </Flex>
      <Box overflowY="auto" h="calc(100% - 80px)" pb={10}>
        {LinkItems.map((link) => (
          <NavItem key={link.name} icon={link.icon} href={link.href} childrenItems={link.children} onNavItemClick={onClose}>
            {link.name}
          </NavItem>
        ))}
      </Box>
    </Box>
  );
}

function NavItem({ icon, children, href, childrenItems, onNavItemClick }: { icon: IconType; children: React.ReactNode; href?: string; childrenItems?: NavItemProps[]; onNavItemClick?: () => void; }) {
  const { isOpen, onToggle } = useDisclosure();
  const location = useLocation();
  const isActive = href ? location.pathname.startsWith(href) : false;

  const handleLinkClick = () => {
    if (onNavItemClick) {
      onNavItemClick();
    }
  };

  const linkStyles = {
    display: 'flex',
    alignItems: 'center',
    padding: '0.75rem 1.5rem',
    margin: '0.25rem 0.5rem',
    borderRadius: 'md',
    textDecoration: 'none',
    color: useColorModeValue('gray.700', 'gray.200'),
    bg: isActive ? useColorModeValue('purple.500', 'purple.300') : 'transparent',
    colorScheme: isActive ? 'white' : 'inherit',
  };

  if (childrenItems) {
    return (
      <Box>
        <Flex
          onClick={onToggle}
          style={linkStyles}
          cursor="pointer"
          justifyContent="space-between"
          _hover={{
            bg: 'purple.400',
            color: 'white',
          }}
        >
          <Flex align="center">
            <Icon mr={4} as={icon} color={isActive ? 'white' : undefined} />
            <Text color={isActive ? 'white' : undefined}>{children}</Text>
          </Flex>
          <Icon
            as={FiChevronDown}
            transition="all .25s ease-in-out"
            transform={isOpen ? 'rotate(180deg)' : ''}
            color={isActive ? 'white' : undefined}
          />
        </Flex>
        <Collapse in={isOpen} animateOpacity>
          <Box pl="12" py="2" borderLeft="1px solid" borderColor={useColorModeValue('gray.200', 'gray.700')} ml={8}>
            {childrenItems?.map((child) => (
              <NavItem key={child.name} icon={child.icon} href={child.href} childrenItems={child.children} onNavItemClick={onNavItemClick}>
                {child.name}
              </NavItem>
            ))}
          </Box>
        </Collapse>
      </Box>
    );
  }

  return (
    <Flex
      as={NavLink}
      to={href || '#'}
      style={linkStyles}
      onClick={handleLinkClick}
      _hover={{
        bg: 'purple.400',
        color: 'white',
      }}
      _activeLink={{
        bg: useColorModeValue('purple.500', 'purple.300'),
        color: 'white',
      }}
    >
      <Flex align="center">
        <Icon mr={4} as={icon} />
        {children}
      </Flex>
    </Flex>
  );
}

