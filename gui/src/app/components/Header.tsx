import { Badge, Box, Button, Flex, Icon, Menu, MenuButton, MenuDivider, MenuItem, MenuList, Text, Avatar } from "@chakra-ui/react";
import { useNavigate } from "react-router-dom";
import { FiBookOpen, FiCode, FiLogOut } from "react-icons/fi";
import { useAuthStore } from "../../services/auth";
import { API_BASE_URL } from "../../services/api";
import { useMe } from "../../features/auth/me";

export function Header() {
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();
  const { me } = useMe();

  const handleLogout = () => {
    logout();
    navigate("/login", { replace: true });
  };

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
        <img src="/gohead_logo.svg" alt="" style={{ height: 32, width: 32 }} />
        <Text fontSize="lg" fontWeight="bold">
          GoHead
        </Text>
      </Flex>
      <Menu>
        <MenuButton as={Button} variant="ghost" px={2} py={1} borderRadius="full" aria-label="Account menu">
          <Avatar size="sm" name={me?.username ?? "?"} />
        </MenuButton>
        <MenuList>
          <Box px={4} py={2}>
            <Text fontWeight="bold">{me?.username ?? "…"}</Text>
            {me && (
              <Badge mt={1} colorScheme={me.role === "admin" ? "purple" : "gray"}>
                {me.role}
              </Badge>
            )}
          </Box>
          <MenuDivider />
          <MenuItem as="a" href={`${API_BASE_URL}/api/docs`} target="_blank" rel="noreferrer" icon={<Icon as={FiCode} />}>
            API reference
          </MenuItem>
          <MenuItem as="a" href="https://gohead-cms.github.io/gohead/" target="_blank" rel="noreferrer" icon={<Icon as={FiBookOpen} />}>
            Documentation
          </MenuItem>
          <MenuDivider />
          <MenuItem onClick={handleLogout} icon={<Icon as={FiLogOut} />}>
            Sign out
          </MenuItem>
        </MenuList>
      </Menu>
    </Flex>
  );
}
