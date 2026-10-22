import { Box, Text } from "@chakra-ui/react";
import { useMe } from "../auth/me";
import { resetLabel, useDemoInfo } from "./demo";

/** Tells visitors of a public demo what they can do. */
export function DemoBanner() {
  const { isDemo } = useMe();
  const info = useDemoInfo();
  if (!isDemo) return null;
  const reset = info ? resetLabel(info) : "";
  return (
    <Box role="status" bg="purple.600" color="white" px={{ base: 4, md: 8 }} py={2} fontSize="sm">
      <Text>
        <strong>GoHead demo.</strong> Look around, write drafts and try agents with <em>Test run</em>. Published content,
        settings and connected apps are read-only, and email addresses are hidden.{reset && ` Content is reset at ${reset}.`}
      </Text>
    </Box>
  );
}
