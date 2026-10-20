import { useQuery } from "@tanstack/react-query";
import { createFileRoute, notFound } from "@tanstack/react-router";
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
  const subscription = useQuery({
    queryKey: ["billing", "subscription"],
    queryFn: api.billing.subscription,
  });
  const quote = useQuery({
    queryKey: ["billing", "quote", quantity],
    queryFn: () => api.billing.quote(quantity),
    enabled: valid,
  });
  const subscriptionRecord = subscription.data?.subscription;
  const current = subscriptionRecord?.status === "canceled" ? null : subscriptionRecord;

  async function submit() {
    if (!valid || !quote.data) return;
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

      <div className="mt-10 border-t border-border pt-7">
        <p className="text-xs uppercase tracking-widest text-muted-foreground">
          {t("currentSubscription")}
        </p>
        <p className="mt-3 text-lg font-medium">
          {subscription.isLoading
            ? t("loading")
            : current
              ? t("current", { quantity: current.quantity, status: current.status })
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
      </div>

      <div className="mt-10 border-t border-border pt-7">
        <Label htmlFor="billing-prompts">{t("trackedPrompts")}</Label>
        <p className="mt-1 text-sm text-muted-foreground">{t("rangeDescription")}</p>
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
          <Button
            size="lg"
            disabled={!valid || !quote.data || busy || subscription.isLoading}
            onClick={() => void submit()}
          >
            {current ? t("changeAllowance") : t("checkout")}
          </Button>
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
