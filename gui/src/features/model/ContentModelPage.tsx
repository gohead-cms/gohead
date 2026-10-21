import { Link as RouterLink, useSearchParams } from "react-router-dom";
import { Button, Tabs } from "@chakra-ui/react";
import { FiShare2 } from "react-icons/fi";
import { Page, PageHeader } from "../../shared/ui/PageHeader";
import { TypeTable } from "./components/TypeTable";
import { isModelKind, KIND_LABELS, MODEL_KINDS } from "./schema";

/** Collections, components and singletons: list, create, edit, delete. */
export function ContentModelPage() {
  const [params, setParams] = useSearchParams();
  const tab = params.get("tab") ?? undefined;
  const value = isModelKind(tab) ? tab : MODEL_KINDS[0];

  return (
    <Page>
      <PageHeader
        title="Content model"
        description="The types your content is made of. Changes apply to the API and GraphQL schema right away."
        actions={
          <Button asChild variant="outline" size="xs">
            <RouterLink to="/workspace">
              <FiShare2 />
              Schema designer
            </RouterLink>
          </Button>
        }
      />
      <Tabs.Root
        value={value}
        onValueChange={(e) => setParams(e.value === MODEL_KINDS[0] ? {} : { tab: e.value }, { replace: true })}
        colorPalette="teal"
        lazyMount
        unmountOnExit
      >
        <Tabs.List>
          {MODEL_KINDS.map((k) => (
            <Tabs.Trigger key={k} value={k}>
              {KIND_LABELS[k].plural}
            </Tabs.Trigger>
          ))}
        </Tabs.List>
        {MODEL_KINDS.map((k) => (
          <Tabs.Content key={k} value={k} px={0}>
            <TypeTable kind={k} />
          </Tabs.Content>
        ))}
      </Tabs.Root>
    </Page>
  );
}
