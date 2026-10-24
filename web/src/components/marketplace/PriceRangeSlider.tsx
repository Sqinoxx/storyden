import { useState } from "react";

import { Input } from "@/components/ui/input";
import { Slider } from "@/components/ui/slider";
import { useTranslation } from "@/lib/i18n";
import { formatCents } from "@/lib/marketplace/marketplace";
import { HStack, LStack, WStack, styled } from "@/styled-system/jsx";

const STEPS = [1, 2, 5, 10, 20, 50, 100, 200, 500, 1000];
const TARGET_STEPS = 50;

function stepFor(span: number) {
  return (
    STEPS.find((s) => span / s <= TARGET_STEPS) ?? STEPS[STEPS.length - 1]!
  );
}

type Props = {
  boundsMin: number;
  boundsMax: number;
  min: number | null;
  max: number | null;
  onChange: (min: number | null, max: number | null) => void;
};

export function PriceRangeSlider({
  boundsMin,
  boundsMax,
  min,
  max,
  onChange,
}: Props) {
  const t = useTranslation();

  const step = stepFor(boundsMax - boundsMin);
  const lower = Math.floor(boundsMin / step) * step;
  const upper = Math.max(Math.ceil(boundsMax / step) * step, lower + step);

  const clamp = (v: number) => Math.min(Math.max(v, lower), upper);
  const [value, setValue] = useState([
    clamp(min ?? lower),
    clamp(max ?? upper),
  ]);

  const [loText, setLoText] = useState<string | null>(null);
  const [hiText, setHiText] = useState<string | null>(null);

  const [lo = lower, hi = upper] = value;

  const commit = (a = lower, b = upper) => {
    onChange(a <= lower ? null : a, b >= upper ? null : b);
  };

  const commitText = (text: string | null, which: "lo" | "hi") => {
    if (text === null) return;
    const parsed = Number(text);
    if (text.trim() === "" || Number.isNaN(parsed)) {
      setLoText(null);
      setHiText(null);
      return;
    }
    const n = clamp(Math.round(parsed));
    const next = which === "lo" ? [Math.min(n, hi), hi] : [lo, Math.max(n, lo)];
    setLoText(null);
    setHiText(null);
    setValue(next);
    commit(next[0], next[1]);
  };
  const format = (euros: number) => formatCents(euros * 100);
  const label =
    lo <= lower && hi >= upper
      ? t.marketplace.anyPrice
      : hi >= upper
        ? `${format(lo)}+`
        : `${format(lo)} – ${format(hi)}`;

  return (
    <LStack gap="1" justifyContent="center" px="1">
      <WStack fontSize="xs">
        <styled.span color="fg.muted">{t.marketplace.priceLabel}</styled.span>
        <styled.span fontWeight="medium">{label}</styled.span>
      </WStack>
      <Slider
        size="sm"
        min={lower}
        max={upper}
        step={step}
        value={value}
        aria-label={[t.marketplace.priceFrom, t.marketplace.priceTo]}
        onValueChange={(d) => setValue(d.value)}
        onValueChangeEnd={(d) => commit(d.value[0], d.value[1])}
      />
      <HStack gap="2">
        <Input
          size="sm"
          type="number"
          inputMode="numeric"
          min={lower}
          max={upper}
          step={step}
          aria-label={t.marketplace.priceFrom}
          value={loText ?? lo}
          onChange={(e) => setLoText(e.target.value)}
          onBlur={() => commitText(loText, "lo")}
          onKeyDown={(e) => e.key === "Enter" && commitText(loText, "lo")}
        />
        <Input
          size="sm"
          type="number"
          inputMode="numeric"
          min={lower}
          max={upper}
          step={step}
          aria-label={t.marketplace.priceTo}
          value={hiText ?? hi}
          onChange={(e) => setHiText(e.target.value)}
          onBlur={() => commitText(hiText, "hi")}
          onKeyDown={(e) => e.key === "Enter" && commitText(hiText, "hi")}
        />
      </HStack>
    </LStack>
  );
}
