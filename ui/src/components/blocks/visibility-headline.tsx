import NumberFlow from "@number-flow/react";
import { useTranslation } from "react-i18next";
import { BrandIcon } from "@/components/blocks/brand-icon";

type ChartBrand = { id: string; name: string; domain?: string };

function RateValue({
  value,
  locale,
  animated,
}: {
  value: number | null;
  locale: string;
  animated: boolean;
}) {
  const { t } = useTranslation("home");

  return (
    <>
      {value === null ? (
        t("dashboard.noData")
      ) : (
        <NumberFlow
          value={Number(value.toFixed(1))}
          locales={locale}
          format={{ minimumFractionDigits: 1, maximumFractionDigits: 1 }}
          animated={animated}
          className="[--number-flow-mask-height:0.3em]"
        />
      )}
      {value !== null && (
        <span className="ml-0.5 text-[0.667em] text-muted-foreground">
          {t("dashboard.percentSign")}
        </span>
      )}
    </>
  );
}

export function VisibilityHeadline({
  brand,
  competitor,
  rate,
  comparisonRate,
  paired,
  locale,
  animated,
  variant,
}: {
  brand: ChartBrand;
  competitor?: ChartBrand;
  rate: number | null;
  comparisonRate: number | null;
  paired: boolean;
  locale: string;
  animated: boolean;
  variant: "home" | "detail";
}) {
  if (paired && competitor) {
    return (
      <div className={`${variant === "detail" ? "mt-4 " : ""}flex flex-wrap items-center gap-5`}>
        {[
          { subject: brand, value: rate },
          { subject: competitor, value: comparisonRate },
        ].map(({ subject, value }) => (
          <span key={subject.id} className="inline-flex items-center gap-2 whitespace-nowrap">
            <BrandIcon name={subject.name} domain={subject.domain} className="size-6" />
            <strong className="text-[36px] leading-[1.1] font-medium tracking-[-0.025em] tabular-nums">
              <RateValue value={value} locale={locale} animated={animated} />
            </strong>
          </span>
        ))}
      </div>
    );
  }

  return (
    <strong
      className={`${variant === "detail" ? "mt-4 block " : "whitespace-nowrap "}text-[36px] leading-[1.1] font-medium tracking-[-0.025em] tabular-nums`}
    >
      <RateValue value={rate} locale={locale} animated={animated} />
    </strong>
  );
}
