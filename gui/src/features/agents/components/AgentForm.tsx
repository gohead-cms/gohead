import { type FormEvent, useMemo, useState } from "react";
import { Link as RouterLink, useNavigate } from "react-router-dom";
import { Alert, AlertDescription, AlertIcon, AlertTitle, Box, Button, HStack, Text, VStack } from "@chakra-ui/react";
import { api } from "../../../services/api";
import { useNotify } from "../../../shared/hooks/useNotify";
import { errorMessage } from "../../../shared/utils/errors";
import { type AgentFunctionInfo } from "../../../shared/types";
import { type CollectionChoice } from "../hook/useEditorData";
import { type AgentDraft, buildPayload, validateDraft } from "./agentForm";
import { BasicsSection, LimitsSection, MemorySection, ModelSection, TriggerSection } from "./AgentFormSections";
import { FunctionsEditor } from "./FunctionsEditor";

interface AgentFormProps {
  initial: AgentDraft;
  /** The agent's current name when editing; null when creating. */
  originalName: string | null;
  agentId?: number;
  takenNames: string[];
  catalog: AgentFunctionInfo[];
  catalogError: string | null;
  collections: CollectionChoice[];
  collectionsError: string | null;
}

export function AgentForm({ initial, originalName, agentId, takenNames, catalog, catalogError, collections, collectionsError }: AgentFormProps) {
  const [draft, setDraft] = useState(initial);
  const [attempted, setAttempted] = useState(false);
  const [saving, setSaving] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);
  const navigate = useNavigate();
  const notify = useNotify();

  const allErrors = useMemo(() => validateDraft(draft, takenNames), [draft, takenNames]);
  const errors = attempted ? allErrors : {};
  const errorCount = Object.keys(allErrors).length;
  const set = <K extends keyof AgentDraft>(key: K, value: AgentDraft[K]) => setDraft((d) => ({ ...d, [key]: value }));
  const cancelTo = originalName ? `/agents?name=${encodeURIComponent(originalName)}` : "/agents";

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setAttempted(true);
    setServerError(null);
    if (errorCount > 0) return;
    setSaving(true);
    try {
      const body = buildPayload(draft);
      if (originalName) await api("PUT", `/admin/agents/${encodeURIComponent(originalName)}`, body);
      else await api("POST", "/admin/agents", body);
      notify.success(originalName ? "Agent saved" : "Agent created");
      navigate(`/agents?name=${encodeURIComponent(draft.name.trim())}`);
    } catch (err) {
      setServerError(errorMessage(err));
      setSaving(false);
    }
  };

  const sectionProps = { draft, set, errors };
  return (
    <Box as="form" onSubmit={submit} noValidate>
      <VStack spacing={5} align="stretch">
        <BasicsSection {...sectionProps} />
        <ModelSection {...sectionProps} />
        <TriggerSection {...sectionProps} collections={collections} collectionsError={collectionsError} agentId={agentId} />
        <FunctionsEditor
          functions={draft.functions}
          onChange={(fns) => set("functions", fns)}
          catalog={catalog}
          catalogError={catalogError}
          collections={collections}
          errors={errors}
        />
        <LimitsSection {...sectionProps} />
        <MemorySection {...sectionProps} />

        {serverError && (
          <Alert status="error" borderRadius="md">
            <AlertIcon />
            <Box>
              <AlertTitle>The agent was not saved</AlertTitle>
              <AlertDescription>{serverError}</AlertDescription>
            </Box>
          </Alert>
        )}
        <HStack
          justify="flex-end"
          spacing={3}
          position="sticky"
          bottom={0}
          bg="gray.100"
          py={3}
          borderTopWidth="1px"
          zIndex={1}
        >
          {attempted && errorCount > 0 && (
            <Text fontSize="sm" color="red.600" mr="auto">
              Fix {errorCount} field{errorCount === 1 ? "" : "s"} above before saving.
            </Text>
          )}
          <Button as={RouterLink} to={cancelTo} variant="ghost" isDisabled={saving}>
            Cancel
          </Button>
          <Button type="submit" colorScheme="teal" isLoading={saving} loadingText="Saving">
            {originalName ? "Save changes" : "Create agent"}
          </Button>
        </HStack>
      </VStack>
    </Box>
  );
}
