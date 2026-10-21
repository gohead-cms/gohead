import { Badge, Box, Button, Flex, Icon, Text } from "@chakra-ui/react";
import { Avatar } from "@/components/ui/avatar";
import { MenuContent, MenuItem, MenuRoot, MenuSeparator, MenuTrigger } from "@/components/ui/menu";
import { useNavigate } from "react-router-dom";
import { FiBookOpen, FiCode, FiLogOut } from "react-icons/fi";
import { useAuthStore } from "../../services/auth";
import { API_BASE_URL } from "../../services/api";
import { useMe } from "../../features/auth/me";

// As in v2, the avatar gets a color picked from the name.
const AVATAR_PALETTES = ["red", "orange", "green", "teal", "blue", "cyan", "purple", "pink"];
function avatarPalette(name: string) {
  let hash = 0;
  for (const ch of name) hash = (hash * 31 + ch.charCodeAt(0)) | 0;
  return AVATAR_PALETTES[Math.abs(hash) % AVATAR_PALETTES.length];
}

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
      <MenuRoot>
        <MenuTrigger asChild>
          <Button variant="ghost" px={2} py={1} borderRadius="full" aria-label="Account menu">
            <Avatar size="xs" variant="solid" colorPalette={avatarPalette(me?.username ?? "?")} name={me?.username ?? "?"} />
          </Button>
        </MenuTrigger>
        <MenuContent>
          <Box px={4} py={2}>
            <Text fontWeight="bold">{me?.username ?? "…"}</Text>
            {me && (
              <Badge mt={1} colorPalette={me.role === "admin" ? "purple" : "gray"}>
                {me.role}
              </Badge>
            )}
          </Box>
          <MenuSeparator />
          <MenuItem value="api-reference" asChild>
            <a href={`${API_BASE_URL}/api/docs`} target="_blank" rel="noreferrer">
              <Icon as={FiCode} />
              API reference
            </a>
          </MenuItem>
          <MenuItem value="documentation" asChild>
            <a href="https://gohead-cms.github.io/gohead/" target="_blank" rel="noreferrer">
              <Icon as={FiBookOpen} />
              Documentation
            </a>
          </MenuItem>
          <MenuSeparator />
          <MenuItem value="sign-out" onClick={handleLogout}>
            <Icon as={FiLogOut} />
            Sign out
          </MenuItem>
        </MenuContent>
      </MenuRoot>
    </Flex>
  );
}
