import React, { ReactNode } from "react";
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
  HStack,
} from "@chakra-ui/react";
import {
  FiHome,
  FiDatabase,
  FiCpu,
  FiEdit2,
  FiSettings,
  FiMenu,
  FiLayout,
  FiFileText,
  FiUsers,
  FiCode,
  FiChevronDown,
  FiImage,
  FiGrid,
  FiCloud,
  FiActivity,
  FiBarChart,
  FiTrendingUp
} from 'react-icons/fi';
import { IconBaseProps, IconType } from 'react-icons';
import { NavLink, useLocation } from "react-router-dom";
import { LiaProjectDiagramSolid } from "react-icons/lia";
import { LuBrain } from "react-icons/lu";
import { VscRobot } from "react-icons/vsc";
import { TbMathFunction } from "react-icons/tb";
import { PiPlug } from "react-icons/pi";

// --- Menu Structure ---
interface NavItemProps {
  name: string;
  icon: IconType;
  href?: string;
  children?: NavItemProps[];
}

const LinkItems: Array<NavItemProps> = [
  { name: 'Dashboard', icon: FiHome, href: '/dashboard' },
  {
    name: 'Data Management',
    icon: FiDatabase,
    children: [
      { name: 'Collections', icon: FiFileText, href: '/collections' },
      { name: 'Schema Designer', icon: LiaProjectDiagramSolid, href: '/data/workspace' },
      { name: 'Contribution', icon: FiEdit2, href: '/data/contrib' },
      { name: 'Media Library', icon: FiImage, href: '/dam' },
    ],
  },
  {
    name: 'AI Management',
    icon: LuBrain,
    children: [
        { name: 'Agents', icon: VscRobot, href: '/workspace' },
        { name: 'LLM Primitives', icon: TbMathFunction, href: '/primitives' },
    ],
  },
  {
    name: 'Monitoring',
    icon: FiActivity,
    children: [
      { name: 'Analytics', icon: FiBarChart, href: '/analytics' },
      { name: 'Logs', icon: FiActivity, href: '/logs' },
      { name: 'Agent Performance', icon: FiTrendingUp, href: '/performance' },
    ]
  },
  { name: 'Settings', icon: FiSettings, href: '/settings',
    children: [
        { name: 'LLM Providers', icon: FiCloud, href: '/providers' },
        { name: 'Integration', icon: PiPlug, href: '/integration' },
    ],
  },
];

// --- Main Sidebar Component ---
export function Sidebar() {
  const { isOpen, onOpen, onClose } = useDisclosure();

  return (
    <Box>
      <SidebarContent 
        display={{ base: "none", md: "block" }} 
      />
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

// --- Sidebar Content & Styling ---
interface SidebarContentProps extends BoxProps {
  onClose?: () => void;
}

function SidebarContent({ onClose, ...rest }: SidebarContentProps) {
  return (
    <Box
      transition="3s ease"
      bg={useColorModeValue('gray.50', 'gray.900')}
      borderRight="1px"
      borderRightColor={useColorModeValue('gray.200', 'gray.700')}
      w={{ base: 'full', md: 60 }}
      pos="fixed"
      h="full"
      {...rest}
    >
      <Flex h="20" alignItems="center" mx="8" justifyContent="space-between">
        <Text fontSize="2xl" fontWeight="bold" color="purple.600">
          GoHead
        </Text>
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

// --- Navigation Item Component ---
function NavItem({ icon, children, href, childrenItems, onNavItemClick }: { icon: IconType; children: React.ReactNode; href?: string; childrenItems?: NavItemProps[]; onNavItemClick?: () => void; }) {
  const { isOpen, onToggle } = useDisclosure();
  const location = useLocation();
  
  const isParentActive = childrenItems?.some(child => child.href && location.pathname.startsWith(child.href));

  const handleLinkClick = () => {
    if (childrenItems) {
      onToggle();
    } else if (onNavItemClick) {
      onNavItemClick(); // Close mobile drawer on link click
    }
  };

  const commonStyles = {
    align: "center",
    p: "3",
    mx: "2",
    my: "1",
    borderRadius: "lg",
    role: "group",
    cursor: "pointer",
    fontWeight: "medium",
    fontSize: "sm",
  };

  const hoverStyles = {
    bg: useColorModeValue('gray.100', 'gray.700'),
    color: useColorModeValue('gray.900', 'white'),
  };

  if (childrenItems) {
    return (
      <Box>
        <Flex
          {...commonStyles}
          onClick={handleLinkClick}
          justifyContent="space-between"
          bg={isParentActive ? useColorModeValue('purple.50', 'purple.900') : 'transparent'}
          color={isParentActive ? useColorModeValue('purple.700', 'white') : 'inherit'}
          fontWeight={isParentActive ? 'semibold' : 'medium'}
          _hover={hoverStyles}
        >
          <HStack>
            <Icon mr="2" fontSize="16" as={icon} />
            <Text>{children}</Text>
          </HStack>
          <Icon
            as={FiChevronDown}
            transition="all .25s ease-in-out"
            transform={isOpen ? 'rotate(180deg)' : ''}
          />
        </Flex>
        <Collapse in={isOpen} animateOpacity style={{ marginTop: '0!' }}>
          {/* FIX: Adjusted margin and padding for a tighter look */}
          <Box pl={1} py={1} borderLeft="1px solid" borderColor={useColorModeValue('gray.200', 'gray.700')} ml={2} mr={4}>
            {childrenItems.map((child) => (
              <NavItem
                key={child.name}
                icon={child.icon}
                href={child.href}
                childrenItems={child.children}
                onNavItemClick={onNavItemClick}
              >
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
      {...commonStyles}
      _hover={hoverStyles}
      onClick={handleLinkClick}
      _activeLink={{
        bg: useColorModeValue('purple.50', 'purple.900'),
        color: useColorModeValue('purple.700', 'white'),
        fontWeight: 'semibold',
      }}
    >
      <HStack>
        <Icon mr="2" fontSize="16" as={icon} />
        <Text>{children}</Text>
      </HStack>
    </Flex>
  );
}
