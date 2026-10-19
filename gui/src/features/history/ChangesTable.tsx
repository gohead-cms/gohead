import { Badge, Box, Code, HStack, Icon, Table, TableContainer, Tbody, Td, Text, Th, Thead, Tr, useColorModeValue } from "@chakra-ui/react";
import { FiGlobe } from "react-icons/fi";
import { type FieldChange, REDACTED, splitField } from "./history";
import { type DiffPart, isLongText, wordDiff } from "./wordDiff";

const changeColor: Record<FieldChange["change"], string> = {
  added: "green",
  removed: "red",
  changed: "blue",
};

/** A value as stored: objects and arrays as pretty JSON, nothing as a dash. */
function Value({ value }: { value: unknown }) {
  const codeBg = useColorModeValue("gray.50", "gray.800");
  if (value === undefined || value === null || value === "") {
    return (
      <Text as="span" color="gray.400" fontStyle="italic">
        empty
      </Text>
    );
  }
  if (value === REDACTED) {
    // Password attributes are redacted by the API.
    return (
      <Text as="span" color="gray.500" fontStyle="italic" title="Passwords are not shown">
        (hidden)
      </Text>
    );
  }
  if (typeof value === "object") {
    return (
      <Code display="block" whiteSpace="pre-wrap" wordBreak="break-word" p={2} fontSize="xs" bg={codeBg} maxH="240px" overflowY="auto">
        {JSON.stringify(value, null, 2)}
      </Code>
    );
  }
  return (
    <Text as="span" whiteSpace="pre-wrap" wordBreak="break-word">
      {String(value)}
    </Text>
  );
}

/** Two long texts as one, with removed words struck out and added words highlighted. */
function InlineDiff({ parts }: { parts: DiffPart[] }) {
  const removedBg = useColorModeValue("red.100", "red.900");
  const addedBg = useColorModeValue("green.100", "green.900");
  return (
    <Text whiteSpace="pre-wrap" wordBreak="break-word" fontSize="sm" data-testid="inline-diff">
      {parts.map((p, i) =>
        p.kind === "same" ? (
          <span key={i}>{p.text}</span>
        ) : p.kind === "removed" ? (
          <Box as="del" key={i} bg={removedBg} textDecoration="line-through" px="1px" mr="2px" borderRadius="sm" aria-label="removed">
            {p.text}
          </Box>
        ) : (
          <Box as="ins" key={i} bg={addedBg} textDecoration="none" px="1px" borderRadius="sm" aria-label="added">
            {p.text}
          </Box>
        ),
      )}
    </Text>
  );
}

/** The fields that differ between two versions, before and after. */
export function ChangesTable({ changes, beforeLabel, afterLabel }: { changes: FieldChange[]; beforeLabel: string; afterLabel: string }) {
  const headBg = useColorModeValue("gray.50", "gray.700");
  return (
    <TableContainer borderWidth="1px" borderRadius="md" whiteSpace="normal">
      <Table size="sm" sx={{ tableLayout: "fixed", "th, td": { px: 3, verticalAlign: "top" } }}>
        <Thead bg={headBg}>
          <Tr>
            <Th w="26%">Field</Th>
            <Th>{beforeLabel}</Th>
            <Th>{afterLabel}</Th>
          </Tr>
        </Thead>
        <Tbody>
          {changes.map((c) => {
            const { name, locale } = splitField(c.field);
            // Long texts on both sides read better as one inline diff.
            const parts = c.change === "changed" && isLongText(c.before) && isLongText(c.after) ? wordDiff(c.before, c.after) : null;
            return (
              <Tr key={c.field} data-testid={`change-${c.field}`}>
                <Td>
                  <HStack spacing={1.5} align="center" wrap="wrap">
                    <Text fontWeight="medium" wordBreak="break-word">
                      {name}
                    </Text>
                    {locale && (
                      <Badge
                        colorScheme="purple"
                        variant="outline"
                        fontSize="2xs"
                        textTransform="none"
                        display="inline-flex"
                        alignItems="center"
                        gap={0.5}
                        title={`The ${locale} translation`}
                      >
                        <Icon as={FiGlobe} boxSize="0.8em" />
                        {locale}
                      </Badge>
                    )}
                  </HStack>
                  <Badge mt={1} colorScheme={changeColor[c.change]} fontSize="2xs">
                    {c.change}
                  </Badge>
                </Td>
                {parts ? (
                  <Td colSpan={2}>
                    <InlineDiff parts={parts} />
                  </Td>
                ) : (
                  <>
                    <Td>
                      <Value value={c.before} />
                    </Td>
                    <Td>
                      <Value value={c.after} />
                    </Td>
                  </>
                )}
              </Tr>
            );
          })}
        </Tbody>
      </Table>
    </TableContainer>
  );
}
