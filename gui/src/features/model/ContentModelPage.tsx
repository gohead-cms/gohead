import { Link as RouterLink, useSearchParams } from "react-router-dom";
import { Button, Tab, TabList, TabPanel, TabPanels, Tabs } from "@chakra-ui/react";
import { FiShare2 } from "react-icons/fi";
import { Page, PageHeader } from "../../shared/ui/PageHeader";
import { TypeTable } from "./components/TypeTable";
import { isModelKind, KIND_LABELS, MODEL_KINDS } from "./schema";

/** Collections, components and singletons: list, create, edit, delete. */
export function ContentModelPage() {
  const [params, setParams] = useSearchParams();
  const tab = params.get("tab") ?? undefined;
  const index = isModelKind(tab) ? MODEL_KINDS.indexOf(tab) : 0;

  return (
    <Page>
      <PageHeader
        title="Content model"
        description="The types your content is made of. Changes apply to the API and GraphQL schema right away."
        actions={
          <Button as={RouterLink} to="/workspace" leftIcon={<FiShare2 />} variant="outline" size="sm">
            Schema designer
          </Button>
        }
      />
      <Tabs
        index={index}
        onChange={(i) => setParams(i === 0 ? {} : { tab: MODEL_KINDS[i] }, { replace: true })}
        colorScheme="teal"
        isLazy
      >
        <TabList>
          {MODEL_KINDS.map((k) => (
            <Tab key={k}>{KIND_LABELS[k].plural}</Tab>
          ))}
        </TabList>
        <TabPanels>
          {MODEL_KINDS.map((k) => (
            <TabPanel key={k} px={0}>
              <TypeTable kind={k} />
            </TabPanel>
          ))}
        </TabPanels>
      </Tabs>
    </Page>
  );
}
