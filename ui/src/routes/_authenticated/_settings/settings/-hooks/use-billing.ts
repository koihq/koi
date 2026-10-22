import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";

import { useApi } from "@/api/apiProvider";
import type { BillingChange, BillingStatus } from "@/api/generated";
import { billingCatalogKey, billingSubscriptionKey } from "@/lib/query-keys";

export function billingUpgradeExpired(
  status: BillingStatus | undefined,
  change: BillingChange | undefined,
  now = Date.now(),
) {
  const subscription = status?.subscription;
  const target =
    subscription?.pendingUpgradeQuantity ??
    (change?.effective === "after_payment" ? change.quantity : undefined);
  const expiry = subscription?.pendingUpgradeExpiresAt ?? change?.expiresAt;

  return Boolean(target && subscription?.quantity !== target && expiry && expiry.getTime() <= now);
}

export function billingNeedsConfirmation(
  status: BillingStatus | undefined,
  change: BillingChange | undefined,
  checkoutReturned: boolean,
) {
  if (billingUpgradeExpired(status, change)) return false;
  if (checkoutReturned && !status?.promptLimit) return true;
  if (status?.subscription?.pendingUpgradeQuantity) return true;

  return (
    change?.effective === "after_payment" && status?.subscription?.quantity !== change.quantity
  );
}

export function billingSelection(status: BillingStatus | undefined, firstOption: number) {
  return status?.subscription?.pendingQuantity ?? status?.subscription?.quantity ?? firstOption;
}

export function useBilling(checkoutReturned: boolean, organizationID: string) {
  const api = useApi();
  const queryClient = useQueryClient();
  const request = useRef<{ quantity: number; key: string } | null>(null);
  const [change, setChange] = useState<BillingChange>();
  const [pollStartedAt, setPollStartedAt] = useState(Date.now);
  const [pollExpired, setPollExpired] = useState(false);
  const subscriptionKey = billingSubscriptionKey(organizationID);

  useEffect(() => {
    setPollExpired(false);
    const timer = window.setTimeout(
      () => setPollExpired(true),
      Math.max(0, pollStartedAt + 60_000 - Date.now()),
    );
    return () => window.clearTimeout(timer);
  }, [pollStartedAt]);

  const catalog = useQuery({
    queryKey: billingCatalogKey,
    queryFn: api.billing.catalog,
    staleTime: 60_000,
  });

  const subscription = useQuery({
    queryKey: subscriptionKey,
    queryFn: api.billing.subscription,
    refetchInterval: (query) =>
      !pollExpired && billingNeedsConfirmation(query.state.data, change, checkoutReturned)
        ? 2000
        : false,
    refetchIntervalInBackground: false,
  });

  const update = useMutation({
    mutationFn: (quantity: number) => {
      if (request.current?.quantity !== quantity) {
        request.current = { quantity, key: crypto.randomUUID() };
      }
      return api.billing.changeAllowance(quantity, request.current.key);
    },
    onSuccess: async (result) => {
      request.current = null;
      setChange(result);
      setPollStartedAt(Date.now());
      await queryClient.invalidateQueries({ queryKey: subscriptionKey });
    },
  });

  const checkout = useMutation({
    mutationFn: api.billing.checkout,
    onSuccess: (result) => window.location.assign(result.url),
  });

  const portal = useMutation({
    mutationFn: api.billing.portal,
    onSuccess: (result) => window.location.assign(result.url),
  });

  function refresh() {
    setPollStartedAt(Date.now());
    void subscription.refetch();
  }

  return {
    catalog,
    subscription,
    update,
    checkout,
    portal,
    change,
    refresh,
    awaitingConfirmation: billingNeedsConfirmation(subscription.data, change, checkoutReturned),
    paymentExpired: billingUpgradeExpired(subscription.data, change),
    pollExpired,
  };
}
