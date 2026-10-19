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
  type BoxProps,
  HStack,
  Badge,
} from "@chakra-ui/react";
import {
  FiHome,
  FiDatabase,
  FiSettings,
  FiMenu,
  FiGrid,
  FiActivity,
  FiChevronDown,
  FiImage,
  FiUsers,
  FiKey,
  FiSend,
  FiLayers,
  FiGlobe,
  FiShield,
  FiCheckSquare,
} from 'react-icons/fi';
import { type IconType } from 'react-icons';
import { NavLink, useLocation } from "react-router-dom";
import { LiaProjectDiagramSolid } from "react-icons/lia";
import { LuBrain } from "react-icons/lu";
import { VscRobot } from "react-icons/vsc";
import { useMe } from "../../../features/auth/me";
import { MEDIA } from "../../../features/auth/permissions";
import { usePendingCount } from "../../../features/approvals/useProposals";

// --- Menu Structure ---
interface NavItemProps {
  name: string;
  icon: IconType;
  href?: string;
  children?: NavItemProps[];
  /** Only shown to admins (the pages behind it need the admin role). */
  adminOnly?: boolean;
  /** Only shown when the signed-in role has this permission (review: can change some content). */
  needs?: "content" | "media" | "review";
  /** A count shown next to the name. */
  badge?: "approvals";
}

/** Counts shown next to menu items, by NavItemProps.badge. */
type Badges = Partial<Record<NonNullable<NavItemProps["badge"]>, number | null>>;

const LinkItems: Array<NavItemProps> = [
  { name: 'Dashboard', icon: FiHome, href: '/dashboard' },
  {
    name: 'Content',
    icon: FiDatabase,
    children: [
      { name: 'Content Editor', icon: FiGrid, href: '/contrib', needs: 'content' },
      { name: 'Media Library', icon: FiImage, href: '/media', needs: 'media' },
      { name: 'Approvals', icon: FiCheckSquare, href: '/approvals', needs: 'review', badge: 'approvals' },
    ],
  },
  {
    name: 'Content Model',
    icon: FiLayers,
    adminOnly: true,
    children: [
      { name: 'Types', icon: FiLayers, href: '/model' },
      { name: 'Schema Designer', icon: LiaProjectDiagramSolid, href: '/workspace' },
    ],
  },
  {
    name: 'AI',
    icon: LuBrain,
    adminOnly: true,
    children: [
      { name: 'Agents', icon: VscRobot, href: '/agents' },
      { name: 'Activity', icon: FiActivity, href: '/activity' },
    ],
  },
  {
    name: 'Settings',
    icon: FiSettings,
    adminOnly: true,
    children: [
      { name: 'Users', icon: FiUsers, href: '/settings/users' },
      { name: 'Roles', icon: FiShield, href: '/settings/roles' },
      { name: 'API Tokens', icon: FiKey, href: '/settings/tokens' },
      { name: 'Webhooks', icon: FiSend, href: '/settings/webhooks' },
      { name: 'Locales', icon: FiGlobe, href: '/settings/locales' },
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
  const { me, isAdmin, canReadContent, canReview, can } = useMe();
  // Until the account is known, content pages stay listed (as before roles);
  // approvals wait for it (viewers never review).
  const allowed = (l: NavItemProps) =>
    (isAdmin || !l.adminOnly) &&
    (l.needs !== "content" || !me || canReadContent) &&
    (l.needs !== "media" || !me || can("read", MEDIA)) &&
    (l.needs !== "review" || (!!me && canReview));
  const badges: Badges = { approvals: usePendingCount(!!me && canReview) };
  // A section is shown when at least one of its pages is.
  const visibleLinks = LinkItems.filter(allowed)
    .map((l) => (l.children ? { ...l, children: l.children.filter(allowed) } : l))
    .filter((l) => !l.children || l.children.length > 0);
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
        {visibleLinks.map((link) => (
          <NavItem key={link.name} icon={link.icon} href={link.href} childrenItems={link.children} onNavItemClick={onClose} badges={badges} badge={link.badge}>
            {link.name}
          </NavItem>
        ))}
      </Box>
    </Box>
  );
}

// --- Navigation Item Component ---
function NavItem({ icon, children, href, childrenItems, onNavItemClick, badges, badge }: { icon: IconType; children: React.ReactNode; href?: string; childrenItems?: NavItemProps[]; onNavItemClick?: () => void; badges?: Badges; badge?: NavItemProps["badge"] }) {
  const { isOpen, onToggle } = useDisclosure({ defaultIsOpen: true });
  const location = useLocation();
  
  const isParentActive = childrenItems?.some(child => child.href && location.pathname.startsWith(child.href));

  const handleLinkClick = () => {
    if (childrenItems) {
      onToggle();
    } else if (onNavItemClick) {
      onNavItemClick();
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

  // Hooks first, before any branch.
  const hoverStyles = {
    bg: useColorModeValue('gray.100', 'gray.700'),
    color: useColorModeValue('gray.900', 'white'),
  };
  const activeBg = useColorModeValue('purple.50', 'purple.900');
  const activeColor = useColorModeValue('purple.700', 'white');
  const childBorder = useColorModeValue('gray.200', 'gray.700');
  const count = badge ? badges?.[badge] : null;

  if (childrenItems) {
    return (
      <Box>
        <Flex
          {...commonStyles}
          onClick={handleLinkClick}
          justifyContent="space-between"
          bg={isParentActive ? activeBg : 'transparent'}
          color={isParentActive ? activeColor : 'inherit'}
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
          <Box pl={1} py={1} borderLeft="1px solid" borderColor={childBorder} ml={2} mr={4}>
            {childrenItems.map((child) => (
              <NavItem
                key={child.name}
                icon={child.icon}
                href={child.href}
                childrenItems={child.children}
                onNavItemClick={onNavItemClick}
                badges={badges}
                badge={child.badge}
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
        bg: activeBg,
        color: activeColor,
        fontWeight: 'semibold',
      }}
    >
      <HStack flex="1">
        <Icon mr="2" fontSize="16" as={icon} />
        <Text>{children}</Text>
        {count ? (
          <Badge ml="auto" colorScheme="orange" variant="solid" borderRadius="full" px={2} title={`${count} pending`} data-testid="approvals-count">
            {count > 99 ? "99+" : count}
          </Badge>
        ) : null}
      </HStack>
    </Flex>
  );
}

