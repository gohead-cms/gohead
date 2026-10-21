import { Badge } from "@chakra-ui/react";
import { CUSTOM_ROLE_COLOR, roleColor } from "./settingsFormat";

export function RoleBadge({ role }: { role: string }) {
  return <Badge colorPalette={role ? (roleColor[role] ?? CUSTOM_ROLE_COLOR) : "gray"}>{role || "none"}</Badge>;
}
