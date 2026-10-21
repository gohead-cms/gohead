import { Alert, Badge, Button, Grid, GridItem, VStack } from "@chakra-ui/react";
import { FiRefreshCw } from "react-icons/fi";
import { Page, PageHeader } from "../../../shared/ui/PageHeader";
import { PageLoader } from "../../../shared/ui/page-loader";
import { useOverview } from "../hooks/useOverview";
import { AgentActivityCard } from "./AgentActivityCard";
import { DashCard } from "./DashCard";
import { DeliveriesCard } from "./DeliveriesCard";
import { QuickActions } from "./QuickActions";
import { RecentContent } from "./RecentContent";
import { RecentRuns } from "./RecentRuns";
import { StatGrid } from "./StatGrid";

/** Counts and recent activity of the whole instance (GET /admin/overview). */
export function AdminDashboard() {
  const { data, loading, error, refetch } = useOverview();
  const o = data?.overview;

  return (
    <Page>
      <PageHeader
        title="Dashboard"
        description="Overview of your GoHead CMS system"
        actions={
          <>
            {data?.version && (
              <Badge variant="outline" colorPalette="gray" px={2} py={1} borderRadius="md" textTransform="none">
                API {data.version}
              </Badge>
            )}
            <Button size="xs" variant="outline" bg="white" onClick={refetch} loading={loading && !!data}>
              <FiRefreshCw />
              Refresh
            </Button>
          </>
        }
      />
      {error && (
        <Alert.Root role="alert" status="error" borderRadius="md" mb={6} alignItems="center">
          <Alert.Indicator />
          <Alert.Description flex="1">{error}</Alert.Description>
          <Button size="xs" variant="subtle" onClick={refetch}>
            Retry
          </Button>
        </Alert.Root>
      )}
      {!o && loading && <PageLoader text="Loading dashboard..." />}
      {o && (
        <VStack gap={6} align="stretch">
          <StatGrid counts={o.counts} />
          <Grid templateColumns={{ base: "1fr", xl: "minmax(0, 2fr) minmax(0, 1fr)" }} gap={6}>
            <GridItem display="flex" flexDirection="column" gap={6} alignItems="stretch" minW={0}>
              <DashCard title="Recent content changes" link={{ to: "/contrib", label: "Content editor" }}>
                <RecentContent items={o.recentItems ?? []} />
              </DashCard>
              <DashCard title="Recent agent runs" link={{ to: "/activity", label: "All activity" }}>
                <RecentRuns runs={o.recentRuns ?? []} hasAgents={o.counts.agents > 0} />
              </DashCard>
            </GridItem>
            <GridItem display="flex" flexDirection="column" gap={6} alignItems="stretch" minW={0}>
              <DashCard title="Quick actions">
                <QuickActions />
              </DashCard>
              <DashCard title="Agent activity, last 24 h" link={{ to: "/activity", label: "Activity" }}>
                <AgentActivityCard runs={o.runs24h} />
              </DashCard>
              <DashCard title="Webhook deliveries" link={{ to: "/settings/webhooks", label: "Webhooks" }}>
                <DeliveriesCard webhooks={o.counts.webhooks} deliveries={o.deliveries24h} recent={o.recentDeliveries ?? []} />
              </DashCard>
            </GridItem>
          </Grid>
        </VStack>
      )}
    </Page>
  );
}
