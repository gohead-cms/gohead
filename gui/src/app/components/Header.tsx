import { Box, Flex, Text, Button, Icon } from "@chakra-ui/react";
import { Menu, MenuButton, MenuList, MenuItem } from "@chakra-ui/react";
import { Avatar } from "@chakra-ui/react";
import { useAuthStore } from "../../services/auth";
import { useNavigate } from "react-router-dom";
import { FiLogOut } from "react-icons/fi";
import React from 'react';

export function Header() {
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate("/", { replace: true });
  };

  const user = { name: "SUDO", avatarUrl: "https://api.dicebear.com/7.x/adventurer/svg?seed=GoHead" };

  return (
    <Flex
      as="header"
      align="center"
      justify="space-between"
      bg="white"
      px={6}
      h={16}
      borderBottom="1px solid"
      borderColor="gray.200"
      position="fixed"
      top={0}
      left={0}
      right={0}
      width="100vw"
      zIndex={100}
    >
      <Flex align="center" gap={3}>
        <img
          src="/gohead_logo.svg"
          alt="GoHead Logo"
          style={{ height: 32, width: 32 }}
        />
        <Text fontSize="lg" fontWeight="bold">
          GoHead!
        </Text>
      </Flex>
      <Menu>
        {/* MenuButton is the trigger for the MenuList */}
        <MenuButton as={Button} variant="ghost" px={2} py={1} borderRadius="full">
          {/* The Chakra Avatar component handles both the image and fallback text */}
          <Avatar
            size="sm"
            name={user.name}
            src={user.avatarUrl}
          />
        </MenuButton>
        <MenuList>
          <Box px={4} py={2}>
            <Text fontWeight="bold">{user.name}</Text>
          </Box>
          <MenuItem onClick={handleLogout}>
            <Flex align="center">
              <Icon as={FiLogOut} fontSize="md" mr={2} />
              Sign out
            </Flex>
          </MenuItem>
        </MenuList>
      </Menu>
    </Flex>
  );
}
