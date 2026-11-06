import { createFileRoute, Link, notFound } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import { messageFromUnknown } from "@/api/errors";
import type { BillingSubscription } from "@/api/generated";
import { TickPicker } from "@/components/blocks/tick-picker";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { formatDay } from "@/lib/format";
import {
  billingSelection,
  useBilling,
} from "@/routes/_authenticated/_settings/settings/-hooks/use-billing";

const USAGE_TICKS = Array.from({ length: 30 }, (_, index) => index);

export const Route = createFileRoute("/_authenticated/_settings/settings/billing")({
  beforeLoad: async ({ context }) => {
    if (!context.capabilities.cloudBilling) throw notFound();
  },
  validateSearch: (
    search: Record<string, unknown>,
  ): { checkout?: "success" | "cancel"; sessionId?: string } => ({
    checkout:
      search.checkout === "success" || search.checkout === "cancel" ? search.checkout : undefined,
    sessionId: typeof search.session_id === "string" ? search.session_id : undefined,
  }),
  component: BillingSettings,
});

/** The date that decides what happens next, and the words for it. */
function nextDate(record: BillingSubscription) {
  if (record.cancelAtPeriodEnd && record.currentPeriodEnd) {
    return { key: "accessEnds", date: record.currentPeriodEnd } as const;
  }
  if (record.status === "trialing" && record.trialEnd) {
    return { key: "trialEnds", date: record.trialEnd } as const;
  }
  if (record.status === "active" && record.currentPeriodEnd) {
    return { key: "nextPayment", date: record.currentPeriodEnd } as const;
  }

  return null;
}

function PromptUsage({ used, limit }: { used: number; limit: number }) {
  const { t } = useTranslation("billing");
  const over = used > limit;
  const filled = Math.min(USAGE_TICKS.length, Math.ceil((used / limit) * USAGE_TICKS.length));

  return (
    <div className="mt-7 grid gap-2">
      <p className={`text-body font-semibold tabular-nums ${over ? "text-negative" : ""}`}>
        {t("usage", { used, limit })}
      </p>
      <div aria-hidden="true" className="flex gap-1">
        {USAGE_TICKS.map((tick) => (
          <span
            key={tick}
            className={`h-3.5 w-0.75 rounded-full ${
              tick < filled ? (over ? "bg-negative" : "bg-foreground") : "bg-foreground/12"
            }`}
          />
        ))}
      </div>
      <p className="text-xs text-muted-foreground">{t(over ? "usageOver" : "usageNote")}</p>
    </div>
  );
}

/** Nothing to manage yet, so the page is the offer: what it costs today, then what it costs. */
function ChoosePlan({
  trial,
  today,
  price,
  quantity,
  options,
  pricesFailed,
  pricesLoading,
  disabled,
  error,
  onChange,
  onRetry,
  onSubmit,
}: {
  trial: boolean;
  today: string | null;
  price: string | null;
  quantity: number;
  options: number[];
  pricesFailed: boolean;
  pricesLoading: boolean;
  disabled: boolean;
  error: unknown;
  onChange: (quantity: number) => void;
  onRetry: () => void;
  onSubmit: () => void;
}) {
  const { t, i18n } = useTranslation("billing");

  return (
    <section
      className="mx-auto grid w-full max-w-xl gap-8 rounded-panel bg-surface p-4 sm:px-11 sm:py-9"
      aria-labelledby="billing-heading"
    >
      <div className="grid gap-2">
        {pricesLoading && (
          <output className="block">
            <span className="sr-only">{t("calculating")}</span>
            <Skeleton className="h-[42px] w-44" />
          </output>
        )}
        {!pricesLoading && (
          <h1
            id="billing-heading"
            className={price ? "text-metric font-medium tabular-nums" : "text-title font-semibold"}
          >
            {!price ? (
              t("chooseAllowance")
            ) : trial ? (
              t("trialToday", { price: today })
            ) : (
              <>
                {price}
                <span className="ms-1.5 text-section font-normal tracking-normal text-muted-foreground">
                  {t("perMonthSuffix")}
                </span>
              </>
            )}
          </h1>
        )}
        {price && (
          <p
            className={
              trial ? "text-section font-medium tabular-nums" : "text-body text-muted-foreground"
            }
          >
            {t(trial ? "trialThen" : "description", { price })}
          </p>
        )}
        {pricesFailed && (
          <p role="alert" className="text-sm text-destructive">
            {t("priceUnavailable")}{" "}
            <Button variant="link" onClick={onRetry}>
              {t("retry")}
            </Button>
          </p>
        )}
      </div>

      {options.length > 0 && (
        <TickPicker
          compact
          options={options}
          value={quantity}
          onChange={onChange}
          unit={t("promptsUnit")}
          label={t("trackedPrompts")}
          locale={i18n.language}
          disabled={disabled}
        />
      )}

      <div className="grid gap-3">
        <Button size="lg" disabled={disabled || !price} onClick={onSubmit}>
          {t(trial ? "startTrial" : "checkout")}
        </Button>
        {price && (
          <div className="grid gap-1 text-sm text-muted-foreground">
            {trial && <p>{t("trialIncludes", { quantity })}</p>}
            <p>{t(trial ? "trialTerms" : "rangeDescription")}</p>
          </div>
        )}
        {error != null && (
          <p role="alert" className="text-sm text-destructive">
            {messageFromUnknown(error)}
          </p>
        )}
      </div>
    </section>
  );
}

function BillingSettings() {
  const { t, i18n } = useTranslation("billing");
  const search = Route.useSearch();
  const { me } = Route.useRouteContext();
  const billing = useBilling(search.checkout === "success", me.organization.id, search.sessionId);
  const [selected, setSelected] = useState<number | null>(null);
  const [confirmingStart, setConfirmingStart] = useState(false);
  const { subscription, catalog, update, start, checkout, portal } = billing;
  const record = subscription.data?.subscription;
  const options = catalog.data?.options ?? [];
  const promptLimit = subscription.data?.promptLimit ?? 0;
  const baseQuantity = record?.pendingQuantity ?? record?.quantity;
  const quantity = selected ?? billingSelection(subscription.data, options[0]?.prompts ?? 5);
  const quote = options.find((option) => option.prompts === quantity);
  const current = promptLimit > 0 ? record : null;
  const canCheckout =
    !record || record.status === "canceled" || record.status === "incomplete_expired";
  const canChange = current && !current.cancelAtPeriodEnd;
  const busy = update.isPending || start.isPending || checkout.isPending || portal.isPending;
  const unchanged = baseQuantity === quantity;
  const inTrial = record?.status === "trialing";
  const paymentOpen = Boolean(record?.pendingUpgradeQuantity && !billing.paymentExpired);
  const paymentURL =
    !billing.paymentExpired &&
    (record?.paymentUrl || (billing.awaitingConfirmation ? billing.change?.paymentUrl : undefined));
  const formError = update.error ?? checkout.error;
  const planError = start.error ?? portal.error ?? billing.confirmation.error;
  const upcoming = record ? nextDate(record) : null;

  function money(cents: number | undefined) {
    if (cents === undefined) return null;

    return new Intl.NumberFormat(i18n.language, {
      style: "currency",
      currency: catalog.data?.currency ?? "EUR",
      minimumFractionDigits: 0,
      maximumFractionDigits: 2,
    }).format(cents / 100);
  }

  const price = money(quote?.amountCents);
  const planPrice = money(
    options.find((option) => option.prompts === record?.quantity)?.amountCents,
  );
  const basePrice = money(options.find((option) => option.prompts === baseQuantity)?.amountCents);

  function changeAllowance(target: number) {
    start.reset();
    checkout.reset();
    portal.reset();
    update.mutate(target, { onSuccess: () => setSelected(null) });
  }

  function submit() {
    if (!quote || busy) return;
    if (canChange) changeAllowance(quantity);
    else if (canCheckout) {
      update.reset();
      start.reset();
      portal.reset();
      checkout.mutate(quantity);
    }
  }

  // A returning checkout may not have its subscription yet, and that page says it is confirming.
  const choosing =
    subscription.isSuccess &&
    !record &&
    search.checkout !== "success" &&
    !billing.awaitingConfirmation;

  if (choosing) {
    return (
      <div className="pb-24">
        <ChoosePlan
          trial={subscription.data.trialAvailable}
          today={money(0)}
          price={price}
          quantity={quantity}
          options={options.map((option) => option.prompts)}
          pricesFailed={catalog.isError}
          pricesLoading={catalog.isPending}
          disabled={busy}
          error={formError}
          onChange={setSelected}
          onRetry={() => void catalog.refetch()}
          onSubmit={submit}
        />
      </div>
    );
  }

  return (
    <div className="grid gap-6 pb-24 xl:grid-cols-[minmax(0,1fr)_22rem]">
      <section
        className="min-w-0 rounded-panel bg-surface p-4 sm:px-11 sm:py-9"
        aria-labelledby="billing-heading"
      >
        <div className="flex min-h-8 items-center justify-between gap-3">
          <h1 id="billing-heading" className="text-section font-semibold">
            {t("currentSubscription")}
          </h1>
          {record && (
            <span className="rounded-chip bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
              {t(`status.${record.status}`, { defaultValue: record.status })}
            </span>
          )}
        </div>

        {subscription.isError && (
          <p role="alert" className="mt-4 text-sm text-destructive">
            {messageFromUnknown(subscription.error)}{" "}
            <Button variant="link" onClick={billing.refresh}>
              {t("retry")}
            </Button>
          </p>
        )}

        {subscription.isPending ? (
          <output className="mt-4 block">
            <span className="sr-only">{t("loading")}</span>
            <Skeleton className="h-[42px] w-44" />
            <Skeleton className="mt-2 h-5 w-36" />
          </output>
        ) : record ? (
          <div className="mt-4">
            <p className="text-metric font-medium tabular-nums">
              {planPrice ?? t("planQuantity", { quantity: record.quantity })}
              {planPrice && (
                <span className="ms-1.5 text-section font-normal text-muted-foreground">
                  {t(inTrial ? "perMonthAfterTrial" : "perMonthSuffix")}
                </span>
              )}
            </p>
            <p className="mt-1 text-body text-muted-foreground tabular-nums">
              {t("planQuantity", { quantity: record.quantity })}
              {upcoming && (
                <>
                  {" · "}
                  <span className="text-foreground">
                    {t(upcoming.key, { date: formatDay(upcoming.date, i18n.language) })}
                  </span>
                </>
              )}
            </p>
          </div>
        ) : subscription.isSuccess ? (
          <div className="mt-4 grid gap-1">
            <p className="text-body font-semibold">{t("none")}</p>
            <p className="max-w-[48ch] text-sm text-muted-foreground">{t("description")}</p>
          </div>
        ) : null}

        {current && <PromptUsage used={subscription.data?.usedPrompts ?? 0} limit={promptLimit} />}

        <div className="mt-6 grid max-w-[60ch] gap-2 text-sm empty:hidden">
          {billing.awaitingConfirmation && !(inTrial && paymentOpen) && (
            <output className="block text-muted-foreground">
              {t(billing.pollExpired ? "confirmationDelayed" : "awaitingConfirmation")}
            </output>
          )}
          {inTrial && current && current.quantity !== promptLimit && (
            <p className="text-muted-foreground">
              {t("trialAllowance", { limit: promptLimit, quantity: current.quantity })}
            </p>
          )}
          {record?.pendingQuantity && (
            <p className="text-muted-foreground">
              {t("pending", { quantity: record.pendingQuantity })}
            </p>
          )}
          {paymentOpen && (
            <p className="text-muted-foreground">
              {inTrial
                ? t("startPlanPending")
                : t("pendingUpgrade", { quantity: record?.pendingUpgradeQuantity })}
            </p>
          )}
          {billing.paymentExpired && (
            <output className="block">{t(inTrial ? "startPlanExpired" : "upgradeExpired")}</output>
          )}
          {record?.cancelAtPeriodEnd && (
            <p className="text-muted-foreground">{t("cancellationScheduled")}</p>
          )}
          {record && !current && !canCheckout && (
            <p className="text-muted-foreground">{t("resolvePayment")}</p>
          )}
          {planError && (
            <p role="alert" className="text-destructive">
              {messageFromUnknown(planError)}
            </p>
          )}
        </div>

        <div className="mt-6 flex flex-wrap items-center gap-2 empty:hidden">
          {paymentURL && (
            <Button
              size="lg"
              nativeButton={false}
              role="link"
              render={<a href={paymentURL}>{t("completePayment")}</a>}
            />
          )}
          {search.checkout === "success" && current && (
            <Button size="lg" nativeButton={false} role="link" render={<Link to="/" />}>
              {t("continueToApp")}
            </Button>
          )}
          {billing.awaitingConfirmation && billing.pollExpired && (
            <Button size="lg" variant="outline" onClick={billing.refresh}>
              {t("refresh")}
            </Button>
          )}
          {canChange && inTrial && !paymentOpen && planPrice && (
            <Button size="lg" disabled={busy} onClick={() => setConfirmingStart(true)}>
              {t("startPlan")}
            </Button>
          )}
          {record && (
            <Button
              size="lg"
              variant="outline"
              disabled={busy}
              onClick={() => {
                update.reset();
                start.reset();
                checkout.reset();
                portal.mutate();
              }}
            >
              {t("portal")}
            </Button>
          )}
        </div>
      </section>

      <aside
        className="min-w-0 rounded-panel bg-surface p-4 sm:px-11 sm:py-9 xl:px-5"
        aria-labelledby="allowance-heading"
      >
        <h2 id="allowance-heading" className="flex min-h-8 items-center text-section font-semibold">
          {t(canCheckout ? "chooseAllowance" : "changeAllowance")}
        </h2>

        {catalog.isError && (
          <p role="alert" className="mt-3 text-sm text-destructive">
            {t("priceUnavailable")}{" "}
            <Button variant="link" onClick={() => void catalog.refetch()}>
              {t("retry")}
            </Button>
          </p>
        )}

        <div className="mt-4">
          {options.length ? (
            <TickPicker
              options={options.map((option) => option.prompts)}
              value={quantity}
              onChange={setSelected}
              unit={t("promptsUnit")}
              label={t("trackedPrompts")}
              locale={i18n.language}
              mark={current?.quantity}
              markLabel={t("currentAllowance")}
              disabled={
                subscription.isPending || busy || !(canChange || canCheckout) || paymentOpen
              }
            />
          ) : (
            catalog.isPending && <Skeleton className="h-32 w-full" />
          )}
        </div>

        <p className="mt-4 text-section font-semibold tabular-nums">
          {catalog.isPending
            ? t("calculating")
            : !price
              ? t("priceUnavailable")
              : !unchanged && record && basePrice
                ? t("priceChange", { from: basePrice, to: price })
                : t("perMonth", { price })}
        </p>
        <p className="mt-1 text-xs text-muted-foreground">
          {current?.status === "trialing"
            ? t("trialChangeNotice", { limit: promptLimit })
            : subscription.data?.trialAvailable && !record
              ? t("trialDescription")
              : current
                ? t("taxNote")
                : t("rangeDescription")}
        </p>

        <div className="mt-5 grid gap-2 empty:hidden">
          {(canCheckout || (canChange && !unchanged)) && (
            <Button
              size="lg"
              disabled={
                !quote || busy || subscription.isPending || subscription.isError || paymentOpen
              }
              onClick={submit}
            >
              {canChange ? t("changeAllowance") : t("checkout")}
            </Button>
          )}
          {canChange && record?.pendingQuantity && (
            <Button
              size="lg"
              variant="outline"
              disabled={busy}
              onClick={() => changeAllowance(record.quantity)}
            >
              {t("cancelDowngrade")}
            </Button>
          )}
        </div>
        {billing.change && !billing.awaitingConfirmation && !billing.paymentExpired && (
          <output className="mt-4 block text-sm">
            {t(`effective.${billing.change.effective}`)}
          </output>
        )}
        {formError && (
          <p role="alert" className="mt-4 text-sm text-destructive">
            {messageFromUnknown(formError)}
          </p>
        )}
      </aside>

      <AlertDialog open={confirmingStart} onOpenChange={setConfirmingStart}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("startPlanTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("startPlanDescription", { price: planPrice, quantity: record?.quantity })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("startPlanCancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                setConfirmingStart(false);
                update.reset();
                checkout.reset();
                portal.reset();
                start.mutate();
              }}
            >
              {t("startPlanConfirm", { price: planPrice })}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
