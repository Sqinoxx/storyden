"use client";

import { SliderValueChangeDetails } from "@ark-ui/react";

import {
  LightBgStyle,
  LightPreset,
  PRESET_CONFIGS,
  useTheme,
} from "@/lib/theme/ThemeProvider";

import { Unready } from "@/components/site/Unready";
import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { RadioGroupField } from "@/components/ui/form/RadioGroupField";
import { Heading } from "@/components/ui/heading";
import { Slider } from "@/components/ui/slider";
import { css } from "@/styled-system/css";
import { Box, CardBox, Grid, HStack, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import {
  Props,
  useMemberInterfaceSettings,
} from "./useMemberInterfaceSettings";

function warmthLabel(value: number): string {
  if (value === 0) return "Neutral";
  if (value <= 20) return "Leicht warm";
  if (value <= 50) return "Warm";
  if (value <= 75) return "Sehr warm";
  return "Maximal warm";
}

const PRESET_LIST: Array<{
  id: LightPreset;
  title: string;
  subtitle: string;
  previewBg: string;
  previewBorder: string;
  previewDot: string;
}> = [
    {
      id: "neutral",
      title: "Neutral",
      subtitle: "Notion / GitHub",
      previewBg: "#ffffff",
      previewBorder: "#e4e4e7",
      previewDot: "#27272a",
    },
    {
      id: "paper",
      title: "Warmes Papier",
      subtitle: "Substack / Medium",
      previewBg: "#fbf9f5",
      previewBorder: "#e8ded0",
      previewDot: "#d97706",
    },
    {
      id: "nordic",
      title: "Kühles Nordisch",
      subtitle: "Linear / Arc",
      previewBg: "#f5f7fa",
      previewBorder: "#d0e2ec",
      previewDot: "#2563eb",
    },
  ];

const BG_STYLE_OPTIONS: Array<{
  id: LightBgStyle;
  label: string;
  desc: string;
}> = [
    { id: "default", label: "Reinweiß", desc: "Standard hell" },
    { id: "soft", label: "Augenschonend", desc: "Sanftes Grau-Weiß" },
    { id: "warm", label: "Warmes Creme", desc: "Papier-Tönung" },
  ];

export function MemberInterfaceSettings(props: Props) {
  const result = useMemberInterfaceSettings(props);
  const {
    resolvedTheme,
    warmth,
    setWarmth,
    lightPreset,
    setLightPreset,
    lightBgStyle,
    setLightBgStyle,
  } = useTheme();

  if (!result.ready) {
    return <Unready />;
  }

  const { control, formState, onSubmit } = result;

  function handleWarmthChange(details: SliderValueChangeDetails) {
    setWarmth(details.value[0] ?? 0);
  }

  return (
    <styled.div width="full" display="flex" flexDirection="column" gap="4">
      {/* Main settings form */}
      <styled.form
        width="full"
        display="flex"
        flexDirection="column"
        gap="4"
        onSubmit={onSubmit}
      >
        <CardBox className={lstack()}>
          <WStack>
            <Heading size="md">Interface settings</Heading>
            <Button type="submit" loading={formState.isSubmitting}>
              Save
            </Button>
          </WStack>

          <FormControl>
            <FormLabel>Text editor style</FormLabel>
            <RadioGroupField
              control={control}
              name="editorMode"
              items={[
                { label: "Rich text", value: "richtext" },
                { label: "Markdown", value: "markdown" },
              ]}
            />
            <FormHelperText>
              Choose your preferred editor style for composing threads, replies
              and pages.
            </FormHelperText>
          </FormControl>

          <FormControl>
            <FormLabel>Sidebar default state</FormLabel>
            <RadioGroupField
              control={control}
              name="sidebarDefaultState"
              items={[
                { label: "Open", value: "open" },
                { label: "Closed", value: "closed" },
              ]}
            />
            <FormHelperText>
              Choose your preferred default state for the sidebar when you visit
              the site.
            </FormHelperText>
          </FormControl>
        </CardBox>
      </styled.form>

      {/* Light Mode Customization Suite – only active when in Light Mode */}
      {resolvedTheme === "light" && (
        <CardBox className={lstack()}>
          <WStack justifyContent="space-between" alignItems="center">
            <Heading size="md">Light Mode: Farbschema & Tönung</Heading>
            <span
              className={css({
                px: "2.5",
                py: "0.5",
                borderRadius: "full",
                bg: "bg.subtle",
                color: "fg.subtle",
                fontSize: "xs",
                fontWeight: "semibold",
              })}
            >
              Live Aktiv
            </span>
          </WStack>

          <FormHelperText>
            Wähle ein inspiriertes Farb-Thema für den Light Mode (z. B. Substack, Notion, Linear oder Zen Mode). Bilder und Medien bleiben dabei zu 100% farbecht und unverändert.
          </FormHelperText>

          {/* 1. Theme Presets */}
          <FormControl>
            <FormLabel>1. Farb-Themen (Presets)</FormLabel>
            <Grid gridTemplateColumns={{ base: "repeat(2, 1fr)", md: "repeat(3, 1fr)" }} gap="3" pt="1">
              {PRESET_LIST.map((preset) => {
                const isActive = lightPreset === preset.id;
                return (
                  <Box
                    key={preset.id}
                    onClick={() => setLightPreset(preset.id)}
                    p="3"
                    borderRadius="lg"
                    cursor="pointer"
                    position="relative"
                    style={{
                      backgroundColor: preset.previewBg,
                      border: isActive
                        ? "2px solid var(--colors-accent-default)"
                        : `1px solid ${preset.previewBorder}`,
                      boxShadow: isActive ? "0 0 0 1px var(--colors-accent-default)" : "none",
                      transition: "all 0.15s ease",
                    }}
                  >
                    <HStack gap="2.5" alignItems="center">
                      <Box
                        w="4"
                        h="4"
                        borderRadius="full"
                        style={{ backgroundColor: preset.previewDot }}
                      />
                      <Box>
                        <Box fontSize="sm" fontWeight="semibold" color="fg.default">
                          {preset.title}
                        </Box>
                        <Box fontSize="xs" color="fg.muted">
                          {preset.subtitle}
                        </Box>
                      </Box>
                    </HStack>
                  </Box>
                );
              })}
            </Grid>
          </FormControl>

          {/* 2. Background Nuances */}
          <FormControl pt="2">
            <FormLabel>2. Hintergrund-Nuance</FormLabel>
            <Grid gridTemplateColumns={{ base: "repeat(1, 1fr)", sm: "repeat(3, 1fr)" }} gap="2.5">
              {BG_STYLE_OPTIONS.map((styleOpt) => {
                const isActive = lightBgStyle === styleOpt.id;
                return (
                  <Box
                    key={styleOpt.id}
                    onClick={() => setLightBgStyle(styleOpt.id)}
                    p="2.5"
                    borderRadius="md"
                    cursor="pointer"
                    textAlign="center"
                    style={{
                      border: isActive
                        ? "2px solid var(--colors-accent-default)"
                        : "1px solid var(--colors-border-subtle)",
                      backgroundColor: isActive
                        ? "var(--colors-bg-subtle)"
                        : "transparent",
                      transition: "all 0.15s ease",
                    }}
                  >
                    <Box fontSize="xs" fontWeight="semibold" color="fg.default">
                      {styleOpt.label}
                    </Box>
                    <Box fontSize="2xs" color="fg.muted">
                      {styleOpt.desc}
                    </Box>
                  </Box>
                );
              })}
            </Grid>
          </FormControl>

          {/* 3. Farbwärme Slider */}
          <Box pt="2">
            <FormControl>
              <WStack alignItems="center" justifyContent="space-between" mb="1">
                <FormLabel mb="0">Farbwärme (Papier-Wärme)</FormLabel>
                <span
                  className={css({
                    px: "2.5",
                    py: "0.5",
                    borderRadius: "full",
                    bg: "bg.subtle",
                    fontSize: "xs",
                    fontWeight: "medium",
                    color: "fg.subtle",
                  })}
                >
                  {warmthLabel(warmth)} ({warmth}%)
                </span>
              </WStack>

              <Box
                w="full"
                h="2.5"
                borderRadius="full"
                mb="2"
                style={{
                  background:
                    "linear-gradient(to right, #f8f8f8 0%, #fdf5e6 40%, #fde8b0 70%, #f5c842 100%)",
                  border: "1px solid var(--colors-border-subtle)",
                }}
              />

              <Slider
                colorPalette="accent"
                min={0}
                max={100}
                step={1}
                value={[warmth]}
                onValueChange={handleWarmthChange}
                defaultValue={[0]}
              />
            </FormControl>
          </Box>

          {/* 4. Live Interactive Preview Card */}
          <Box pt="2">
            <FormLabel mb="2">Live-Vorschau</FormLabel>
            <Box
              p="4"
              borderRadius="xl"
              style={{
                backgroundColor: "var(--colors-bg-default)",
                border: "1px solid var(--colors-border-default)",
                boxShadow: "var(--shadows-sm)",
              }}
            >
              <WStack justifyContent="space-between" alignItems="center" mb="3">
                <Heading size="sm">Vorschau</Heading>
                <span
                  className={css({
                    px: "2",
                    py: "0.5",
                    borderRadius: "md",
                    bg: "bg.subtle",
                    fontSize: "xs",
                    color: "fg.muted",
                  })}
                >
                  {PRESET_CONFIGS[lightPreset]?.name ?? "Custom"}
                </span>
              </WStack>
              <Box fontSize="xs" color="fg.subtle" mb="3">
                Dies ist eine Vorschau, wie Beiträge, Karten und Schaltflächen mit deiner ausgewählten Farbkonfiguration im Light Mode dargestellt werden.
              </Box>
              <HStack gap="2">
                <Button size="sm" variant="solid">
                  Primärer Button
                </Button>
                <Button size="sm" variant="subtle">
                  Sekundär
                </Button>
              </HStack>
            </Box>
          </Box>
        </CardBox>
      )}
    </styled.div>
  );
}


