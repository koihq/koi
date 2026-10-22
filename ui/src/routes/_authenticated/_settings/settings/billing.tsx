import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link, notFound } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import { useApi } from "@/api/apiProvider";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const allowances = [5, 10, 15, 25, 50, 75, 100, 150, 200, 250, 350, 500];
const allowanceItems = allowances.map((quantity) => ({
  value: String(quantity),
  label: String(quantity),
}));

export const Route = createFileRoute("/_authenticated/_settings/settings/billing")({
  beforeLoad: async ({ context }) => {
    if (!(await context.api.capabilities.get()).cloudBilling) throw notFound();
  },
  component: BillingSettings,
});

function BillingSettings() {
  const { t } = useTranslation("billing");
  const api = useApi();
  const [prompts, setPrompts] = useState("5");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const quantity = Number(prompts);
  const valid = allowances.includes(quantity);
  const checkoutReturned =
    new URLSearchParams(window.location.search).get("checkout") === "success";
  const subscription = useQuery({
    queryKey: ["billing", "subscription"],
    queryFn: api.billing.subscription,
    refetchInterval: (query) =>
      checkoutReturned && (query.state.data?.promptLimit ?? 0) === 0 ? 2000 : false,
  });
  const quote = useQuery({
    queryKey: ["billing", "quote", quantity],
    queryFn: () => api.billing.quote(quantity),
    enabled: valid,
  });
  const subscriptionRecord = subscription.data?.subscription;
  const current =
    (subscription.data?.promptLimit ?? 0) > 0 &&
    (subscriptionRecord?.status === "active" || subscriptionRecord?.status === "trialing")
      ? subscriptionRecord
      : null;
  const canCheckout = !subscriptionRecord || subscriptionRecord.status === "canceled";

  async function submit() {
    if (!valid || !quote.data || (!current && !canCheckout)) return;
    setBusy(true);
    setError("");
    setMessage("");
    try {
      if (current) {
        const result = await api.billing.changeAllowance(quantity);
        setMessage(result.effective === "next_cycle" ? t("nextCycle") : t("afterPayment"));
        await subscription.refetch();
      } else {
        const result = await api.billing.checkout(quantity);
        window.location.assign(result.url);
      }
    } catch {
      setError(t("updateError"));
    } finally {
      setBusy(false);
    }
  }

  async function openPortal() {
    setBusy(true);
    setError("");
    try {
      const result = await api.billing.portal();
      window.location.assign(result.url);
    } catch {
      setError(t("portalError"));
      setBusy(false);
    }
  }

  const price = quote.data
    ? new Intl.NumberFormat("en", {
        style: "currency",
        currency: "EUR",
        maximumFractionDigits: 0,
      }).format(quote.data.amountCents / 100)
    : null;

  return (
    <div className="max-w-2xl pb-24">
      <h1 className="text-[30px] font-semibold tracking-[-0.035em]">{t("title")}</h1>
      <p className="mt-2 text-sm text-muted-foreground">{t("description")}</p>
      {checkoutReturned && (subscription.data?.promptLimit ?? 0) === 0 && (
        <output className="mt-4 block text-sm text-muted-foreground">
          {t("awaitingConfirmation")}
        </output>
      )}
      {checkoutReturned && (subscription.data?.promptLimit ?? 0) > 0 && (
        <Button render={<Link to="/" />} className="mt-4">
          {t("continueToApp")}
        </Button>
      )}

      <div className="mt-10 border-t border-border pt-7">
        {subscription.isError && (
          <p role="alert" className="mb-4 text-sm text-destructive">
            {t("updateError")}{" "}
            <Button variant="link" onClick={() => void subscription.refetch()}>
              {t("retry")}
            </Button>
          </p>
        )}
        <p className="text-xs uppercase tracking-widest text-muted-foreground">
          {t("currentSubscription")}
        </p>
        <p className="mt-3 text-lg font-medium">
          {subscription.isLoading
            ? t("loading")
            : subscriptionRecord
              ? t("current", {
                  quantity: subscriptionRecord.quantity,
                  status: subscriptionRecord.status,
                })
              : t("none")}
        </p>
        {current?.pendingQuantity && (
          <p className="mt-1 text-sm text-muted-foreground">
            {t("pending", { quantity: current.pendingQuantity })}
          </p>
        )}
        {current && (
          <p className="mt-1 text-sm text-muted-foreground">
            {t("allowance", { quantity: subscription.data?.promptLimit ?? 0 })}
          </p>
        )}
        {subscriptionRecord && !current && subscriptionRecord.status !== "canceled" && (
          <p className="mt-3 text-sm text-muted-foreground">{t("resolvePayment")}</p>
        )}
      </div>

      <div className="mt-10 border-t border-border pt-7">
        <Label htmlFor="billing-prompts">{t("trackedPrompts")}</Label>
        <p className="mt-1 text-sm text-muted-foreground">{t("rangeDescription")}</p>
        {subscription.data?.trialAvailable && !subscriptionRecord && (
          <p className="mt-2 text-sm text-muted-foreground">{t("trialDescription")}</p>
        )}
        <Select
          value={prompts}
          items={allowanceItems}
          onValueChange={(next) => {
            if (next !== null) setPrompts(next);
          }}
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
          {valid
            ? quote.isLoading
              ? t("calculating")
              : price
                ? t("perMonth", { price })
                : t("priceUnavailable")
            : t("invalidQuantity")}
        </p>
        <p className="mt-1 text-xs text-muted-foreground">{t("taxNote")}</p>
        <div className="mt-6 flex flex-wrap gap-2">
          {(current || canCheckout) && (
            <Button
              size="lg"
              disabled={
                !valid || !quote.data || busy || subscription.isLoading || subscription.isError
              }
              onClick={() => void submit()}
            >
              {current ? t("changeAllowance") : t("checkout")}
            </Button>
          )}
          {subscriptionRecord && (
            <Button size="lg" variant="outline" disabled={busy} onClick={() => void openPortal()}>
              {t("portal")}
            </Button>
          )}
        </div>
        {message && <output className="mt-4 block text-sm">{message}</output>}
        {error && (
          <p role="alert" className="mt-4 text-sm text-destructive">
            {error}
          </p>
        )}
      </div>
    </div>
  );
}
