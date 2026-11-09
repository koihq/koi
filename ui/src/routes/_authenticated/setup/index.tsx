import { ArrowLeft01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link, redirect, useNavigate, useRouter } from "@tanstack/react-router";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useAnalytics } from "@/analytics/analyticsProvider";
import { useApi } from "@/api/apiProvider";
import type { Brand, CreateBrand } from "@/api/brands.api";
import { ApiError } from "@/api/errors";
import { KoiMark } from "@/components/blocks/koi-mark";
import { SetupProgress } from "@/components/blocks/setup-progress";
import { Button } from "@/components/ui/button";
import { useOnboardingStep } from "@/hooks/use-onboarding-step";
import { brandsKey } from "@/lib/query-keys";
import { type SetupStep, setupProgress } from "@/lib/setup-steps";
import { AgentConnect, AgentPrompt } from "./-components/agent-setup";
import { siteFromEmail } from "./-components/first-brand";
import { FirstBrandForm } from "./-components/first-brand-form";
import { SetupChoice } from "./-components/setup-choice";
import { WebsiteStep } from "./-components/website-step";

type SetupSearch = {
  path?: "koi" | "agent";
  site?: string;
  from?: string;
  stage?: "prompt";
  client?: string;
};

const text = (value: unknown) => (typeof value === "string" && value ? value : undefined);

export const Route = createFileRoute("/_authenticated/setup/")({
  validateSearch: (search: Record<string, unknown>): SetupSearch => ({
    path: search.path === "koi" || search.path === "agent" ? search.path : undefined,
    site: text(search.site),
    from: text(search.from),
    stage: search.stage === "prompt" ? "prompt" : undefined,
    client: text(search.client),
  }),
  beforeLoad: ({ context }) => {
    if (context.brand) {
      throw redirect({ to: "/" });
    }
  },
  component: Setup,
});

function currentStep({ path, site, stage }: SetupSearch, canChoose: boolean): SetupStep {
  if (path === "agent" && canChoose) return stage === "prompt" ? "agent_prompt" : "agent_connect";
  if (path === "koi" || !canChoose) return site ? "confirm" : "website";

  return "choice";
}

function Setup() {
  const { t } = useTranslation("brand");
  const api = useApi();
  const analytics = useAnalytics();
  const router = useRouter();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { capabilities, me } = Route.useRouteContext();
  const search = Route.useSearch();

  // Without a public MCP address an agent cannot connect, so there is nothing to choose.
  const mcpUrl = capabilities.mcpUrl;
  const canChoose = Boolean(mcpUrl);
  const step = currentStep(search, canChoose);
  const progress = setupProgress(step, { canChoose, cloud: capabilities.cloudBilling });

  useOnboardingStep(step);

  // Setup is where people stop, so it is the one place recorded, with consent and inputs masked.
  useEffect(() => {
    analytics.startReplay();

    return () => analytics.stopReplay();
  }, [analytics]);

  async function createBrand(body: CreateBrand) {
    let brand: Brand | undefined;

    try {
      brand = await api.brands.create(body);
      await queryClient.invalidateQueries({ queryKey: brandsKey(brand.organizationId) });
    } catch (error) {
      if (!(error instanceof ApiError) || error.code !== "brand_limit_reached") {
        throw error;
      }
    }

    let draftStart: "failed" | "unavailable" | undefined;
    if (brand && capabilities.aiDrafting) {
      try {
        await api.generations.create(
          { brandId: brand.id, startTracking: capabilities.cloudBilling },
          `setup-${brand.id}`,
        );
      } catch (error) {
        draftStart =
          error instanceof ApiError && error.code === "first_look_unavailable"
            ? "unavailable"
            : "failed";
      }
    }

    await router.invalidate();
    await router.navigate({
      to: "/",
      search: draftStart ? { draftStart } : {},
    });
  }

  const headings: Record<SetupStep, { title: string; description: string }> = {
    choice: { title: t("setup.title"), description: t("setup.description") },
    website: { title: t("setup.websiteTitle"), description: t("setup.websiteDescription") },
    confirm: { title: t("setup.confirmTitle"), description: t("setup.formDescription") },
    agent_connect: { title: t("setup.agentTitle"), description: t("setup.agentDescription") },
    agent_prompt: { title: t("setup.promptTitle"), description: t("setup.promptDescription") },
  };
  const heading = headings[step];

  const back: SetupSearch | null =
    step === "confirm"
      ? { path: "koi", from: search.site }
      : step === "agent_prompt"
        ? { path: "agent", client: search.client }
        : step === "website" || step === "agent_connect"
          ? canChoose
            ? {}
            : null
          : null;

  return (
    <main className="flex min-h-svh flex-col items-center justify-center px-6 py-12">
      <div className={step === "choice" ? "w-full max-w-2xl" : "w-full max-w-md"}>
        <div className="mb-8 flex items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            {back && (
              <Button
                variant="ghost"
                size="circle-sm"
                role="link"
                aria-label={t("setup.back")}
                className="-ms-2 text-muted-foreground"
                render={<Link to="/setup" search={back} />}
                nativeButton={false}
              >
                <HugeiconsIcon icon={ArrowLeft01Icon} aria-hidden="true" />
              </Button>
            )}
            <KoiMark />
          </div>
          <SetupProgress current={progress.current} total={progress.total} />
        </div>
        <h1 className="text-2xl font-semibold tracking-tight">{heading.title}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{heading.description}</p>
        <div className="mt-8">
          {step === "choice" && <SetupChoice />}
          {step === "website" && (
            <WebsiteStep
              initial={search.from ?? siteFromEmail(me.email)}
              onSubmit={(site) => void navigate({ to: "/setup", search: { path: "koi", site } })}
            />
          )}
          {step === "confirm" && search.site && (
            <FirstBrandForm key={search.site} site={search.site} onSubmit={createBrand} />
          )}
          {step === "agent_connect" && mcpUrl && (
            <AgentConnect mcpUrl={mcpUrl} clientName={search.client} />
          )}
          {step === "agent_prompt" && <AgentPrompt clientName={search.client} />}
        </div>
      </div>
    </main>
  );
}
