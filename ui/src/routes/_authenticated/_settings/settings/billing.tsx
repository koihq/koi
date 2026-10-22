import { createFileRoute, Link, notFound } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import { messageFromUnknown } from "@/api/errors";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  billingSelection,
  useBilling,
} from "@/routes/_authenticated/_settings/settings/-hooks/use-billing";

export const Route = createFileRoute("/_authenticated/_settings/settings/billing")({
  beforeLoad: async ({ context }) => {
    if (!(await context.api.capabilities.get()).cloudBilling) throw notFound();
  },
  validateSearch: (search: Record<string, unknown>): { checkout?: "success" | "cancel" } => ({
    checkout:
      search.checkout === "success" || search.checkout === "cancel" ? search.checkout : undefined,
  }),
  component: BillingSettings,
});

function BillingSettings() {
  const { t, i18n } = useTranslation("billing");
  const search = Route.useSearch();
  const { me } = Route.useRouteContext();
  const billing = useBilling(search.checkout === "success", me.organization.id);
  const [selected, setSelected] = useState<string | null>(null);
  const { subscription, catalog, update, checkout, portal } = billing;
  const record = subscription.data?.subscription;
  const options = catalog.data?.options ?? [];
  const quantity =
    selected === null
      ? billingSelection(subscription.data, options[0]?.prompts ?? 5)
      : Number(selected);
  const allowanceItems = options.map((option) => ({
    value: String(option.prompts),
    label: String(option.prompts),
  }));
  const quote = options.find((option) => option.prompts === quantity);
  const current = (subscription.data?.promptLimit ?? 0) > 0 ? record : null;
  const canCheckout =
    !record || record.status === "canceled" || record.status === "incomplete_expired";
  const canChange = current && !current.cancelAtPeriodEnd;
  const busy = update.isPending || checkout.isPending || portal.isPending;
  const unchanged = (record?.pendingQuantity ?? record?.quantity) === quantity;
  const paymentURL =
    !billing.paymentExpired &&
    (record?.paymentUrl || (billing.awaitingConfirmation ? billing.change?.paymentUrl : undefined));
  const error = update.error ?? checkout.error ?? portal.error;
  const price = quote
    ? new Intl.NumberFormat(i18n.language, {
        style: "currency",
        currency: catalog.data?.currency ?? "EUR",
        minimumFractionDigits: 0,
        maximumFractionDigits: 2,
      }).format(quote.amountCents / 100)
    : null;

  function changeAllowance(target: number) {
    checkout.reset();
    portal.reset();
    update.mutate(target, { onSuccess: () => setSelected(null) });
  }

  function submit() {
    if (!quote || busy) return;
    if (canChange) changeAllowance(quantity);
    else if (canCheckout) {
      update.reset();
      portal.reset();
      checkout.mutate(quantity);
    }
  }

  return (
    <div className="max-w-2xl pb-24">
      <h1 className="text-[30px] font-semibold tracking-[-0.035em]">{t("title")}</h1>
      <p className="mt-2 text-sm text-muted-foreground">{t("description")}</p>
      {billing.awaitingConfirmation && (
        <div className="mt-4 space-y-2 text-sm">
          <output className="block text-muted-foreground">
            {t(billing.pollExpired ? "confirmationDelayed" : "awaitingConfirmation")}
          </output>
          {billing.pollExpired && (
            <Button variant="outline" onClick={billing.refresh}>
              {t("refresh")}
            </Button>
          )}
        </div>
      )}
      {search.checkout === "success" && current && (
        <Button nativeButton={false} role="link" render={<Link to="/" />} className="mt-4">
          {t("continueToApp")}
        </Button>
      )}

      <div className="mt-10 space-y-3">
        {subscription.isError && (
          <p role="alert" className="text-sm text-destructive">
            {messageFromUnknown(subscription.error)}{" "}
            <Button variant="link" onClick={billing.refresh}>
              {t("retry")}
            </Button>
          </p>
        )}
        <h2 className="text-lg font-medium tracking-tight">{t("currentSubscription")}</h2>
        <p>
          {subscription.isPending
            ? t("loading")
            : record
              ? t("current", {
                  quantity: record.quantity,
                  status: t(`status.${record.status}`, { defaultValue: record.status }),
                })
              : t("none")}
        </p>
        {record?.pendingQuantity && (
          <p className="text-sm text-muted-foreground">
            {t("pending", { quantity: record.pendingQuantity })}
          </p>
        )}
        {record?.pendingUpgradeQuantity && !billing.paymentExpired && (
          <p className="text-sm text-muted-foreground">
            {t("pendingUpgrade", { quantity: record.pendingUpgradeQuantity })}
          </p>
        )}
        {billing.paymentExpired && <output className="block text-sm">{t("upgradeExpired")}</output>}
        {current && (
          <p className="text-sm text-muted-foreground">
            {t("allowance", { quantity: subscription.data?.promptLimit ?? 0 })}
          </p>
        )}
        {record?.cancelAtPeriodEnd && (
          <p className="text-sm text-muted-foreground">{t("cancellationScheduled")}</p>
        )}
        {record && !current && !canCheckout && (
          <p className="text-sm text-muted-foreground">{t("resolvePayment")}</p>
        )}
        {paymentURL && (
          <Button
            nativeButton={false}
            role="link"
            render={<a href={paymentURL}>{t("completePayment")}</a>}
          />
        )}
      </div>

      <div className="mt-10">
        <Label htmlFor="billing-prompts">{t("trackedPrompts")}</Label>
        <p className="mt-1 text-sm text-muted-foreground">{t("rangeDescription")}</p>
        {subscription.data?.trialAvailable && !record && (
          <p className="mt-2 text-sm text-muted-foreground">{t("trialDescription")}</p>
        )}
        {catalog.isError && (
          <p role="alert" className="mt-3 text-sm text-destructive">
            {t("priceUnavailable")}{" "}
            <Button variant="link" onClick={() => void catalog.refetch()}>
              {t("retry")}
            </Button>
          </p>
        )}
        <Select
          value={String(quantity)}
          items={allowanceItems}
          onValueChange={setSelected}
          disabled={catalog.isPending || subscription.isPending || busy}
        >
          <SelectTrigger id="billing-prompts" className="mt-4 w-28">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {allowanceItems.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                {item.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="mt-5 text-2xl font-semibold tabular-nums">
          {catalog.isPending
            ? t("calculating")
            : price
              ? t("perMonth", { price })
              : t("priceUnavailable")}
        </p>
        <p className="mt-1 text-xs text-muted-foreground">
          {current?.status === "trialing" ? t("trialChangeNotice") : t("taxNote")}
        </p>
        <div className="mt-6 flex flex-wrap gap-2">
          {(canChange || canCheckout) && (
            <Button
              size="lg"
              disabled={
                !quote ||
                busy ||
                subscription.isPending ||
                subscription.isError ||
                Boolean(canChange && unchanged) ||
                Boolean(record?.pendingUpgradeQuantity && !billing.paymentExpired)
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
          {record && (
            <Button
              size="lg"
              variant="outline"
              disabled={busy}
              onClick={() => {
                update.reset();
                checkout.reset();
                portal.mutate();
              }}
            >
              {t("portal")}
            </Button>
          )}
        </div>
        {billing.change && !billing.awaitingConfirmation && !billing.paymentExpired && (
          <output className="mt-4 block text-sm">
            {t(`effective.${billing.change.effective}`)}
          </output>
        )}
        {error && (
          <p role="alert" className="mt-4 text-sm text-destructive">
            {messageFromUnknown(error)}
          </p>
        )}
      </div>
    </div>
  );
}
